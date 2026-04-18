package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type UserHandler struct {
	user *services.UserService
}

func NewUserHandler(user *services.UserService) *UserHandler {
	return &UserHandler{user: user}
}

// Me godoc
// @Summary      Get current user profile
// @Tags         users
// @Produce      json
// @Success      200  {object}  Response{data=UserResponse}
// @Failure      404  {object}  Response
// @Security     BearerAuth
// @Router       /users/me [get]
func (h *UserHandler) Me(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	user, err := h.user.GetByID(userID)
	if err != nil {
		notFound(c, "user not found")
		return
	}

	ok(c, gin.H{"user": user})
}

// UpdateMe godoc
// @Summary      Update current user profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body services.UpdateProfileInput true "Profile fields to update"
// @Success      200  {object}  Response{data=UserResponse}
// @Failure      400  {object}  Response
// @Failure      409  {object}  Response
// @Security     BearerAuth
// @Router       /users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	var input services.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, err := h.user.UpdateProfile(userID, input)
	if err != nil {
		switch err {
		case services.ErrUsernameExists:
			conflict(c, "username already taken")
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"user": user})
}

// LinkWallet godoc
// @Summary      Bind a MetaMask wallet to the current user
// @Description  Verifies a SIWE signature and links the wallet address to the authenticated user.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body services.LinkWalletInput true "address, nonce, signature"
// @Success      200  {object}  Response{data=WalletResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Failure      409  {object}  Response
// @Security     BearerAuth
// @Router       /users/me/wallet [post]
func (h *UserHandler) LinkWallet(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	var input services.LinkWalletInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	addr, err := h.user.LinkWallet(userID, input)
	if err != nil {
		switch err {
		case services.ErrInvalidWalletAddress:
			badRequest(c, "invalid wallet address")
		case services.ErrInvalidNonce:
			unauthorized(c, "invalid or expired nonce")
		case services.ErrInvalidSignature:
			unauthorized(c, "invalid wallet signature")
		case services.ErrProviderLinked:
			conflict(c, "wallet already linked to another account")
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"address": addr})
}

// ListProviders godoc
// @Summary      List linked OAuth providers
// @Tags         users
// @Produce      json
// @Success      200  {object}  Response{data=ProvidersResponse}
// @Security     BearerAuth
// @Router       /users/me/providers [get]
func (h *UserHandler) ListProviders(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	providers, err := h.user.ListProviders(userID)
	if err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{"providers": providers})
}
