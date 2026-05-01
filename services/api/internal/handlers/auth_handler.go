package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/v1/auth"
)

type AuthHandler struct {
	auth      *services.AuthService
	cfg       *config.Config
	emailAuth *services.EmailAuthService
}

func NewAuthHandler(auth *services.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{auth: auth, cfg: cfg}
}

// WithEmailAuth attaches an EmailAuthService so that a verification email is
// sent automatically after registration.
func (h *AuthHandler) WithEmailAuth(svc *services.EmailAuthService) *AuthHandler {
	h.emailAuth = svc
	return h
}

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body services.RegisterInput true "Registration payload"
// @Success      201  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      409  {object}  Response
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input services.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, tokens, err := h.auth.Register(input)
	if err != nil {
		switch err {
		case services.ErrEmailExists:
			conflict(c, "email already registered")
		case services.ErrUsernameExists:
			conflict(c, "username already taken")
		default:
			internal(c)
		}
		return
	}

	// Best-effort: send verification email if mailer is configured
	if h.emailAuth != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.emailAuth.SendVerificationEmail(ctx, user.ID); err != nil {
				log.Printf("auth register: send verification email failed user_id=%s err=%v", user.ID, err)
			}
		}()
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	created(c, gin.H{"user": user, "tokens": tokens})
}

// Login godoc
// @Summary      Login with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body services.LoginInput true "Login payload"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input services.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, tokens, err := h.auth.Login(input)
	if err != nil {
		unauthorized(c, "invalid email or password")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	ok(c, gin.H{"user": user, "tokens": tokens})
}

// Refresh godoc
// @Summary      Refresh access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{refresh_token=string} false "Refresh token fallback when cookie is unavailable"
// @Success      200  {object}  Response{data=TokensResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := h.refreshTokenFromRequest(c)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	tokens, err := h.auth.Refresh(refreshToken)
	if err != nil {
		unauthorized(c, "invalid or expired refresh token")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	ok(c, gin.H{"tokens": tokens})
}

// Logout godoc
// @Summary      Logout and invalidate refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{refresh_token=string} false "Refresh token fallback when cookie is unavailable"
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := h.refreshTokenFromRequest(c)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	h.auth.Logout(refreshToken)
	h.clearRefreshCookie(c)
	ok(c, gin.H{"message": "logged out"})
}

// TwitchLogin godoc
// @Summary      Redirect to Twitch OAuth
// @Tags         auth
// @Produce      json
// @Success      302
// @Router       /auth/twitch [get]
func (h *AuthHandler) TwitchLogin(c *gin.Context) {
	state := oauthState()
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	c.Redirect(http.StatusFound, h.auth.TwitchAuthURL(state))
}

// TwitchCallback godoc
// @Summary      Twitch OAuth callback
// @Tags         auth
// @Produce      json
// @Param        code  query string true "OAuth authorization code"
// @Param        state query string true "OAuth state"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /auth/twitch/callback [get]
func (h *AuthHandler) TwitchCallback(c *gin.Context) {
	if err := validateOAuthState(c); err != nil {
		badRequest(c, "invalid state parameter")
		return
	}

	code := c.Query("code")
	user, tokens, err := h.auth.TwitchCallback(c.Request.Context(), code)
	if err != nil {
		unauthorized(c, "twitch authentication failed")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	ok(c, gin.H{"user": user, "tokens": tokens})
}

// GoogleLogin godoc
// @Summary      Redirect to Google OAuth
// @Tags         auth
// @Produce      json
// @Success      302
// @Router       /auth/google [get]
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	state := oauthState()
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	c.Redirect(http.StatusFound, h.auth.GoogleAuthURL(state))
}

// GoogleCallback godoc
// @Summary      Google OAuth callback
// @Tags         auth
// @Produce      json
// @Param        code  query string true "OAuth authorization code"
// @Param        state query string true "OAuth state"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /auth/google/callback [get]
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	if err := validateOAuthState(c); err != nil {
		badRequest(c, "invalid state parameter")
		return
	}

	code := c.Query("code")
	user, tokens, err := h.auth.GoogleCallback(c.Request.Context(), code)
	if err != nil {
		unauthorized(c, "google authentication failed")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	ok(c, gin.H{"user": user, "tokens": tokens})
}

