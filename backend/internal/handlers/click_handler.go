package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type ClickHandler struct {
	clickSvc *services.ClickService
}

func NewClickHandler(clickSvc *services.ClickService) *ClickHandler {
	return &ClickHandler{clickSvc: clickSvc}
}

type clickBody struct {
	ChannelID string `json:"channel_id" binding:"required"`
}

// Click handles POST /extension/click
//
//	@Summary      Click to earn points
//	@Description  Awards 1 point per click; enforces a 10-second server-side cooldown.
//	@Tags         extension
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      clickBody  true  "channel_id"
//	@Success      200   {object}  Response
//	@Failure      400   {object}  Response
//	@Failure      429   {object}  Response
//	@Router       /extension/click [post]
func (h *ClickHandler) Click(c *gin.Context) {
	claims := middleware.MustClaims(c)
	var body clickBody
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	result, err := h.clickSvc.Click(userID, body.ChannelID)
	if errors.Is(err, services.ErrClickCooldown) {
		c.JSON(http.StatusTooManyRequests, Response{
			Success: false,
			Error:   "cooldown active",
			Data: map[string]interface{}{
				"cooldown_remaining_ms": result.CooldownRemainingMs,
				"current_balance":       result.NewBalance,
			},
		})
		return
	}
	if err != nil {
		internal(c)
		return
	}

	ok(c, map[string]interface{}{
		"points_earned":        result.PointsEarned,
		"new_balance":          result.NewBalance,
		"cooldown_remaining_ms": 0,
	})
}
