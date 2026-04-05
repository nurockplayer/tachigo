package handlers

import (
	"time"

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

type pointsHistoryItem struct {
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	Note      *string   `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type pointsHistoryResponse struct {
	Transactions []pointsHistoryItem `json:"transactions"`
}

// GetBalance handles GET /api/v1/users/me/points?channel_id=...
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
	ok(c, balance)
}

// GetHistory handles GET /api/v1/users/me/points/history?channel_id=...
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

	items := make([]pointsHistoryItem, 0, len(txs))
	for _, tx := range txs {
		txType := "earn"
		amount := tx.Delta
		if tx.Delta < 0 {
			txType = "spend"
			amount = -tx.Delta
		}
		items = append(items, pointsHistoryItem{
			Type:      txType,
			Amount:    amount,
			Note:      tx.Note,
			CreatedAt: tx.CreatedAt,
		})
	}

	ok(c, pointsHistoryResponse{Transactions: items})
}
