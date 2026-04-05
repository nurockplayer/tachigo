package handlers

import (
	"math"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type PointsHandler struct {
	pointsSvc *services.PointsService
}

func NewPointsHandler(pointsSvc *services.PointsService) *PointsHandler {
	return &PointsHandler{pointsSvc: pointsSvc}
}

// GetBalance godoc
// @Summary      Get my points balance
// @Tags         points
// @Produce      json
// @Param        channel_id query string true "Twitch channel ID"
// @Success      200 {object} Response{data=PointsBalanceResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Security     BearerAuth
// @Router       /users/me/points [get]
func (h *PointsHandler) GetBalance(c *gin.Context) {
	claims := middleware.MustClaims(c)
	channelID := c.Query("channel_id")
	if channelID == "" {
		badRequest(c, "channel_id is required")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	balance, err := h.pointsSvc.GetBalance(userID, channelID)
	if err != nil {
		internal(c)
		return
	}

	ok(c, PointsBalanceResponse{
		CumulativeTotal:  balance.CumulativeTotal,
		SpendableBalance: balance.SpendableBalance,
	})
}

// GetHistory godoc
// @Summary      List my recent points transactions
// @Tags         points
// @Produce      json
// @Param        channel_id query string true "Twitch channel ID"
// @Success      200 {object} Response{data=PointsHistoryResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Security     BearerAuth
// @Router       /users/me/points/history [get]
func (h *PointsHandler) GetHistory(c *gin.Context) {
	claims := middleware.MustClaims(c)
	channelID := c.Query("channel_id")
	if channelID == "" {
		badRequest(c, "channel_id is required")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	txs, err := h.pointsSvc.ListTransactions(userID, channelID)
	if err != nil {
		internal(c)
		return
	}

	items := make([]PointsHistoryItem, 0, len(txs))
	for _, tx := range txs {
		txType := "earn"
		amount := tx.Delta
		if tx.Delta < 0 {
			if tx.Delta == math.MinInt64 {
				internal(c)
				return
			}
			txType = "spend"
			amount = -tx.Delta
		}
		items = append(items, PointsHistoryItem{
			Type:      txType,
			Amount:    amount,
			SKU:       tx.SKU,
			Note:      tx.Note,
			CreatedAt: tx.CreatedAt,
		})
	}

	ok(c, PointsHistoryResponse{Transactions: items})
}
