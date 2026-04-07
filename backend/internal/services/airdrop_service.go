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
	TotalAmount   int64              `json:"total_amount"`
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
	location, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		return 0, err
	}

	now := time.Now().In(location)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)

	var total int64
	err = s.db.Model(&models.PointsTransaction{}).
		Select("COALESCE(SUM(points_transactions.delta), 0)").
		Joins("JOIN points_ledgers ON points_ledgers.id = points_transactions.ledger_id").
		Where(
			"points_ledgers.channel_id = ? AND points_transactions.source = ? AND points_transactions.created_at >= ?",
			channelID,
			models.TxSourceAirdrop,
			startOfDay,
		).
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *AirdropService) Execute(req AirdropRequest) (*AirdropResult, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidPointsAmount
	}

	viewers, err := s.activeViewers(req.ChannelID)
	if err != nil {
		return nil, err
	}
	if len(viewers) == 0 {
		return nil, ErrNoActiveViewers
	}

	limit, err := s.dailyLimit(req.ChannelID)
	if err != nil {
		return nil, err
	}

	distributeAirdrop(viewers, req.Amount)
	note := req.Note
	meta := PointsCreditMeta{}
	if note != "" {
		meta.Note = &note
	}

	for attempt := 0; attempt < maxAirdropTxRetries; attempt++ {
		result := &AirdropResult{
			TotalAmount:  req.Amount,
			Distribution: make([]AirdropRecipient, 0, len(viewers)),
		}

		err = s.db.Transaction(func(tx *gorm.DB) error {
			todayTotal, err := s.todayTotal(tx, req.ChannelID)
			if err != nil {
				return err
			}
			if todayTotal+req.Amount > limit {
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

			for _, viewer := range viewers {
				if viewer.Share <= 0 {
					continue
				}
				if err := s.pointsSvc.addPointsWithMeta(tx, viewer.UserID, req.ChannelID, models.TxSourceAirdrop, viewer.Share, meta); err != nil {
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
		if err == nil {
			return result, nil
		}
		if !isRetryableAirdropTxError(err) {
			return nil, err
		}
	}

	return nil, err
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

func (s *AirdropService) todayTotal(db *gorm.DB, channelID string) (int64, error) {
	location, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		return 0, err
	}

	now := time.Now().In(location)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)

	var total int64
	err = db.Model(&models.PointsTransaction{}).
		Select("COALESCE(SUM(points_transactions.delta), 0)").
		Joins("JOIN points_ledgers ON points_ledgers.id = points_transactions.ledger_id").
		Where(
			"points_ledgers.channel_id = ? AND points_transactions.source = ? AND points_transactions.created_at >= ?",
			channelID,
			models.TxSourceAirdrop,
			startOfDay,
		).
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *AirdropService) activeViewers(channelID string) ([]airdropViewer, error) {
	var viewers []airdropViewer
	err := s.db.Model(&models.WatchSession{}).
		Select("user_id, accumulated_seconds").
		Where("channel_id = ? AND is_active = ?", channelID, true).
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
