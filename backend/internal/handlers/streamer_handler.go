package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type StreamerHandler struct {
	streamerSvc *services.StreamerService
}

func NewStreamerHandler(svc *services.StreamerService) *StreamerHandler {
	return &StreamerHandler{streamerSvc: svc}
}

func (h *StreamerHandler) Register(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	var body struct {
		ChannelID   string `json:"channel_id" binding:"required"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	streamer, err := h.streamerSvc.Register(userID, body.ChannelID, body.DisplayName)
	if err != nil {
		if errors.Is(err, services.ErrChannelNotOwned) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "channel_id does not match your Twitch account"})
			return
		}
		internal(c)
		return
	}

	ok(c, gin.H{"streamer": streamer})
}

func (h *StreamerHandler) Create(c *gin.Context) {
	var body struct {
		UserID       string  `json:"user_id" binding:"required"`
		AgencyUserID *string `json:"agency_user_id"`
		ChannelID    string  `json:"channel_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		badRequest(c, "invalid user_id")
		return
	}

	var agencyUserID *uuid.UUID
	if body.AgencyUserID != nil {
		aid, err := uuid.Parse(*body.AgencyUserID)
		if err != nil {
			badRequest(c, "invalid agency_user_id")
			return
		}
		agencyUserID = &aid
	}

	streamer, err := h.streamerSvc.Create(userID, agencyUserID, body.ChannelID)
	if err != nil {
		if errors.Is(err, services.ErrChannelNotOwned) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "channel_id does not match user's Twitch account"})
			return
		}
		internal(c)
		return
	}

	ok(c, gin.H{"streamer": streamer})
}

func (h *StreamerHandler) ListChannels(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	channels, err := h.streamerSvc.ListChannels(userID)
	if err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{"channels": channels})
}

func (h *StreamerHandler) List(c *gin.Context) {
	claims := middleware.MustClaims(c)

	var (
		streamers []models.Streamer
		err       error
	)
	switch claims.Role {
	case models.RoleAdmin:
		streamers, err = h.streamerSvc.ListAll()
	case models.RoleAgency:
		agencyUserID, parseErr := uuid.Parse(claims.UserID)
		if parseErr != nil {
			badRequest(c, "invalid user id")
			return
		}
		streamers, err = h.streamerSvc.ListByAgencyUserID(agencyUserID)
	default:
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return
	}

	if err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{"streamers": streamers})
}

func (h *StreamerHandler) GetChannelStats(c *gin.Context) {
	channelID := c.Param("channel_id")
	if channelID == "" {
		badRequest(c, "channel_id is required")
		return
	}

	claims := middleware.MustClaims(c)
	if claims.Role != models.RoleAdmin {
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			badRequest(c, "invalid user id")
			return
		}
		owns, err := h.streamerSvc.OwnsChannel(userID, channelID)
		if err != nil {
			internal(c)
			return
		}
		if !owns {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
	}

	stats, err := h.streamerSvc.GetChannelStats(channelID)
	if err != nil {
		if errors.Is(err, services.ErrStreamerNotFound) {
			notFound(c, "streamer not found")
			return
		}
		internal(c)
		return
	}

	ok(c, gin.H{"stats": stats})
}

func (h *StreamerHandler) GetStats(c *gin.Context) {
	streamerID, err := uuid.Parse(c.Param("streamer_id"))
	if err != nil {
		badRequest(c, "invalid streamer_id")
		return
	}

	streamer, err := h.streamerSvc.GetByID(streamerID)
	if err != nil {
		if errors.Is(err, services.ErrStreamerNotFound) {
			notFound(c, "streamer not found")
			return
		}
		internal(c)
		return
	}

	claims := middleware.MustClaims(c)
	switch claims.Role {
	case models.RoleStreamer:
		if claims.UserID != streamer.UserID.String() {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
	case models.RoleAgency:
		agencyUserID, parseErr := uuid.Parse(claims.UserID)
		if parseErr != nil {
			badRequest(c, "invalid user id")
			return
		}
		owns, ownErr := h.streamerSvc.OwnsStreamer(agencyUserID, streamerID)
		if ownErr != nil {
			internal(c)
			return
		}
		if !owns {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
	case models.RoleAdmin:
	default:
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return
	}

	stats, err := h.streamerSvc.GetStats(streamer.UserID)
	if err != nil {
		if errors.Is(err, services.ErrStreamerNotFound) {
			notFound(c, "streamer not found")
			return
		}
		internal(c)
		return
	}

	ok(c, gin.H{"stats": stats, "channel_id": streamer.ChannelID})
}
