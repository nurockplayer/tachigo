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

	streamer, err := h.streamerSvc.RegisterContext(c.Request.Context(), userID, body.ChannelID, body.DisplayName)
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

	streamer, err := h.streamerSvc.CreateContext(c.Request.Context(), userID, agencyUserID, body.ChannelID)
	if err != nil {
		if errors.Is(err, services.ErrChannelNotOwned) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "channel_id does not match user's Twitch account"})
			return
		}
		if errors.Is(err, services.ErrAgencyUserInvalid) {
			badRequest(c, "agency_user_id must reference an agency user")
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

	channels, err := h.streamerSvc.ListChannelsContext(c.Request.Context(), userID)
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
		streamers, err = h.streamerSvc.ListAllContext(c.Request.Context())
	case models.RoleAgency:
		agencyUserID, parseErr := uuid.Parse(claims.UserID)
		if parseErr != nil {
			badRequest(c, "invalid user id")
			return
		}
		streamers, err = h.streamerSvc.ListByAgencyUserIDContext(c.Request.Context(), agencyUserID)
	default:
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return
	}

	if err != nil {
		internal(c)
		return
	}

	channelIDs := make([]string, 0, len(streamers))
	for _, s := range streamers {
		channelIDs = append(channelIDs, s.ChannelID)
	}

	summaryMap, err := h.streamerSvc.GetSummaryStatsContext(c.Request.Context(), channelIDs)
	if err != nil {
		internal(c)
		return
	}

	type streamerWithSummary struct {
		models.Streamer
		DailySeconds     int64 `json:"daily_seconds"`
		UniqueMiners     int64 `json:"unique_miners"`
		TotalTokenMinted int64 `json:"total_token_minted"`
	}

	items := make([]streamerWithSummary, 0, len(streamers))
	for _, s := range streamers {
		item := streamerWithSummary{Streamer: s}
		if sm, ok := summaryMap[s.ChannelID]; ok {
			item.DailySeconds = sm.DailySeconds
			item.UniqueMiners = sm.UniqueMiners
			item.TotalTokenMinted = sm.TotalTokenMinted
		}
		items = append(items, item)
	}

	ok(c, gin.H{"streamers": items})
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
		owns, err := h.streamerSvc.OwnsChannelContext(c.Request.Context(), userID, channelID)
		if err != nil {
			internal(c)
			return
		}
		if !owns {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
	}

	stats, err := h.streamerSvc.GetChannelStatsContext(c.Request.Context(), channelID)
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

	streamer, err := h.streamerSvc.GetByIDContext(c.Request.Context(), streamerID)
	if err != nil {
		if errors.Is(err, services.ErrStreamerNotFound) {
			notFound(c, "streamer not found")
			return
		}
		internal(c)
		return
	}

	claims := middleware.MustClaims(c)
	// Non-admin callers get 404 for both unknown and unauthorized streamer_ids
	// to prevent existence enumeration via 403 vs 404 distinction.
	switch claims.Role {
	case models.RoleStreamer:
		if claims.UserID != streamer.UserID.String() {
			notFound(c, "streamer not found")
			return
		}
	case models.RoleAgency:
		agencyUserID, parseErr := uuid.Parse(claims.UserID)
		if parseErr != nil {
			badRequest(c, "invalid user id")
			return
		}
		owns, ownErr := h.streamerSvc.OwnsStreamerContext(c.Request.Context(), agencyUserID, streamerID)
		if ownErr != nil {
			internal(c)
			return
		}
		if !owns {
			notFound(c, "streamer not found")
			return
		}
	case models.RoleAdmin:
	default:
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return
	}

	stats, err := h.streamerSvc.GetStatsContext(c.Request.Context(), streamer.ID)
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
