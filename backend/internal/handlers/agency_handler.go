package handlers

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/services"
)

type AgencyHandler struct {
	agencySvc *services.AgencyService
}

func NewAgencyHandler(agencySvc *services.AgencyService) *AgencyHandler {
	return &AgencyHandler{agencySvc: agencySvc}
}

type createAgencyRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type createAgencyResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
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

	created(c, createAgencyResponse{
		ID:   user.ID,
		Name: req.Name,
	})
}
