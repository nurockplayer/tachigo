package main

import (
	"fmt"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
	"github.com/tachigo/tachigo/internal/models"
)

func main() {
	stmts, err := gormschema.New("postgres").Load(
		&models.User{},
		&models.AuthProvider{},
		&models.ShippingAddress{},
		&models.AgencyStreamer{},
		&models.ChannelConfig{},
		&models.Claim{},
		&models.ClaimItem{},
		&models.CouponRedemption{},
		&models.EmailVerification{},
		&models.PasswordReset{},
		&models.PointsLedger{},
		&models.PointsTransaction{},
		&models.Raffle{},
		&models.RaffleEntry{},
		&models.RaffleDraw{},
		&models.RaffleClaim{},
		&models.RefreshToken{},
		&models.Web3Nonce{},
		&models.Streamer{},
		&models.TachiBalance{},
		&models.WatchSession{},
		&models.WatchTimeStat{},
		&models.BroadcastTimeStat{},
		&models.BroadcastTimeLog{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(stmts)
}
