package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type RaffleHandler struct {
	raffleSvc *services.RaffleService
}

func NewRaffleHandler(svc *services.RaffleService) *RaffleHandler {
	return &RaffleHandler{raffleSvc: svc}
}

// ── Dashboard endpoints (JWT + RoleStreamer) ──────────────────────────────────

// Create godoc
// @Summary      Create a raffle
// @Tags         raffles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body object{title=string} true "Raffle title"
// @Success      201  {object}  Response
// @Failure      400  {object}  Response
// @Router       /dashboard/raffles [post]
func (h *RaffleHandler) Create(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	var body struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	raffle, err := h.raffleSvc.Create(userID, body.Title)
	if err != nil {
		log.Printf("raffle create: %v", err)
		internal(c)
		return
	}
	created(c, gin.H{"raffle": raffle})
}

// List godoc
// @Summary      List my raffles
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  Response
// @Router       /dashboard/raffles [get]
func (h *RaffleHandler) List(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	raffles, err := h.raffleSvc.ListByStreamer(userID)
	if err != nil {
		log.Printf("raffle list: %v", err)
		internal(c)
		return
	}
	ok(c, gin.H{"raffles": raffles})
}

// Get godoc
// @Summary      Get raffle by ID
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path string true "Raffle ID"
// @Success      200  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Router       /dashboard/raffles/{id} [get]
func (h *RaffleHandler) Get(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}

	raffle, err := h.raffleSvc.GetByID(raffleID, userID)
	if err != nil {
		if errors.Is(err, services.ErrRaffleNotFound) {
			notFound(c, "raffle not found")
			return
		}
		if errors.Is(err, services.ErrRaffleForbidden) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
		internal(c)
		return
	}
	ok(c, gin.H{"raffle": raffle})
}

// ImportCSV godoc
// @Summary      Import participants from CSV
// @Tags         raffles
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        id   path     string true "Raffle ID"
// @Param        file formData file   true "CSV file (column 1: twitch_login, column 2: display_name)"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Router       /dashboard/raffles/{id}/entries/import-csv [post]
func (h *RaffleHandler) ImportCSV(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return
	}
	defer file.Close()

	result, err := h.raffleSvc.ImportCSV(raffleID, userID, file)
	if err != nil {
		if errors.Is(err, services.ErrRaffleNotFound) {
			notFound(c, "raffle not found")
			return
		}
		if errors.Is(err, services.ErrRaffleForbidden) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
		log.Printf("raffle import-csv raffle_id=%s: %v", raffleID, err)
		internal(c)
		return
	}
	ok(c, result)
}

// DrawNext godoc
// @Summary      Draw the next winner
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path string true "Raffle ID"
// @Success      201  {object}  Response
// @Failure      409  {object}  Response
// @Router       /dashboard/raffles/{id}/draws [post]
func (h *RaffleHandler) DrawNext(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}

	draw, err := h.raffleSvc.DrawNext(raffleID, userID)
	if err != nil {
		if errors.Is(err, services.ErrRaffleNotFound) {
			notFound(c, "raffle not found")
			return
		}
		if errors.Is(err, services.ErrRaffleForbidden) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
		if errors.Is(err, services.ErrRaffleExhausted) {
			conflict(c, "all entries have been drawn")
			return
		}
		if errors.Is(err, services.ErrRaffleCompleted) {
			conflict(c, "raffle is already completed")
			return
		}
		log.Printf("raffle draw-next raffle_id=%s: %v", raffleID, err)
		internal(c)
		return
	}
	created(c, gin.H{"draw": draw})
}

// ListDraws godoc
// @Summary      List draws for a raffle
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path string true "Raffle ID"
// @Success      200  {object}  Response
// @Router       /dashboard/raffles/{id}/draws [get]
func (h *RaffleHandler) ListDraws(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}

	draws, err := h.raffleSvc.ListDraws(raffleID, userID)
	if err != nil {
		if errors.Is(err, services.ErrRaffleNotFound) {
			notFound(c, "raffle not found")
			return
		}
		if errors.Is(err, services.ErrRaffleForbidden) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
		internal(c)
		return
	}
	ok(c, gin.H{"draws": draws})
}

