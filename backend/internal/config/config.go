package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	SMTP     SMTPConfig
	App      AppConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type AppConfig struct {
	FrontendURL string // base URL for links in emails, e.g. https://app.tachigo.io
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	DSN string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type OAuthConfig struct {
	Twitch TwitchConfig
	Google GoogleConfig
}

type TwitchConfig struct {
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	ExtensionSecret string // base64-encoded secret for verifying Extension JWTs
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func Load() *Config {
	accessTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MINUTES", "15"))
	refreshTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL_DAYS", "30"))
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))

	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			DSN: getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=tachigo port=5432 sslmode=disable"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "change-me-access-secret"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret"),
			AccessTTL:     time.Duration(accessTTL) * time.Minute,
			RefreshTTL:    time.Duration(refreshTTL) * 24 * time.Hour,
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     smtpPort,
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@tachigo.io"),
		},
		App: AppConfig{
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
		OAuth: OAuthConfig{
			Twitch: TwitchConfig{
				ClientID:        getEnv("TWITCH_CLIENT_ID", ""),
				ClientSecret:    getEnv("TWITCH_CLIENT_SECRET", ""),
				RedirectURL:     getEnv("TWITCH_REDIRECT_URL", "http://localhost:8080/api/v1/auth/twitch/callback"),
				ExtensionSecret: getEnv("TWITCH_EXTENSION_SECRET", ""),
			},
			Google: GoogleConfig{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
			},
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
