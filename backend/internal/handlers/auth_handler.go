package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type AuthHandler struct {
	auth      *services.AuthService
	emailAuth *services.EmailAuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
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
// @Security
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
		go h.emailAuth.SendVerificationEmail(user.ID)
	}

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
// @Security
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

	ok(c, gin.H{"user": user, "tokens": tokens})
}

// Refresh godoc
// @Summary      Refresh access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{refresh_token=string} true "Refresh token"
// @Success      200  {object}  Response{data=TokensResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Security
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	tokens, err := h.auth.Refresh(body.RefreshToken)
	if err != nil {
		unauthorized(c, "invalid or expired refresh token")
		return
	}

	ok(c, gin.H{"tokens": tokens})
}

// Logout godoc
// @Summary      Logout and invalidate refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{refresh_token=string} true "Refresh token"
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Security
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	h.auth.Logout(body.RefreshToken)
	ok(c, gin.H{"message": "logged out"})
}

// TwitchLogin godoc
// @Summary      Redirect to Twitch OAuth
// @Tags         auth
// @Produce      json
// @Success      302
// @Security
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
// @Security
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

	ok(c, gin.H{"user": user, "tokens": tokens})
}

// GoogleLogin godoc
// @Summary      Redirect to Google OAuth
// @Tags         auth
// @Produce      json
// @Success      302
// @Security
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
// @Security
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
// @Security
// @Router       /auth/web3/nonce [post]
func (h *AuthHandler) Web3Nonce(c *gin.Context) {
	var body struct {
		Address string `json:"address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	nonce, err := h.auth.Web3Nonce(body.Address)
	if err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{"nonce": nonce})
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
// @Security
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
