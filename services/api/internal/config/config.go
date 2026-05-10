package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultJWTAccessSecret    = "change-me-access-secret"
	defaultJWTRefreshSecret   = "change-me-refresh-secret"
	minJWTSecretLength        = 32
	defaultSepoliaRPCEndpoint = "https://ethereum-sepolia-rpc.publicnode.com"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	SMTP     SMTPConfig
	App      AppConfig
	Contract ContractConfig
	Internal InternalConfig
}

type ContractConfig struct {
	TachiContractAddress string // TACHI_CONTRACT_ADDRESS
	SepoliaSignerKey     string // SEPOLIA_SIGNER_KEY — never log or expose
	RPCEndpoint          string // SEPOLIA_RPC_URL
}

type InternalConfig struct {
	TachiyaSharedSecret string // TACHIYA_INTERNAL_SHARED_SECRET
	TachiyaBaseURL      string // TACHIYA_BASE_URL
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
	Port              string
	Env               string
	EnvSet            bool
	LogLevel          string
	EnableSwagger     bool
	EnableAutoMigrate bool
	EnableScheduler   bool
	AllowedOrigins    []string
	GinMode           string
	TrustedProxies    []string
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
	appEnv, appEnvSet := getEnvWithPresence("APP_ENV", "development")
	isProduction := appEnvSet && appEnv == "production"

	defaultEnableSwagger := !isProduction
	defaultGinMode := "debug"
	if isProduction {
		defaultGinMode = "release"
	}
	defaultAllowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}

	return &Config{
		Server: ServerConfig{
			Port:              getEnv("PORT", "8080"),
			Env:               appEnv,
			EnvSet:            appEnvSet,
			LogLevel:          getEnv("LOG_LEVEL", "info"),
			EnableSwagger:     getBoolEnv("ENABLE_SWAGGER", defaultEnableSwagger),
			EnableAutoMigrate: getBoolEnv("ENABLE_AUTOMIGRATE", true),
			EnableScheduler:   getBoolEnv("ENABLE_SCHEDULER", true),
			AllowedOrigins:    getCommaEnv("ALLOWED_ORIGINS", defaultAllowedOrigins),
			GinMode:           getEnv("GIN_MODE", defaultGinMode),
			TrustedProxies:    getCommaEnv("TRUSTED_PROXIES", nil),
		},
		Database: DatabaseConfig{
			DSN: getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=tachigo port=5432 sslmode=disable"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", defaultJWTAccessSecret),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", defaultJWTRefreshSecret),
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
		Contract: ContractConfig{
			TachiContractAddress: getEnv("TACHI_CONTRACT_ADDRESS", ""),
			SepoliaSignerKey:     getEnv("SEPOLIA_SIGNER_KEY", ""),
			RPCEndpoint:          getEnv("SEPOLIA_RPC_URL", defaultSepoliaRPCEndpoint),
		},
		Internal: InternalConfig{
			TachiyaSharedSecret: getEnv("TACHIYA_INTERNAL_SHARED_SECRET", ""),
			TachiyaBaseURL:      getEnv("TACHIYA_BASE_URL", "http://localhost:8001"),
		},
	}
}

func getEnv(key, fallback string) string {
	value, _ := getEnvWithPresence(key, fallback)
	return value
}

func getEnvWithPresence(key, fallback string) (string, bool) {
	if v := os.Getenv(key); v != "" {
		return v, true
	}
	return fallback, false
}

func ShouldValidateProductionSecrets(cfg *Config) bool {
	if cfg == nil {
		return true
	}
	return !cfg.Server.EnvSet || cfg.Server.Env != "development"
}

func ValidateProductionSecrets(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	if err := validateJWTSecret("JWT_ACCESS_SECRET", cfg.JWT.AccessSecret, defaultJWTAccessSecret); err != nil {
		return err
	}
	if err := validateJWTSecret("JWT_REFRESH_SECRET", cfg.JWT.RefreshSecret, defaultJWTRefreshSecret); err != nil {
		return err
	}
	if cfg.JWT.AccessSecret == cfg.JWT.RefreshSecret {
		return fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be different")
	}

	return nil
}

func validateJWTSecret(name, value, fallback string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	if value == fallback {
		return fmt.Errorf("%s must not use the default value", name)
	}
	if len(value) < minJWTSecretLength {
		return fmt.Errorf("%s must be at least %d characters", name, minJWTSecretLength)
	}
	return nil
}

func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getCommaEnv(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
