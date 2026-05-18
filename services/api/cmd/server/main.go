// @title           tachigo API
// @version         1.0
// @description     Backend API for tachigo — Twitch extension + Web3 rewards platform
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Enter: Bearer {access_token}
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	_ "github.com/tachigo/tachigo/docs"
	"github.com/tachigo/tachigo/internal/config"
)

func main() {
	// Load .env (ignore error in production where env is set externally)
	_ = godotenv.Load()

	cfg := config.Load()
	if config.ShouldValidateProductionSecrets(cfg) {
		if err := config.ValidateProductionSecrets(cfg); err != nil {
			log.Fatalf("invalid secrets: %v", err)
		}
	}

	db := bootstrap(cfg)
	serverCtx, serverStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer serverStop()
	tracingShutdown, err := configureTracing(serverCtx, cfg)
	if err != nil {
		log.Fatalf("invalid tracing config: %v", err)
	}
	r := wire(db, cfg, serverCtx)

	addr := ":" + cfg.Server.Port
	log.Printf("server starting on %s (env=%s)", addr, cfg.Server.Env)
	srv := newHTTPServer(addr, r)
	if err := runHTTPServer(serverCtx, srv, closeServerResources(db, tracingShutdown)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
