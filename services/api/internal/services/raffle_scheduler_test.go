package services

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/testutil"
)

func TestNextSchedulerRun(t *testing.T) {
	utc := time.UTC
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "before 23:55 same day",
			now:  time.Date(2025, 1, 1, 12, 0, 0, 0, utc),
			want: time.Date(2025, 1, 1, 23, 55, 0, 0, utc),
		},
		{
			name: "one second before 23:55",
			now:  time.Date(2025, 1, 1, 23, 54, 59, 0, utc),
			want: time.Date(2025, 1, 1, 23, 55, 0, 0, utc),
		},
		{
			name: "exactly at 23:55 → tomorrow",
			now:  time.Date(2025, 1, 1, 23, 55, 0, 0, utc),
			want: time.Date(2025, 1, 2, 23, 55, 0, 0, utc),
		},
		{
			name: "after 23:55 → tomorrow",
			now:  time.Date(2025, 1, 1, 23, 56, 0, 0, utc),
			want: time.Date(2025, 1, 2, 23, 55, 0, 0, utc),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextSchedulerRun(tc.now)
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRunScheduledSnapshots_SkipsOutOfWindow(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)

	user := insertRaffleTestUser(t, db)
	scheduledAt := time.Now().UTC().Add(2 * time.Hour)
	raffle := insertScheduledRaffle(t, db, user.ID, scheduledAt, models.RaffleSourceTwitchAPI)

	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var r models.Raffle
	db.First(&r, "id = ?", raffle.ID)
	if r.Status != models.RaffleStatusDraft {
		t.Errorf("expected draft, got %s", r.Status)
	}
}

func TestRunScheduledSnapshots_SkipsCompleted(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)

	user := insertRaffleTestUser(t, db)
	scheduledAt := time.Now().UTC().Add(5 * time.Minute)
	raffle := insertScheduledRaffle(t, db, user.ID, scheduledAt, models.RaffleSourceTwitchAPI)
	db.Model(&models.Raffle{}).Where("id = ?", raffle.ID).Update("status", models.RaffleStatusCompleted)

	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var r models.Raffle
	db.First(&r, "id = ?", raffle.ID)
	if r.Status != models.RaffleStatusCompleted {
		t.Errorf("expected completed, got %s", r.Status)
	}
}

func TestRunScheduledSnapshots_TwitchAPISuccess(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "test-client-id", "", nil)
	svc.SetTwitchBaseURL("https://twitch.test")
	svc.httpClient = testutil.NewHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet && r.URL.Path == "/helix/subscriptions" {
			return testutil.NewStringResponse(http.StatusOK, `{"data":[],"pagination":{}}`), nil
		}
		return testutil.NewStringResponse(http.StatusNotFound, ""), nil
	})

	user := insertRaffleTestUser(t, db)
	insertRaffleTwitchProvider(t, db, user.ID, "broadcaster123", "streamer_token")

	scheduledAt := time.Now().UTC().Add(5 * time.Minute)
	raffle := insertScheduledRaffle(t, db, user.ID, scheduledAt, models.RaffleSourceTwitchAPI)

	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var r models.Raffle
	db.First(&r, "id = ?", raffle.ID)
	if r.Status != models.RaffleStatusActive {
		t.Errorf("expected active after snapshot, got %s", r.Status)
	}
}

func TestRunScheduledSnapshots_TwitchTokenMissing_StaysDraft(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "test-client-id", "", nil)

	user := insertRaffleTestUser(t, db)
	scheduledAt := time.Now().UTC().Add(5 * time.Minute)
	raffle := insertScheduledRaffle(t, db, user.ID, scheduledAt, models.RaffleSourceTwitchAPI)

	// Per-raffle errors are logged, not propagated.
	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("unexpected batch error: %v", err)
	}

	var r models.Raffle
	db.First(&r, "id = ?", raffle.ID)
	if r.Status != models.RaffleStatusDraft {
		t.Errorf("expected draft after failed snapshot, got %s", r.Status)
	}
}

func TestRunScheduledSnapshots_LogsSafeJobEvents(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "test-client-id", "", nil)

	user := insertRaffleTestUser(t, db)
	scheduledAt := time.Now().UTC().Add(5 * time.Minute)
	raffle := insertScheduledRaffle(t, db, user.ID, scheduledAt, models.RaffleSourceTwitchAPI)

	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
	})

	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("unexpected batch error: %v", err)
	}

	line := logs.String()
	for _, want := range []string{
		"event=raffle_scheduled_snapshots_start",
		"event=raffle_scheduled_snapshot_error",
		"event=raffle_scheduled_snapshots_complete",
		"job=raffle_scheduled_snapshots",
		"candidate_count=1",
		"failed=1",
		"raffle_id=" + raffle.ID.String(),
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected scheduler log to contain %q, got %q", want, line)
		}
	}
	for _, leaked := range []string{"access_token", "refresh_token", "Authorization", "Bearer"} {
		if strings.Contains(line, leaked) {
			t.Fatalf("scheduler log leaked %q: %s", leaked, line)
		}
	}
}

func TestSafeScheduledJobErrorRedactsSensitiveMaterial(t *testing.T) {
	for _, err := range []error{
		errors.New("upstream failed with access_token=secret"),
		errors.New("Authorization: Bearer secret"),
		errors.New("client_secret leaked"),
	} {
		if got := safeScheduledJobError(err); got != "[redacted]" {
			t.Fatalf("expected redacted error, got %q", got)
		}
	}

	if got := safeScheduledJobError(errors.New("twitch token missing")); got != `"twitch token missing"` {
		t.Fatalf("expected safe operational error to remain visible, got %q", got)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func insertRaffleTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	uname := "raffleuser_" + uuid.New().String()[:8]
	user := models.User{
		Username: &uname,
		Role:     models.RoleViewer,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return user
}

func insertRaffleTwitchProvider(t *testing.T, db *gorm.DB, userID uuid.UUID, broadcasterID, token string) {
	t.Helper()
	tok := token
	ap := models.AuthProvider{
		UserID:      userID,
		Provider:    models.ProviderTwitch,
		ProviderID:  broadcasterID,
		AccessToken: &tok,
	}
	if err := db.Create(&ap).Error; err != nil {
		t.Fatalf("insert twitch provider: %v", err)
	}
}

func insertScheduledRaffle(t *testing.T, db *gorm.DB, userID uuid.UUID, scheduledAt time.Time, source models.RaffleSource) models.Raffle {
	t.Helper()
	raffle := models.Raffle{
		UserID:      userID,
		Title:       "Test Raffle",
		Status:      models.RaffleStatusDraft,
		Source:      source,
		ScheduledAt: &scheduledAt,
	}
	if err := db.Create(&raffle).Error; err != nil {
		t.Fatalf("insert raffle: %v", err)
	}
	return raffle
}
