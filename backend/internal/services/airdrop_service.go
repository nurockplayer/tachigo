package services

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

const defaultDailyAirdropLimit int64 = 5000
const maxAirdropTxRetries = 5

var (
	ErrNoActiveViewers      = errors.New("no active viewers")
	ErrDailyAirdropExceeded = errors.New("daily airdrop limit exceeded")
)

type DailyAirdropExceededError struct {
	Limit      int64
	TodayTotal int64
	Requested  int64
	Remaining  int64
}

func (e *DailyAirdropExceededError) Error() string {
	return ErrDailyAirdropExceeded.Error()
}

func (e *DailyAirdropExceededError) Unwrap() error {
	return ErrDailyAirdropExceeded
}

type AirdropRequest struct {
	ChannelID string
	Amount    int64
	Note      string
}

type AirdropRecipient struct {
	UserID          uuid.UUID `json:"user_id"`
	AllocatedPoints int64     `json:"allocated_points"`
}

type AirdropResult struct {
	AffectedCount int64              `json:"affected_count"`
	Distribution  []AirdropRecipient `json:"distribution"`
}

type airdropViewer struct {
	UserID             uuid.UUID
	AccumulatedSeconds int64
	Share              int64
	Remainder          int64
}

type AirdropService struct {
	db        *gorm.DB
	pointsSvc *PointsService
	configSvc *ChannelConfigService
}

func NewAirdropService(db *gorm.DB, pointsSvc *PointsService, configSvc *ChannelConfigService) *AirdropService {
	return &AirdropService{db: db, pointsSvc: pointsSvc, configSvc: configSvc}
}

func (s *AirdropService) TodayTotal(channelID string) (int64, error) {
	return s.todayTotal(s.db, channelID, time.Now().UTC())
}

func (s *AirdropService) Execute(req AirdropRequest) (*AirdropResult, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidPointsAmount
	}

	limit, err := s.dailyLimit(req.ChannelID)
	if err != nil {
		return nil, err
	}

	note := req.Note
	meta := PointsCreditMeta{}
	if note != "" {
		meta.Note = &note
	}

	var lastErr error
	for attempt := 0; attempt < maxAirdropTxRetries; attempt++ {
		// Anchor the UTC day once per attempt so that the daily-limit check and
		// every points_transaction.created_at within the same attempt share the
		// same reference time.  If the attempt is retried after a serialization
		// failure, we capture a fresh anchor for the new attempt.
		airdropAt := time.Now().UTC()
		var result *AirdropResult

		lastErr = s.db.Transaction(func(tx *gorm.DB) error {
			// Snapshot viewers inside the transaction so each retry reflects
			// the current session state at commit time.
			viewers, err := s.activeViewersInTx(tx, req.ChannelID)
			if err != nil {
				return err
			}
			if len(viewers) == 0 {
				return ErrNoActiveViewers
			}

			distributeAirdrop(viewers, req.Amount)

			todayTotal, err := s.todayTotal(tx, req.ChannelID, airdropAt)
			if err != nil {
				return err
			}
			if todayTotal > limit || req.Amount > limit-todayTotal {
				remaining := limit - todayTotal
				if remaining < 0 {
					remaining = 0
				}
				return &DailyAirdropExceededError{
					Limit:      limit,
					TodayTotal: todayTotal,
					Requested:  req.Amount,
					Remaining:  remaining,
				}
			}

			result = &AirdropResult{
				Distribution: make([]AirdropRecipient, 0, len(viewers)),
			}

			for _, viewer := range viewers {
				if viewer.Share <= 0 {
					continue
				}
				if err := s.pointsSvc.addPointsWithMetaAt(tx, airdropAt, viewer.UserID, req.ChannelID, models.TxSourceAirdrop, viewer.Share, meta); err != nil {
					return err
				}
				result.AffectedCount++
				result.Distribution = append(result.Distribution, AirdropRecipient{
					UserID:          viewer.UserID,
					AllocatedPoints: viewer.Share,
				})
			}

			return nil
		}, &sql.TxOptions{Isolation: sql.LevelSerializable})

		if lastErr == nil {
			return result, nil
		}
		if !isRetryableAirdropTxError(lastErr) {
			return nil, lastErr
		}
	}

	return nil, lastErr
}

func isRetryableAirdropTxError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked")
}

func (s *AirdropService) todayTotal(db *gorm.DB, channelID string, at time.Time) (int64, error) {
	startOfDay := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	startOfNextDay := startOfDay.Add(24 * time.Hour)

	var total int64
	err := db.Model(&models.PointsTransaction{}).
		Select("COALESCE(SUM(points_transactions.delta), 0)").
		Joins("JOIN points_ledgers ON points_ledgers.id = points_transactions.ledger_id").
		Where(
			"points_ledgers.channel_id = ? AND points_transactions.source = ? AND points_transactions.created_at >= ? AND points_transactions.created_at < ?",
			channelID,
			models.TxSourceAirdrop,
			startOfDay,
			startOfNextDay,
		).
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

// activeViewersInTx returns viewers with an active, non-stale session.
// A session is stale if last_heartbeat_at is older than staleThreshold (2 minutes),
// matching the definition in watch_service.go.
func (s *AirdropService) activeViewersInTx(db *gorm.DB, channelID string) ([]airdropViewer, error) {
	freshCutoff := time.Now().Add(-staleThreshold)
	var viewers []airdropViewer
	err := db.Model(&models.WatchSession{}).
		Select("user_id, accumulated_seconds").
		Where("channel_id = ? AND is_active = ? AND last_heartbeat_at > ?", channelID, true, freshCutoff).
		Order("accumulated_seconds DESC, user_id ASC").
		Scan(&viewers).Error
	return viewers, err
}

func (s *AirdropService) dailyLimit(channelID string) (int64, error) {
	cfg, err := s.configSvc.Get(channelID)
	if err != nil {
		return 0, err
	}
	if cfg == nil || cfg.DailyAirdropLimit <= 0 {
		return defaultDailyAirdropLimit, nil
	}
	return cfg.DailyAirdropLimit, nil
}

func distributeAirdrop(viewers []airdropViewer, amount int64) {
	if amount <= 0 || len(viewers) == 0 {
		return
	}

	var totalAccumulated int64
	for i := range viewers {
		if viewers[i].AccumulatedSeconds > 0 {
			totalAccumulated += viewers[i].AccumulatedSeconds
		}
	}

	if totalAccumulated == 0 {
		base := amount / int64(len(viewers))
		remainder := amount % int64(len(viewers))
		for i := range viewers {
			viewers[i].Share = base
			if int64(i) < remainder {
				viewers[i].Share++
			}
		}
		return
	}

	remaining := amount
	for i := range viewers {
		weighted := amount * viewers[i].AccumulatedSeconds
		viewers[i].Share = weighted / totalAccumulated
		viewers[i].Remainder = weighted % totalAccumulated
		remaining -= viewers[i].Share
	}

	sort.SliceStable(viewers, func(i, j int) bool {
		if viewers[i].Remainder == viewers[j].Remainder {
			if viewers[i].AccumulatedSeconds == viewers[j].AccumulatedSeconds {
				return viewers[i].UserID.String() < viewers[j].UserID.String()
			}
			return viewers[i].AccumulatedSeconds > viewers[j].AccumulatedSeconds
		}
		return viewers[i].Remainder > viewers[j].Remainder
	})

	for i := 0; i < int(remaining); i++ {
		viewers[i].Share++
	}
}
