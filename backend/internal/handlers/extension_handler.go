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
// @Summary      Complete a Bits transaction
// @Tags         extension
// @Accept       json
// @Produce      json
// @Param        body body object{extension_jwt=string,transaction_receipt=string,sku=string} true "Bits transaction payload"
// @Success      200  {object}  Response{data=AuthResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Router       /extension/bits/complete [post]
func (h *ExtensionHandler) BitsComplete(c *gin.Context) {
	var body struct {
		ExtensionJWT        string `json:"extension_jwt" binding:"required"`
		TransactionReceipt  string `json:"transaction_receipt" binding:"required"`
		SKU                 string `json:"sku" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, tokens, err := h.ext.CompleteBitsTransaction(body.ExtensionJWT, body.TransactionReceipt, body.SKU)
	if err != nil {
		switch err {
		case services.ErrInvalidExtJWT:
			unauthorized(c, "invalid extension JWT")
		case services.ErrInvalidReceipt:
			badRequest(c, "invalid transaction receipt")
		case services.ErrExtSecretMissing:
			internal(c)
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"user": user, "tokens": tokens})
}
