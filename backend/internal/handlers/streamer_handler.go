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

func (h *StreamerHandler) List(c *gin.Context) {
	claims := middleware.MustClaims(c)

	var (
		streamers []models.Streamer
		listErr   error
	)
	switch claims.Role {
	case models.RoleAdmin:
		streamers, listErr = h.streamerSvc.ListAll()
	case models.RoleAgency:
		agencyUserID, err := uuid.Parse(claims.UserID)
		if err != nil {
			badRequest(c, "invalid user id")
			return
		}
		streamers, listErr = h.streamerSvc.ListByAgencyUserID(agencyUserID)
	default:
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return
	}

	if listErr != nil {
		internal(c)
		return
	}

	ok(c, gin.H{"streamers": streamers})
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
		agencyUserID, err := uuid.Parse(claims.UserID)
		if err != nil {
			badRequest(c, "invalid user id")
			return
		}
		owns, err := h.streamerSvc.OwnsStreamer(agencyUserID, streamerID)
		if err != nil {
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
