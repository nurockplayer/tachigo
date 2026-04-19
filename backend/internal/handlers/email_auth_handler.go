package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type EmailAuthHandler struct {
	emailAuth *services.EmailAuthService
}

func NewEmailAuthHandler(emailAuth *services.EmailAuthService) *EmailAuthHandler {
	return &EmailAuthHandler{emailAuth: emailAuth}
}

// SendVerification godoc
// @Summary      Send or resend email verification
// @Tags         auth
// @Produce      json
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Failure      404  {object}  Response
// @Security     BearerAuth
// @Router       /auth/verify-email/send [post]
func (h *EmailAuthHandler) SendVerification(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	if err := h.emailAuth.SendVerificationEmail(userID); err != nil {
		switch err {
		case services.ErrAlreadyVerified:
			badRequest(c, "email already verified")
		case services.ErrUserNotFound:
			notFound(c, "user not found")
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"message": "verification email sent"})
}

// ConfirmVerification godoc
// @Summary      Confirm email verification token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{token=string} true "Verification token"
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Router       /auth/verify-email/confirm [post]
func (h *EmailAuthHandler) ConfirmVerification(c *gin.Context) {
	var body struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.emailAuth.VerifyEmail(body.Token); err != nil {
		switch err {
		case services.ErrInvalidVerifyToken:
			badRequest(c, "invalid or expired verification token")
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"message": "email verified"})
}

// ForgotPassword godoc
// @Summary      Request a password reset email
// @Description  Always returns 200 to prevent user enumeration
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{email=string} true "Email address"
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Router       /auth/forgot-password [post]
func (h *EmailAuthHandler) ForgotPassword(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	// Best-effort: ignore errors so we don't reveal whether the email exists
	h.emailAuth.ForgotPassword(body.Email)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "if that email is registered, a reset link has been sent"},
	})
}

// ResetPassword godoc
// @Summary      Reset password using token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body object{token=string,new_password=string} true "Token and new password (min 8 chars)"
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Router       /auth/reset-password [post]
func (h *EmailAuthHandler) ResetPassword(c *gin.Context) {
	var body struct {
		Token       string `json:"token"        binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.emailAuth.ResetPassword(body.Token, body.NewPassword); err != nil {
		switch err {
		case services.ErrInvalidResetToken:
			badRequest(c, "invalid or expired reset token")
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"message": "password reset successfully"})
}
