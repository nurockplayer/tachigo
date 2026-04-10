package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

type InternalPointsHandler struct {
	db *gorm.DB
}

func NewInternalPointsHandler(db *gorm.DB) *InternalPointsHandler {
	return &InternalPointsHandler{db: db}
}

func (h *InternalPointsHandler) GetUserPointsBalance(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		badRequest(c, "email is required")
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(c, "user not found")
			return
		}
		internal(c)
		return
	}

	var balance struct {
		SpendableBalance int64 `gorm:"column:spendable_balance"`
		CumulativeTotal  int64 `gorm:"column:cumulative_total"`
	}
	if err := h.db.Raw(`
		SELECT
			COALESCE(SUM(spendable_balance), 0) AS spendable_balance,
			COALESCE(SUM(cumulative_total), 0) AS cumulative_total
		FROM points_ledgers
		WHERE user_id = ?
	`, user.ID).Scan(&balance).Error; err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{
		"user_id":           user.ID,
		"email":             email,
		"spendable_balance": balance.SpendableBalance,
		"cumulative_total":  balance.CumulativeTotal,
	})
}
