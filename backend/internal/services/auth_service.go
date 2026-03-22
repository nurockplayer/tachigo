package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	googleOAuth "golang.org/x/oauth2/google"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already registered")
	ErrUsernameExists     = errors.New("username already taken")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrProviderLinked     = errors.New("provider already linked to another account")
	ErrInvalidNonce       = errors.New("invalid or expired nonce")
	ErrInvalidSignature   = errors.New("invalid wallet signature")
	ErrLastProvider       = errors.New("cannot unlink the only login method")
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

type Claims struct {
	UserID string          `json:"uid"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	db           *gorm.DB
	cfg          *config.Config
	twitchOAuth  *oauth2.Config
	googleOAuth  *oauth2.Config
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	twitchCfg := &oauth2.Config{
		ClientID:     cfg.OAuth.Twitch.ClientID,
		ClientSecret: cfg.OAuth.Twitch.ClientSecret,
		RedirectURL:  cfg.OAuth.Twitch.RedirectURL,
		Scopes:       []string{"user:read:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://id.twitch.tv/oauth2/authorize",
			TokenURL: "https://id.twitch.tv/oauth2/token",
		},
	}
	googleCfg := &oauth2.Config{
		ClientID:     cfg.OAuth.Google.ClientID,
		ClientSecret: cfg.OAuth.Google.ClientSecret,
		RedirectURL:  cfg.OAuth.Google.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     googleOAuth.Endpoint,
	}
	return &AuthService{db: db, cfg: cfg, twitchOAuth: twitchCfg, googleOAuth: googleCfg}
}

// ─── Email / Password ────────────────────────────────────────────────────────

type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (s *AuthService) Register(input RegisterInput) (*models.User, *TokenPair, error) {
	// Check uniqueness
	var count int64
	s.db.Model(&models.User{}).Where("email = ?", input.Email).Count(&count)
	if count > 0 {
		return nil, nil, ErrEmailExists
	}
	s.db.Model(&models.User{}).Where("username = ?", input.Username).Count(&count)
	if count > 0 {
		return nil, nil, ErrUsernameExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	email := input.Email
	username := input.Username
	hashStr := string(hash)

	user := &models.User{
		Email:        &email,
		Username:     &username,
		PasswordHash: &hashStr,
		Role:         models.RoleViewer,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, nil, err
	}

	// Also create an email AuthProvider record for consistency
	s.db.Create(&models.AuthProvider{
		UserID:     user.ID,
		Provider:   models.ProviderEmail,
		ProviderID: input.Email,
	})

	tokens, err := s.issueTokenPair(user)
	return user, tokens, err
}

type LoginInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (s *AuthService) Login(input LoginInput) (*models.User, *TokenPair, error) {
	var user models.User
	if err := s.db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	if user.PasswordHash == nil {
		return nil, nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.issueTokenPair(&user)
	return &user, tokens, err
}

// ─── Token Refresh / Logout ──────────────────────────────────────────────────

func (s *AuthService) Refresh(rawRefreshToken string) (*TokenPair, error) {
	hash := hashToken(rawRefreshToken)

	var stored models.RefreshToken
	if err := s.db.Where("token_hash = ?", hash).First(&stored).Error; err != nil {
		return nil, ErrInvalidToken
	}
	if stored.IsExpired() {
		s.db.Delete(&stored)
		return nil, ErrInvalidToken
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", stored.UserID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	// Rotate: delete old, issue new
	s.db.Delete(&stored)
	return s.issueTokenPair(&user)
}

func (s *AuthService) Logout(rawRefreshToken string) error {
	hash := hashToken(rawRefreshToken)
	return s.db.Where("token_hash = ?", hash).Delete(&models.RefreshToken{}).Error
}

// ─── Twitch OAuth ────────────────────────────────────────────────────────────

func (s *AuthService) TwitchAuthURL(state string) string {
	return s.twitchOAuth.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

type TwitchUserInfo struct {
	ID          string  `json:"id"`
	Login       string  `json:"login"`
	DisplayName string  `json:"display_name"`
	Email       string  `json:"email"`
	ProfileURL  *string `json:"profile_image_url"`
}

func (s *AuthService) TwitchCallback(ctx context.Context, code string) (*models.User, *TokenPair, error) {
	token, err := s.twitchOAuth.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("twitch token exchange: %w", err)
	}

	info, err := fetchTwitchUser(ctx, s.twitchOAuth, token, s.cfg.OAuth.Twitch.ClientID)
	if err != nil {
		return nil, nil, err
	}

	return s.upsertOAuthUser(ctx, models.ProviderTwitch, info.ID, info.Login, info.Email, info.ProfileURL, token)
}

// ─── Google OAuth ────────────────────────────────────────────────────────────

func (s *AuthService) GoogleAuthURL(state string) string {
	return s.googleOAuth.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (s *AuthService) GoogleCallback(ctx context.Context, code string) (*models.User, *TokenPair, error) {
	token, err := s.googleOAuth.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("google token exchange: %w", err)
	}

	info, err := fetchGoogleUser(ctx, s.googleOAuth, token)
	if err != nil {
		return nil, nil, err
	}

	return s.upsertOAuthUser(ctx, models.ProviderGoogle, info.Sub, info.Name, info.Email, &info.Picture, token)
}

// ─── Web3 / SIWE ─────────────────────────────────────────────────────────────

func (s *AuthService) Web3Nonce(address string) (string, error) {
	address = strings.ToLower(address)
	nonce, err := generateNonce()
	if err != nil {
		return "", err
	}

	// Delete any existing nonces for this address
	s.db.Where("address = ?", address).Delete(&models.Web3Nonce{})

	record := &models.Web3Nonce{
		Nonce:     nonce,
		Address:   address,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.db.Create(record).Error; err != nil {
		return "", err
	}
	return nonce, nil
}

type Web3VerifyInput struct {
	Address   string `json:"address"   binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Nonce     string `json:"nonce"     binding:"required"`
}

