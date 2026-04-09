package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type AgencyHandler struct {
	agencySvc    *services.AgencyService
	emailAuthSvc *services.EmailAuthService
}

func NewAgencyHandler(agencySvc *services.AgencyService, emailAuthSvc *services.EmailAuthService) *AgencyHandler {
	return &AgencyHandler{agencySvc: agencySvc, emailAuthSvc: emailAuthSvc}
}

type createAgencyRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type createAgencyResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type agencyStreamerResponse struct {
	ChannelID string    `json:"channel_id"`
	UserID    uuid.UUID `json:"user_id"`
}

func (h *AgencyHandler) Create(c *gin.Context) {
	var req createAgencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	user, err := h.agencySvc.Create(req.Name, req.Email)
	if err != nil {
		if errors.Is(err, services.ErrAgencyEmailTaken) {
			conflict(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrAgencyNameTaken) {
			conflict(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrAgencyNameTooLong) {
			badRequest(c, err.Error())
			return
		}
		log.Printf("agency create: unexpected error: %v", err)
		internal(c)
		return
	}

	if err := h.emailAuthSvc.ForgotPassword(*user.Email); err != nil {
		// Agency is already committed; ForgotPassword failure is non-fatal.
		// Admin can re-trigger via POST /auth/forgot-password if needed.
		if errors.Is(err, services.ErrPasswordResetEmailSend) {
			log.Printf("agency create: password setup email not delivered for user %s: %v", user.ID, err)
		} else {
			log.Printf("agency create: password reset token setup failed for user %s: %v", user.ID, err)
		}
	}

	created(c, createAgencyResponse{
		ID:   user.ID,
		Name: req.Name,
	})
}

func (h *AgencyHandler) ListStreamers(c *gin.Context) {
	agencyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid agency id")
		return
	}

	claims := middleware.MustClaims(c)
	if claims.Role == models.RoleAgency && claims.UserID != agencyID.String() {
		c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		return
	}

	streamers, err := h.agencySvc.ListStreamers(agencyID)
	if err != nil {
		if errors.Is(err, services.ErrAgencyNotFound) {
			notFound(c, "agency not found")
			return
		}
		log.Printf("agency list streamers: unexpected error: %v", err)
		internal(c)
		return
	}

	channelIDs := make([]string, 0, len(streamers))
	for _, streamer := range streamers {
		channelIDs = append(channelIDs, streamer.ChannelID)
	}

	userIDsByChannel, err := h.agencySvc.ListStreamerUserIDs(channelIDs)
	if err != nil {
		log.Printf("agency list streamer users: unexpected error: %v", err)
		internal(c)
		return
	}

	response := make([]agencyStreamerResponse, 0, len(streamers))
	for _, streamer := range streamers {
		userID, ok := userIDsByChannel[streamer.ChannelID]
		if !ok {
			log.Printf("agency %s: channel %s has no matching streamer user", agencyID, streamer.ChannelID)
			internal(c)
			return
		}
		response = append(response, agencyStreamerResponse{
			ChannelID: streamer.ChannelID,
			UserID:    userID,
		})
	}

	ok(c, gin.H{"streamers": response})
}
