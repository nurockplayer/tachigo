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

type RaffleHandler struct {
	raffleSvc *services.RaffleService
}

const (
	maxRaffleCSVFileBytes          = 1 << 20
	maxRaffleCSVImportRequestBytes = maxRaffleCSVFileBytes + 64*1024
)

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

	raffle, err := h.raffleSvc.CreateContext(c.Request.Context(), userID, body.Title)
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

	raffles, err := h.raffleSvc.ListByStreamerContext(c.Request.Context(), userID)
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

	raffle, err := h.raffleSvc.GetByIDContext(c.Request.Context(), raffleID, userID)
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
// @Failure      409  {object}  Response
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

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRaffleCSVImportRequestBytes)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if isRequestBodyTooLarge(err) {
			raffleCSVTooLarge(c)
			return
		}
		badRequest(c, "file is required")
		return
	}
	defer file.Close()

	if header != nil && header.Size > maxRaffleCSVFileBytes {
		raffleCSVTooLarge(c)
		return
	}

	result, err := h.raffleSvc.ImportCSVContext(c.Request.Context(), raffleID, userID, file)
	if err != nil {
		if errors.Is(err, services.ErrRaffleNotFound) {
			notFound(c, "raffle not found")
			return
		}
		if errors.Is(err, services.ErrRaffleForbidden) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
			return
		}
		if errors.Is(err, services.ErrRaffleNotDraft) {
			conflict(c, err.Error())
			return
		}
		log.Printf("raffle import-csv raffle_id=%s: %v", raffleID, err)
		internal(c)
		return
	}
	ok(c, result)
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func raffleCSVTooLarge(c *gin.Context) {
	c.JSON(http.StatusRequestEntityTooLarge, Response{Success: false, Error: "csv upload is too large"})
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

	draws, err := h.raffleSvc.ListDrawsContext(c.Request.Context(), raffleID, userID)
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

	raffle, err := h.raffleSvc.CompleteContext(c.Request.Context(), raffleID, userID)
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

// Activate godoc
// @Summary      Lock the participant list and move raffle from draft to active
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path string true "Raffle ID"
// @Success      200  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Failure      409  {object}  Response
// @Router       /dashboard/raffles/{id}/activate [post]
func (h *RaffleHandler) Activate(c *gin.Context) {
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

	raffle, err := h.raffleSvc.ActivateContext(c.Request.Context(), raffleID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, "raffle not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrRaffleNotDraft):
			conflict(c, err.Error())
		default:
			log.Printf("raffle activate raffle_id=%s: %v", raffleID, err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"raffle": raffle})
}

// SetDiscordWebhook godoc
// @Summary      Set or clear the Discord webhook URL for a raffle
// @Tags         raffles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path   string                          true "Raffle ID"
// @Param        body body   object{discord_webhook_url=string} true "Webhook URL (empty string to clear)"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Router       /dashboard/raffles/{id}/discord-webhook [patch]
func (h *RaffleHandler) SetDiscordWebhook(c *gin.Context) {
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

	var body struct {
		DiscordWebhookURL *string `json:"discord_webhook_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	if body.DiscordWebhookURL == nil {
		badRequest(c, "discord_webhook_url is required (pass empty string to clear)")
		return
	}

	raffle, err := h.raffleSvc.SetDiscordWebhookContext(c.Request.Context(), raffleID, userID, *body.DiscordWebhookURL)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrInvalidDiscordWebhookURL):
			badRequest(c, err.Error())
		default:
			log.Printf("SetDiscordWebhook: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"raffle": raffle, "discord_webhook_configured": raffle.DiscordWebhookConfigured()})
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

	draw, err := h.raffleSvc.GetDrawByTokenContext(c.Request.Context(), token)
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
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        token path   string true "Claim token"
// @Param        body  body   services.ClaimInput true "Shipping info"
// @Success      200   {object}  Response
// @Failure      400   {object}  Response
// @Failure      401   {object}  Response
// @Failure      403   {object}  Response
// @Failure      404   {object}  Response
// @Failure      409   {object}  Response
// @Failure      410   {object}  Response
// @Router       /claim/{token} [post]
func (h *RaffleHandler) SubmitClaim(c *gin.Context) {
	token := c.Param("token")

	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "invalid user identity"})
		return
	}

	var input services.ClaimInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	claim, err := h.raffleSvc.SubmitClaimContext(c.Request.Context(), token, userID, input)
	if err != nil {
		if errors.Is(err, services.ErrClaimNotFound) {
			notFound(c, "claim not found")
			return
		}
		if errors.Is(err, services.ErrClaimTokenExpired) {
			c.JSON(http.StatusGone, Response{Success: false, Error: "claim token has expired"})
			return
		}
		if errors.Is(err, services.ErrClaimForbidden) {
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
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

// Snapshot godoc
// @Summary      Sync raffle entries from Twitch subscriber list
// @Tags         raffles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string true "Raffle ID"
// @Param        body body object{source=string} true "source must be twitch_api"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Failure      403  {object}  Response "Twitch token lacks required scopes; streamer must re-authorize"
// @Router       /dashboard/raffles/{id}/snapshot [post]
func (h *RaffleHandler) Snapshot(c *gin.Context) {
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

	var body struct {
		Source string `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	if body.Source != string(models.RaffleSourceTwitchAPI) {
		badRequest(c, "unsupported source: must be twitch_api")
		return
	}

	result, err := h.raffleSvc.SyncFromTwitchAPI(c.Request.Context(), raffleID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, "raffle not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrTwitchTokenMissing):
			badRequest(c, "no twitch access token: streamer must log in via twitch")
		case errors.Is(err, services.ErrTwitchInsufficientScope):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"code":    "twitch_reauth_required",
				"error":   "twitch token lacks required scopes; streamer must re-authorize",
			})
		case errors.Is(err, services.ErrUnsupportedRaffleSource):
			badRequest(c, "raffle source must be twitch_api to use snapshot sync")
		default:
			log.Printf("raffle snapshot: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"result": result})
}

// ── Prize Tier endpoints ──────────────────────────────────────────────────────

func (h *RaffleHandler) CreatePrizeTier(c *gin.Context) {
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
	var body services.CreatePrizeTierInput
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	tier, err := h.raffleSvc.CreatePrizeTierContext(c.Request.Context(), raffleID, userID, body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, "raffle not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrPrizeTierInvalidCount):
			badRequest(c, "winner_count must be at least 1")
		default:
			log.Printf("CreatePrizeTier: %v", err)
			internal(c)
		}
		return
	}
	created(c, gin.H{"tier": tier})
}

func (h *RaffleHandler) ListPrizeTiers(c *gin.Context) {
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
	tiers, err := h.raffleSvc.ListPrizeTiersContext(c.Request.Context(), raffleID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, "raffle not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		default:
			log.Printf("ListPrizeTiers: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"tiers": tiers})
}

func (h *RaffleHandler) UpdatePrizeTier(c *gin.Context) {
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
	tierID, err := uuid.Parse(c.Param("tier_id"))
	if err != nil {
		badRequest(c, "invalid tier id")
		return
	}
	var body services.UpdatePrizeTierInput
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	tier, err := h.raffleSvc.UpdatePrizeTierContext(c.Request.Context(), raffleID, tierID, userID, body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound), errors.Is(err, services.ErrPrizeTierNotFound):
			notFound(c, "not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrPrizeTierInvalidCount):
			badRequest(c, "invalid winner_count")
		default:
			log.Printf("UpdatePrizeTier: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"tier": tier})
}

func (h *RaffleHandler) DeletePrizeTier(c *gin.Context) {
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
	tierID, err := uuid.Parse(c.Param("tier_id"))
	if err != nil {
		badRequest(c, "invalid tier id")
		return
	}
	if err := h.raffleSvc.DeletePrizeTierContext(c.Request.Context(), raffleID, tierID, userID); err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound), errors.Is(err, services.ErrPrizeTierNotFound):
			notFound(c, "not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrPrizeTierHasDraws):
			conflict(c, "cannot delete a tier with existing draws")
		default:
			log.Printf("DeletePrizeTier: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{})
}

func (h *RaffleHandler) DrawFromTier(c *gin.Context) {
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
	tierID, err := uuid.Parse(c.Param("tier_id"))
	if err != nil {
		badRequest(c, "invalid tier id")
		return
	}
	draw, err := h.raffleSvc.DrawFromTierContext(c.Request.Context(), raffleID, tierID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound), errors.Is(err, services.ErrPrizeTierNotFound):
			notFound(c, "not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrRaffleNotActive):
			conflict(c, "raffle is not active")
		case errors.Is(err, services.ErrPrizeTierExhausted), errors.Is(err, services.ErrRaffleExhausted):
			conflict(c, "no more entries available")
		default:
			log.Printf("DrawFromTier: %v", err)
			internal(c)
		}
		return
	}
	created(c, gin.H{"draw": draw})
}

// SetMode godoc
// @Summary      Set raffle mode (Dashboard)
// @Tags         raffles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Raffle ID"
// @Param        body  body  object{mode=string}  true  "mode: public | subscribers_only"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Failure      409  {object}  Response
// @Router       /dashboard/raffles/{id}/mode [patch]
func (h *RaffleHandler) SetMode(c *gin.Context) {
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
	var body struct {
		Mode models.RaffleMode `json:"mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	if body.Mode != models.RaffleModePublic && body.Mode != models.RaffleModeSubscribers {
		badRequest(c, "mode must be 'public' or 'subscribers_only'")
		return
	}
	raffle, err := h.raffleSvc.SetModeContext(c.Request.Context(), raffleID, userID, body.Mode)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, err.Error())
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: err.Error()})
		case errors.Is(err, services.ErrRaffleNotDraft):
			conflict(c, err.Error())
		default:
			log.Printf("SetMode: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"raffle": raffle})
}

// SetEntryOpen godoc
// @Summary      Open or close raffle entry (Dashboard)
// @Tags         raffles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Raffle ID"
// @Param        body  body  object{entry_open=bool}  true  "entry_open flag"
// @Success      200  {object}  Response
// @Failure      400  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Failure      409  {object}  Response
// @Router       /dashboard/raffles/{id}/entry [patch]
func (h *RaffleHandler) SetEntryOpen(c *gin.Context) {
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
	var body struct {
		EntryOpen *bool `json:"entry_open"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	if body.EntryOpen == nil {
		badRequest(c, "entry_open is required")
		return
	}
	raffle, err := h.raffleSvc.SetEntryOpenContext(c.Request.Context(), raffleID, userID, *body.EntryOpen)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, err.Error())
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: err.Error()})
		case errors.Is(err, services.ErrRaffleNotActive):
			conflict(c, err.Error())
		default:
			log.Printf("SetEntryOpen: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"raffle": raffle})
}

// GetEntryStats godoc
// @Summary      Get entry statistics for a raffle (Dashboard)
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Raffle ID"
// @Success      200  {object}  Response
// @Failure      403  {object}  Response
// @Failure      404  {object}  Response
// @Router       /dashboard/raffles/{id}/entry-stats [get]
func (h *RaffleHandler) GetEntryStats(c *gin.Context) {
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
	stats, err := h.raffleSvc.GetEntryStatsContext(c.Request.Context(), raffleID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, err.Error())
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: err.Error()})
		default:
			log.Printf("GetEntryStats: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"stats": stats})
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

	draws, err := h.raffleSvc.GetDrawsByRafflePublicContext(c.Request.Context(), raffleID)
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