// Complete godoc
// @Summary      Mark a raffle as completed
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path string true "Raffle ID"
// @Success      200  {object}  Response
// @Router       /dashboard/raffles/{id}/complete [post]
func (h *RaffleHandler) Complete(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}

	raffle, err := h.raffleSvc.Complete(raffleID, userID)
	if err != nil {
		if errors.Is(err, services.ErrRaffleNotFound) {
			notFound(c, "raffle not found")
			return
		}
		if errors.Is(err, services.ErrRaffleForbidden) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
		internal(c)
		return
	}
	ok(c, gin.H{"raffle": raffle})
}

// ── Public endpoints (no auth) ────────────────────────────────────────────────

// GetClaim godoc
// @Summary      Get claim info by token
// @Tags         claim
// @Produce      json
// @Param        token path string true "Claim token"
// @Success      200   {object}  Response
// @Failure      404   {object}  Response
// @Failure      410   {object}  Response
// @Router       /claim/{token} [get]
func (h *RaffleHandler) GetClaim(c *gin.Context) {
	token := c.Param("token")

	draw, err := h.raffleSvc.GetDrawByToken(token)
	if err != nil {
		if errors.Is(err, services.ErrClaimNotFound) {
			notFound(c, "claim not found")
			return
		}
		if errors.Is(err, services.ErrClaimTokenExpired) {
			c.JSON(http.StatusGone, Response{Success: false, Error: "claim token has expired"})
			return
		}
		internal(c)
		return
	}
	ok(c, gin.H{"draw": draw})
}

// SubmitClaim godoc
// @Summary      Submit shipping info for a claim
// @Tags         claim
// @Accept       json
// @Produce      json
// @Param        token path   string true "Claim token"
// @Param        body  body   services.ClaimInput true "Shipping info"
// @Success      200   {object}  Response
// @Failure      404   {object}  Response
// @Failure      409   {object}  Response
// @Failure      410   {object}  Response
// @Router       /claim/{token} [post]
func (h *RaffleHandler) SubmitClaim(c *gin.Context) {
	token := c.Param("token")

	var input services.ClaimInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	claim, err := h.raffleSvc.SubmitClaim(token, input)
	if err != nil {
		if errors.Is(err, services.ErrClaimNotFound) {
			notFound(c, "claim not found")
			return
		}
		if errors.Is(err, services.ErrClaimTokenExpired) {
			c.JSON(http.StatusGone, Response{Success: false, Error: "claim token has expired"})
			return
		}
		if errors.Is(err, services.ErrClaimAlreadyDone) {
			conflict(c, "claim already submitted")
			return
		}
		log.Printf("raffle submit-claim: %v", err)
		internal(c)
		return
	}
	ok(c, gin.H{"claim": claim})
}

// ── Extension endpoint ────────────────────────────────────────────────────────

type publicDrawView struct {
	ID       string `json:"id"`
	RaffleID string `json:"raffle_id"`
	DrawnAt  string `json:"drawn_at"`
	Entry    struct {
		ID          string `json:"id"`
		TwitchLogin string `json:"twitch_login"`
		DisplayName string `json:"display_name"`
	} `json:"entry"`
}

// GetResult godoc
// @Summary      Get drawn winners for a raffle (Extension)
// @Tags         raffles
// @Produce      json
// @Param        id   path string true "Raffle ID"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Router       /extension/raffles/{id}/result [get]
func (h *RaffleHandler) GetResult(c *gin.Context) {
	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}

	draws, err := h.raffleSvc.GetDrawsByRafflePublic(raffleID)
	if err != nil {
		internal(c)
		return
	}

	views := make([]publicDrawView, len(draws))
	for i, d := range draws {
		views[i] = publicDrawView{
			ID:       d.ID.String(),
			RaffleID: d.RaffleID.String(),
			DrawnAt:  d.DrawnAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		views[i].Entry.ID = d.Entry.ID.String()
		views[i].Entry.TwitchLogin = d.Entry.TwitchLogin
		views[i].Entry.DisplayName = d.Entry.DisplayName
	}
	ok(c, gin.H{"draws": views})
}
