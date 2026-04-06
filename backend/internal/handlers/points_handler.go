package handlers

import (
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
