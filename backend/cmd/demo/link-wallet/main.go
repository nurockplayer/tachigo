package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/database"
	"github.com/tachigo/tachigo/internal/demo"
)

func main() {
	var input demo.LinkDemoWalletInput
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags]\n\nDemo-only helper for the current no-MetaMask flow. Remove it once the real wallet connection flow is in place.\n\n", flag.CommandLine.Name())
		flag.PrintDefaults()
	}
	flag.StringVar(&input.UserID, "user-id", "", "demo viewer user UUID")
	flag.StringVar(&input.Email, "email", "", "demo viewer email")
	flag.StringVar(&input.WalletAddress, "wallet", "", "demo recipient wallet address")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()
	db := database.Connect(cfg.Database.DSN)

	linked, err := demo.LinkDemoWallet(context.Background(), db, input)
	if err != nil {
		log.Fatalf("link demo wallet: %v", err)
	}

	fmt.Printf("linked demo wallet: user_id=%s wallet=%s\n", linked.UserID, linked.WalletAddress)
}
