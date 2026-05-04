package schema

import "github.com/tachigo/tachigo/internal/models"

func AutoMigrateModels() []any {
	return []any{
		&models.User{},
		&models.AuthProvider{},
		&models.ShippingAddress{},
		&models.RefreshToken{},
		&models.Web3Nonce{},
		&models.EmailVerification{},
		&models.PasswordReset{},
		&models.Streamer{},
		&models.ChannelConfig{},
		&models.PointsLedger{},
		&models.PointsTransaction{},
		&models.WatchSession{},
		&models.WatchTimeStat{},
		&models.BroadcastTimeStat{},
		&models.BroadcastTimeLog{},
		&models.TachiBalance{},
		&models.Claim{},
		&models.ClaimItem{},
		&models.AgencyStreamer{},
		&models.Raffle{},
		&models.RaffleEntry{},
		&models.RaffleDraw{},
		&models.RaffleClaim{},
		&models.CouponRedemption{},
	}
}
