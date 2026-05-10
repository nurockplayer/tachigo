package services

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tachigo/tachigo/internal/testutil"
)

// discordCapture is a RoundTripper-based fake that captures Discord webhook POST bodies.
type discordCapture struct {
	url    string
	client *http.Client
	ch     chan string
}

func newDiscordCapture(t *testing.T) *discordCapture {
	t.Helper()

	dc := &discordCapture{
		url: "https://discord.com/api/webhooks/123/test",
		ch:  make(chan string, 1),
	}
	dc.client = testutil.NewHTTPClient(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		select {
		case dc.ch <- string(body):
		default:
		}
		return testutil.NewStringResponse(http.StatusNoContent, ""), nil
	})
	return dc
}

func (dc *discordCapture) URL() string          { return dc.url }
func (dc *discordCapture) Client() *http.Client { return dc.client }
func (dc *discordCapture) Close()               {}

func (dc *discordCapture) expectPayload(t *testing.T) map[string]interface{} {
	t.Helper()
	select {
	case raw := <-dc.ch:
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("discord payload not valid JSON: %v", err)
		}
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: no discord webhook request received")
		return nil
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestSetDiscordWebhook_SetsURL(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "owner@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	svc := &RaffleService{db: db}

	webhookURL := "https://discord.com/api/webhooks/123/abc"
	raffle, err := svc.SetDiscordWebhook(raffleID, ownerID, webhookURL)
	if err != nil {
		t.Fatalf("SetDiscordWebhook: %v", err)
	}
	if raffle.DiscordWebhookURL == nil || *raffle.DiscordWebhookURL != webhookURL {
		t.Errorf("expected webhook URL %q, got %v", webhookURL, raffle.DiscordWebhookURL)
	}
}

func TestSetDiscordWebhook_ClearsURL(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "owner2@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	svc := &RaffleService{db: db}

	// set then clear
	if _, err := svc.SetDiscordWebhook(raffleID, ownerID, "https://discord.com/api/webhooks/123/abc"); err != nil {
		t.Fatalf("set: %v", err)
	}
	raffle, err := svc.SetDiscordWebhook(raffleID, ownerID, "")
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if raffle.DiscordWebhookURL != nil {
		t.Errorf("expected nil after clear, got %v", *raffle.DiscordWebhookURL)
	}
}

func TestSetDiscordWebhook_RejectsInvalidURL(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "owner3@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	svc := &RaffleService{db: db}

	_, err := svc.SetDiscordWebhook(raffleID, ownerID, "https://evil.com/webhook")
	if err == nil {
		t.Fatal("expected error for invalid webhook URL")
	}
	if !strings.Contains(err.Error(), "invalid Discord webhook URL") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDrawNext_SendsDiscordNotification(t *testing.T) {
	db := newTestDB(t)
	dc := newDiscordCapture(t)
	defer dc.Close()

	ownerID := seedUserWithEmail(t, db, "owner4@example.com")
	winnerID := seedUserWithEmail(t, db, "winner4@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "winner4_twitch")

	svc := &RaffleService{
		db:          db,
		frontendURL: "http://localhost:3000",
		httpClient:  dc.Client(),
	}
	// patch webhook URL directly into DB
	webhookURL := dc.URL()
	if err := db.Exec(`UPDATE raffles SET discord_webhook_url = ? WHERE id = ?`, webhookURL, raffleID.String()).Error; err != nil {
		t.Fatalf("seed webhook: %v", err)
	}

	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}
	if draw == nil {
		t.Fatal("expected draw, got nil")
	}

	payload := dc.expectPayload(t)
	content, _ := payload["content"].(string)
	if strings.Contains(content, draw.ClaimToken) {
		t.Errorf("discord payload must NOT contain claim token (security: public webhook)")
	}
	if strings.Contains(content, "/claim/") {
		t.Errorf("discord payload must NOT contain claim link (security: public webhook)")
	}
	if strings.Contains(content, "Email") {
		t.Errorf("discord payload must NOT imply winner email delivery succeeded")
	}
	if strings.Contains(content, "已透過 Email 寄送") {
		t.Errorf("discord payload must NOT claim winner email was definitely sent")
	}
	if !strings.Contains(content, draw.Entry.TwitchLogin) {
		t.Errorf("discord payload should contain winner twitch login")
	}
}

func TestDrawNext_SkipsDiscordWhenNoWebhook(t *testing.T) {
	db := newTestDB(t)
	dc := newDiscordCapture(t)
	defer dc.Close()

	ownerID := seedUserWithEmail(t, db, "owner5@example.com")
	winnerID := seedUserWithEmail(t, db, "winner5@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "winner5_twitch")

	svc := &RaffleService{db: db, httpClient: dc.Client()}

	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}
	if draw == nil {
		t.Fatal("expected draw")
	}

	select {
	case <-dc.ch:
		t.Error("discord webhook called unexpectedly")
	case <-time.After(100 * time.Millisecond):
	}
}
