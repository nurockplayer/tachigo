package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type WatchHandler struct {
	watchSvc  *services.WatchService
	pointsSvc *services.PointsService
}

func NewWatchHandler(watchSvc *services.WatchService, pointsSvc *services.PointsService) *WatchHandler {
	return &WatchHandler{watchSvc: watchSvc, pointsSvc: pointsSvc}
}

type watchBody struct {
	ChannelID string `json:"channel_id" binding:"required"`
}

// StartSession handles POST /extension/watch/start
func (h *WatchHandler) StartSession(c *gin.Context) {
	claims := middleware.MustClaims(c)
	var body watchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	session, err := h.watchSvc.StartSession(userID, body.ChannelID)
	if err != nil {
		internal(c)
		return
	}
	ok(c, session)
}

// Heartbeat handles POST /extension/watch/heartbeat
func (h *WatchHandler) Heartbeat(c *gin.Context) {
	claims := middleware.MustClaims(c)
	var body watchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	result, err := h.watchSvc.Heartbeat(userID, body.ChannelID)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSession) {
			badRequest(c, "no active session")
			return
		}
		internal(c)
		return
	}

	// Accumulate time stats after a valid heartbeat (non-fatal: don't fail the heartbeat if these fail).
	if result.DeltaSeconds > 0 {
		_ = h.pointsSvc.AddWatchTime(userID, body.ChannelID, result.DeltaSeconds)
		_ = h.pointsSvc.AddBroadcastTime(body.ChannelID, result.DeltaSeconds)
	}

	ok(c, gin.H{
		"session":       result.Session,
		"points_earned": result.PointsEarned,
	})
}

// EndSession handles POST /extension/watch/end
func (h *WatchHandler) EndSession(c *gin.Context) {
	claims := middleware.MustClaims(c)
	var body watchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	if err := h.watchSvc.EndSession(userID, body.ChannelID); err != nil {
		internal(c)
		return
	}
	ok(c, gin.H{"ended": true})
}

// GetBalance handles GET /extension/watch/balance?channel_id=...
func (h *WatchHandler) GetBalance(c *gin.Context) {
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

	spendable, cumulative, err := h.watchSvc.GetBalance(userID, channelID)
	if err != nil {
		internal(c)
		return
	}
	ok(c, gin.H{
		"spendable_balance": spendable,
		"cumulative_total":  cumulative,
	})
}
