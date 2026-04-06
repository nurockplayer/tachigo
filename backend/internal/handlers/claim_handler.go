package handlers

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type ClaimHandler struct {
	claimSvc *services.ClaimService
}

func NewClaimHandler(claimSvc *services.ClaimService) *ClaimHandler {
	return &ClaimHandler{claimSvc: claimSvc}
}

type claimRequest struct {
	Amount int64 `json:"amount"` // 0 = claim all
}

type tachiBalanceResponse struct {
	TachiBalance int64 `json:"tachi_balance"`
}

// Claim godoc
// @Summary      Claim T-Points as $TACHI
// @Tags         tachi
// @Accept       json
// @Produce      json
// @Param        body body claimRequest false "Amount to claim (0 = all)"
// @Success      200 {object} Response{data=tachiBalanceResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      422 {object} Response
// @Security     BearerAuth
// @Router       /users/me/points/claim [post]
func (h *ClaimHandler) Claim(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	var req claimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Only a truly empty body (EOF) is treated as "claim all".
		// Malformed JSON or type errors must be rejected to avoid silently
		// draining the user's entire balance on a bad request.
		if !errors.Is(err, io.EOF) {
			badRequest(c, "invalid request body: "+err.Error())
			return
		}
		req.Amount = 0
	}
	if req.Amount < 0 {
		badRequest(c, "amount must be >= 0")
		return
	}

	newBalance, err := h.claimSvc.Claim(userID, req.Amount)
	if err != nil {
		if errors.Is(err, services.ErrClaimInsufficientBalance) {
			c.JSON(422, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, services.ErrClaimAmountInvalid) {
			badRequest(c, err.Error())
			return
		}
		internal(c)
		return
	}

	ok(c, tachiBalanceResponse{TachiBalance: newBalance})
}

// GetTachiBalance godoc
// @Summary      Get my $TACHI balance
// @Tags         tachi
// @Produce      json
// @Success      200 {object} Response{data=tachiBalanceResponse}
// @Failure      401 {object} Response
// @Security     BearerAuth
// @Router       /users/me/tachi/balance [get]
func (h *ClaimHandler) GetTachiBalance(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	balance, err := h.claimSvc.GetTachiBalance(userID)
	if err != nil {
		internal(c)
		return
	}

	ok(c, tachiBalanceResponse{TachiBalance: balance})
}