func (s *AuthService) Web3Verify(input Web3VerifyInput) (*models.User, *TokenPair, error) {
	address := strings.ToLower(input.Address)

	var nonceRecord models.Web3Nonce
	if err := s.db.Where("nonce = ? AND address = ?", input.Nonce, address).First(&nonceRecord).Error; err != nil {
		return nil, nil, ErrInvalidNonce
	}
	if nonceRecord.IsExpired() {
		s.db.Delete(&nonceRecord)
		return nil, nil, ErrInvalidNonce
	}

	// Verify signature
	msg := siweMessage(address, input.Nonce)
	if !verifyEthSignature(msg, input.Signature, address) {
		return nil, nil, ErrInvalidSignature
	}

	// Nonce is consumed
	s.db.Delete(&nonceRecord)

	// Upsert user
	checksumAddr := common.HexToAddress(input.Address).Hex()
	return s.upsertOAuthUser(context.Background(), models.ProviderWeb3, checksumAddr, "", "", nil, nil)
}

// ─── Provider Link (attach to existing account) ──────────────────────────────

func (s *AuthService) LinkTwitch(ctx context.Context, userID uuid.UUID, code string) error {
	token, err := s.twitchOAuth.Exchange(ctx, code)
	if err != nil {
		return err
	}
	info, err := fetchTwitchUser(ctx, s.twitchOAuth, token, s.cfg.OAuth.Twitch.ClientID)
	if err != nil {
		return err
	}
	return s.linkProvider(userID, models.ProviderTwitch, info.ID, token)
}

func (s *AuthService) UnlinkProvider(userID uuid.UUID, provider models.ProviderType) error {
	// Ensure the user still has at least one other way to log in
	var count int64
	s.db.Model(&models.AuthProvider{}).Where("user_id = ?", userID).Count(&count)

	var user models.User
	s.db.First(&user, "id = ?", userID)
	hasPassword := user.PasswordHash != nil

	if count <= 1 && !hasPassword {
		return ErrLastProvider
	}

	return s.db.Where("user_id = ? AND provider = ?", userID, provider).Delete(&models.AuthProvider{}).Error
}

// ─── JWT helpers ─────────────────────────────────────────────────────────────