// Web3Nonce godoc
// @Summary      Get a sign-in nonce for a wallet address
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{address=string} true "Wallet address"
// @Success      200  {object}  Response{data=NonceResponse}
// @Failure      400  {object}  Response
// @Failure      500  {object}  Response
// @Router       /auth/web3/nonce [post]
func (h *AuthHandler) Web3Nonce(c *gin.Context) {
	var body struct {
		Address string `json:"address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	nonce, issuedAt, err := h.auth.Web3Nonce(body.Address)
	if err != nil {
		internal(c)
		return
	}

	ok(c, NonceResponse{
		Nonce:    nonce,
		IssuedAt: issuedAt.UTC().Format(time.RFC3339),
	})
}

// Web3Verify godoc
// @Summary      Verify wallet signature and authenticate
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body services.Web3VerifyInput true "Wallet address, nonce and signature"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /auth/web3/verify [post]
func (h *AuthHandler) Web3Verify(c *gin.Context) {
	var input services.Web3VerifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, tokens, err := h.auth.Web3Verify(input)
	if err != nil {
		switch err {
		case services.ErrInvalidNonce:
			unauthorized(c, "invalid or expired nonce")
		case services.ErrInvalidSignature:
			unauthorized(c, "invalid wallet signature")
		default:
			internal(c)
		}
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	ok(c, gin.H{"user": user, "tokens": tokens})
}

// UnlinkProvider godoc
// @Summary      Unlink an OAuth provider from the current user
// @Tags         auth
// @Produce      json
// @Param        provider path string true "Provider name" Enums(twitch, google, web3, email)
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Failure      500  {object}  Response
// @Security     BearerAuth
// @Router       /auth/providers/{provider} [delete]
func (h *AuthHandler) UnlinkProvider(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)
	provider := models.ProviderType(c.Param("provider"))

	if err := h.auth.UnlinkProvider(userID, provider); err != nil {
		switch err {
		case services.ErrLastProvider:
			badRequest(c, "cannot unlink the only login method")
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"message": "provider unlinked"})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func oauthState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func validateOAuthState(c *gin.Context) error {
	cookie, err := c.Cookie("oauth_state")
	if err != nil {
		return err
	}
	if cookie != c.Query("state") {
		return gin.Error{Err: nil, Type: gin.ErrorTypePublic}
	}
	return nil
}

func (h *AuthHandler) refreshTokenFromRequest(c *gin.Context) (string, error) {
	if token, err := c.Cookie(refreshTokenCookieName); err == nil && token != "" {
		return token, nil
	}

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == "" {
		return "", errors.New("refresh token is required")
	}
	return body.RefreshToken, nil
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, refreshToken string) {
	c.SetSameSite(h.refreshCookieSameSite(c))
	c.SetCookie(
		refreshTokenCookieName,
		refreshToken,
		h.refreshCookieMaxAge(),
		refreshTokenCookiePath,
		"",
		h.refreshCookieSecure(),
		true,
	)
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(h.refreshCookieSameSite(c))
	c.SetCookie(
		refreshTokenCookieName,
		"",
		-1,
		refreshTokenCookiePath,
		"",
		h.refreshCookieSecure(),
		true,
	)
}

func (h *AuthHandler) refreshCookieMaxAge() int {
	if h.cfg == nil || h.cfg.JWT.RefreshTTL <= 0 {
		return 0
	}
	return int(h.cfg.JWT.RefreshTTL.Seconds())
}

func (h *AuthHandler) refreshCookieSecure() bool {
	if h.cfg == nil {
		return false
	}
	env := strings.ToLower(h.cfg.Server.Env)
	return env == "production" || env == "prod"
}

func (h *AuthHandler) refreshCookieSameSite(c *gin.Context) http.SameSite {
	if !h.refreshCookieSecure() || h.cfg == nil || h.cfg.App.FrontendURL == "" {
		return http.SameSiteLaxMode
	}

	frontendSite, ok := schemefulSite(h.cfg.App.FrontendURL)
	if !ok {
		return http.SameSiteLaxMode
	}

	requestURL := &url.URL{
		Scheme: requestScheme(c.Request),
		Host:   requestHost(c.Request),
	}
	requestSite, ok := schemefulSite(requestURL.String())
	if !ok {
		return http.SameSiteLaxMode
	}

	if frontendSite != requestSite {
		return http.SameSiteNoneMode
	}

	return http.SameSiteLaxMode
}

func requestScheme(r *http.Request) string {
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		return strings.ToLower(proto)
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return strings.ToLower(r.URL.Scheme)
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func requestHost(r *http.Request) string {
	if host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); host != "" {
		return host
	}
	if host := strings.TrimSpace(r.Host); host != "" {
		return host
	}
	if r.URL != nil {
		return r.URL.Host
	}
	return ""
}

func schemefulSite(rawURL string) (string, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return "", false
	}

	host := strings.ToLower(parsed.Hostname())
	registrableDomain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err == nil {
		host = registrableDomain
	}

	return strings.ToLower(parsed.Scheme) + "://" + host, true
}
