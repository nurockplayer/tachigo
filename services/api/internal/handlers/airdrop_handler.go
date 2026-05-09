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

type AirdropHandler struct {
	airdropSvc  *services.AirdropService
	agencySvc   *services.AgencyService
	streamerSvc *services.StreamerService
}

func NewAirdropHandler(
	airdropSvc *services.AirdropService,
	agencySvc *services.AgencyService,
	streamerSvc *services.StreamerService,
) *AirdropHandler {
	return &AirdropHandler{
		airdropSvc:  airdropSvc,
		agencySvc:   agencySvc,
		streamerSvc: streamerSvc,
	}
}

func (h *AirdropHandler) Airdrop(c *gin.Context) {
	var body struct {
		Amount int64  `json:"amount" binding:"required,min=1"`
		Note   string `json:"note"`
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

	claims := middleware.MustClaims(c)
	allowed, err := h.authorize(claims, channelID)
	if err != nil {
		internal(c)
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return
	}

	result, err := h.airdropSvc.Execute(services.AirdropRequest{
		ChannelID: channelID,
		Amount:    body.Amount,
		Note:      body.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNoActiveViewers):
			badRequest(c, err.Error())
			return
		case errors.Is(err, services.ErrDailyAirdropExceeded):
			var exceededErr *services.DailyAirdropExceededError
			if errors.As(err, &exceededErr) {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Error:   err.Error(),
					Data: gin.H{
						"remaining": exceededErr.Remaining,
					},
				})
				return
			}
			badRequest(c, err.Error())
			return
		case errors.Is(err, services.ErrInvalidPointsAmount):
			badRequest(c, err.Error())
			return
		default:
			internal(c)
			return
		}
	}

	ok(c, result)
}

func (h *AirdropHandler) authorize(claims *services.Claims, channelID string) (bool, error) {
	switch claims.Role {
	case models.RoleAdmin:
		return true, nil
	case models.RoleStreamer, models.RoleAgency:
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return false, err
		}
		if claims.Role == models.RoleStreamer {
			return h.streamerSvc.OwnsChannel(userID, channelID)
		}
		return h.agencySvc.OwnsChannel(userID, channelID)
	default:
		return false, nil
	}
}
