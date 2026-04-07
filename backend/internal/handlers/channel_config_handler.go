package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type ChannelConfigHandler struct {
	configSvc   *services.ChannelConfigService
	streamerSvc *services.StreamerService
}

func NewChannelConfigHandler(configSvc *services.ChannelConfigService, streamerSvc *services.StreamerService) *ChannelConfigHandler {
	return &ChannelConfigHandler{configSvc: configSvc, streamerSvc: streamerSvc}
}

func (h *ChannelConfigHandler) GetChannelConfig(c *gin.Context) {
	channelID, allowed := h.authorizeChannelAccess(c)
	if !allowed {
		return
	}

	cfg, err := h.configSvc.Get(channelID)
	if err != nil {
		internal(c)
		return
	}
	if cfg == nil {
		cfg = &models.ChannelConfig{
			ChannelID:       channelID,
			SecondsPerPoint: services.DefaultSecondsPerPoint,
			Multiplier:      1,
		}
	}

	ok(c, gin.H{"config": cfg})
}

func (h *ChannelConfigHandler) UpdateChannelConfig(c *gin.Context) {
	var body struct {
		SecondsPerPoint int64 `json:"seconds_per_point" binding:"min=0"`
		Multiplier      int64 `json:"multiplier" binding:"min=0"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	if body.SecondsPerPoint == 0 && body.Multiplier == 0 {
		badRequest(c, "at least one field must be a positive value")
		return
	}

	channelID, allowed := h.authorizeChannelAccess(c)
	if !allowed {
		return
	}

	cfg, err := h.configSvc.UpdateChannelConfig(channelID, body.SecondsPerPoint, body.Multiplier)
	if err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{"config": cfg})
}

func (h *ChannelConfigHandler) authorizeChannelAccess(c *gin.Context) (string, bool) {
	channelID := c.Param("channel_id")
	if channelID == "" {
		badRequest(c, "channel_id is required")
		return "", false
	}

	claims := middleware.MustClaims(c)
	switch claims.Role {
	case models.RoleAdmin:
		return channelID, true
	case models.RoleAgency:
		c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "agency channel config not implemented"})
		return "", false
	case models.RoleStreamer:
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			badRequest(c, "invalid user id")
			return "", false
		}
		owns, err := h.streamerSvc.OwnsChannel(userID, channelID)
		if err != nil {
			internal(c)
			return "", false
		}
		if !owns {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return "", false
		}
		return channelID, true
	default:
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return "", false
	}
}
