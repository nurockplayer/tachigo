package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/services"
)

type ExtensionHandler struct {
	ext *services.ExtensionService
}

func NewExtensionHandler(ext *services.ExtensionService) *ExtensionHandler {
	return &ExtensionHandler{ext: ext}
}

// Login godoc
// @Summary      Authenticate via Twitch Extension JWT
// @Tags         extension
// @Accept       json
// @Produce      json
// @Param        body body object{extension_jwt=string} true "Extension JWT from Twitch"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /extension/auth/login [post]
func (h *ExtensionHandler) Login(c *gin.Context) {
	var body struct {
		ExtensionJWT string `json:"extension_jwt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, tokens, err := h.ext.LoginWithExtension(body.ExtensionJWT)
	if err != nil {
		switch err {
		case services.ErrInvalidExtJWT:
			unauthorized(c, "invalid extension JWT")
		case services.ErrUserNotFound:
			unauthorized(c, "tachigo account not found — please sign up at tachigo and link your Twitch account")
		case services.ErrExtSecretMissing:
			internal(c)
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"user": user, "tokens": tokens})
}

// TPointComplete godoc
// @Summary      Complete a T-point transaction
// @Description  Verifies Twitch Extension JWT and transaction receipt, then awards T-points to the viewer.
// @Tags         extension
// @Accept       json
// @Produce      json
// @Param        body body object{extension_jwt=string,transaction_receipt=string,sku=string} true "T-point transaction payload"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /extension/t-point/complete [post]
func (h *ExtensionHandler) TPointComplete(c *gin.Context) {
	var body struct {
		ExtensionJWT       string `json:"extension_jwt" binding:"required"`
		TransactionReceipt string `json:"transaction_receipt" binding:"required"`
		SKU                string `json:"sku" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, tokens, err := h.ext.CompleteTPointTransaction(body.ExtensionJWT, body.TransactionReceipt, body.SKU)
	if err != nil {
		switch err {
		case services.ErrInvalidExtJWT:
			unauthorized(c, "invalid extension JWT")
		case services.ErrInvalidReceipt, services.ErrInvalidReceiptAmount, services.ErrInvalidReceiptType:
			badRequest(c, "invalid transaction receipt")
		case services.ErrDuplicateTransaction:
			conflict(c, "transaction already processed")
		case services.ErrExtSecretMissing:
			internal(c)
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"user": user, "tokens": tokens})
}

// BitsComplete godoc
// @Summary      [Deprecated] Complete a Bits transaction
// @Description  Deprecated alias for /extension/t-point/complete. Use the new endpoint instead.
// @Tags         extension
// @Accept       json
// @Produce      json
// @Param        body body object{extension_jwt=string,transaction_receipt=string,sku=string} true "Bits transaction payload"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /extension/bits/complete [post]
// @Deprecated
func (h *ExtensionHandler) BitsComplete(c *gin.Context) { h.TPointComplete(c) }