func (s *AuthService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWT.AccessSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) issueTokenPair(user *models.User) (*TokenPair, error) {
	// Access token
	accessClaims := Claims{
		UserID: user.ID.String(),
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWT.AccessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWT.AccessSecret))
	if err != nil {
		return nil, err
	}

	// Refresh token – random opaque token stored hashed in DB
	rawRefresh, err := generateNonce()
	if err != nil {
		return nil, err
	}
	s.db.Create(&models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshTTL),
	})

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(s.cfg.JWT.AccessTTL.Seconds()),
	}, nil
}

// ─── OAuth upsert helper ─────────────────────────────────────────────────────

type googleUserInfo struct {
	Sub     string `json:"sub"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

func (s *AuthService) upsertOAuthUser(
	_ context.Context,
	provider models.ProviderType,
	providerID, username, email string,
	avatarURL *string,
	token *oauth2.Token,
) (*models.User, *TokenPair, error) {

	var ap models.AuthProvider
	err := s.db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&ap).Error

	if err == nil {
		// Already linked – update tokens, return user
		if token != nil {
			at := token.AccessToken
			rt := token.RefreshToken
			ap.AccessToken = &at
			ap.RefreshToken = &rt
			ap.TokenExpiresAt = &token.Expiry
			s.db.Save(&ap)
		}
		var user models.User
		if err := s.db.First(&user, "id = ?", ap.UserID).Error; err != nil {
			return nil, nil, ErrUserNotFound
		}
		tokens, err := s.issueTokenPair(&user)
		return &user, tokens, err
	}

	// New provider – find or create user
	var user models.User

	if email != "" {
		s.db.Where("email = ?", email).First(&user)
	}

	if user.ID == uuid.Nil {
		// Brand-new user
		if email != "" {
			user.Email = &email
		}
		if username != "" {
			user.Username = &username
		}
		user.AvatarURL = avatarURL
		user.Role = models.RoleViewer
		if err := s.db.Create(&user).Error; err != nil {
			return nil, nil, err
		}
	}

	// Link provider
	newAP := models.AuthProvider{
		UserID:     user.ID,
		Provider:   provider,
		ProviderID: providerID,
	}
	if token != nil {
		at := token.AccessToken
		rt := token.RefreshToken
		newAP.AccessToken = &at
		newAP.RefreshToken = &rt
		newAP.TokenExpiresAt = &token.Expiry
	}
	s.db.Create(&newAP)

	tokens, err := s.issueTokenPair(&user)
	return &user, tokens, err
}

func (s *AuthService) linkProvider(userID uuid.UUID, provider models.ProviderType, providerID string, token *oauth2.Token) error {
	// Check not already linked to someone else
	var count int64
	s.db.Model(&models.AuthProvider{}).
		Where("provider = ? AND provider_id = ? AND user_id != ?", provider, providerID, userID).
		Count(&count)
	if count > 0 {
		return ErrProviderLinked
	}

	ap := models.AuthProvider{
		UserID:     userID,
		Provider:   provider,
		ProviderID: providerID,
	}
	if token != nil {
		at := token.AccessToken
		rt := token.RefreshToken
		ap.AccessToken = &at
		ap.RefreshToken = &rt
		ap.TokenExpiresAt = &token.Expiry
	}
	return s.db.Save(&ap).Error
}

// ─── SIWE / crypto helpers ───────────────────────────────────────────────────

func siweMessage(address, nonce string) string {
	return fmt.Sprintf(
		"tachigo.io wants you to sign in with your Ethereum account:\n%s\n\nSign in to Tachigo\n\nNonce: %s\nIssued At: %s",
		address, nonce, time.Now().UTC().Format(time.RFC3339),
	)
}

func verifyEthSignature(message, sigHex, expectedAddress string) bool {
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil || len(sigBytes) != 65 {
		return false
	}
	// Ethereum adds "\x19Ethereum Signed Message:\n" prefix
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixed))

	// Adjust v (last byte)
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false
	}
	recovered := strings.ToLower(crypto.PubkeyToAddress(*pubKey).Hex())
	return recovered == strings.ToLower(expectedAddress)
}

// ─── Utility ─────────────────────────────────────────────────────────────────

func generateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
