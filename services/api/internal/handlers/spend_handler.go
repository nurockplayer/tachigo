package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type SpendHandler struct {
	spendSvc *services.SpendService
}

func NewSpendHandler(spendSvc *services.SpendService) *SpendHandler {
	return &SpendHandler{spendSvc: spendSvc}
}

type redeemRequest struct {
	CouponID string `json:"coupon_id" binding:"required"`
	Amount   int64  `json:"amount" binding:"required,min=1"`
}

type redeemResponse struct {
	Balance     int64  `json:"balance"`
	VoucherCode string `json:"voucher_code,omitempty"`
}

// Redeem godoc
// @Summary      Burn $TACHI to redeem a discount coupon
// @Tags         spend
// @Accept       json
// @Produce      json
// @Param        body body redeemRequest true "Amount to burn (must be > 0)"
// @Success      200 {object} Response{data=redeemResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      503 {object} Response
// @Failure      500 {object} Response
// @Security     BearerAuth
// @Router       /spend/redeem [post]
func (h *SpendHandler) Redeem(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	var req redeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid request body: "+err.Error())
		return
	}

	newBalance, voucherCode, err := h.spendSvc.Redeem(c.Request.Context(), userID, req.CouponID, req.Amount)
	if err != nil {
		if errors.Is(err, services.ErrSpendInsufficientBalance) {
			badRequest(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrSpendWalletNotLinked) {
			badRequest(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrSpendAmountInvalid) {
			badRequest(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrTachiyaRedeemFailed) {
			serviceUnavailable(c, "tachiya coupon redeem failed")
			return
		}
		// ErrSpendContractConfig is a server-side misconfiguration; intentionally falls through to internal(c).
		internal(c)
		return
	}

	ok(c, redeemResponse{Balance: newBalance, VoucherCode: voucherCode})
}
