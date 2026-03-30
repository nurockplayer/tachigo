package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type WatchHandler struct {
	watchSvc *services.WatchService
}

func NewWatchHandler(watchSvc *services.WatchService) *WatchHandler {
	return &WatchHandler{watchSvc: watchSvc}
}

// StartSession handles POST /extension/watch/start
func (h *WatchHandler) StartSession(c *gin.Context) {
	claims := middleware.MustExtClaims(c)

	session, err := h.watchSvc.StartSession(claims.OpaqueUserID, claims.ChannelID)
	if err != nil {
		internal(c)
		return
	}
	ok(c, session)
}

// Heartbeat handles POST /extension/watch/heartbeat
func (h *WatchHandler) Heartbeat(c *gin.Context) {
	claims := middleware.MustExtClaims(c)

	result, err := h.watchSvc.Heartbeat(claims.OpaqueUserID, claims.ChannelID)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSession) {
			badRequest(c, "no active session")
			return
		}
		internal(c)
		return
	}
	ok(c, gin.H{
		"session":       result.Session,
		"points_earned": result.PointsEarned,
	})
}

// EndSession handles POST /extension/watch/end
func (h *WatchHandler) EndSession(c *gin.Context) {
	claims := middleware.MustExtClaims(c)

	if err := h.watchSvc.EndSession(claims.OpaqueUserID, claims.ChannelID); err != nil {
		internal(c)
		return
	}
	ok(c, gin.H{"ended": true})
}

// GetBalance handles GET /extension/watch/balance
func (h *WatchHandler) GetBalance(c *gin.Context) {
	claims := middleware.MustExtClaims(c)

	spendable, cumulative, err := h.watchSvc.GetBalance(claims.OpaqueUserID)
	if err != nil {
		internal(c)
		return
	}
	ok(c, gin.H{
		"spendable_balance": spendable,
		"cumulative_total":  cumulative,
	})
}
