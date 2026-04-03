package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/services"
)

type ChannelConfigHandler struct {
	configSvc *services.ChannelConfigService
}

func NewChannelConfigHandler(configSvc *services.ChannelConfigService) *ChannelConfigHandler {
	return &ChannelConfigHandler{configSvc: configSvc}
}

func (h *ChannelConfigHandler) UpdateChannelConfig(c *gin.Context) {
	var body struct {
		SecondsPerPoint int64 `json:"seconds_per_point" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	channelID := c.Param("channel_id")
	if channelID == "" {
		badRequest(c, "channel_id is required")
		return
	}

	cfg, err := h.configSvc.UpdateChannelConfig(channelID, body.SecondsPerPoint)
	if err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{"config": cfg})
}
