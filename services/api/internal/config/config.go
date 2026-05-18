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
	defaultRequestTimeoutSec  = 30
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
	Metrics  MetricsConfig
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

type MetricsConfig struct {
	EnableMetrics bool
	BearerToken   string // METRICS_BEARER_TOKEN — never log or expose
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
	EnableSwaggerSet  bool
	EnableAutoMigrate bool
	EnableScheduler   bool
	AllowedOrigins    []string
	GinMode           string
	TrustedProxies    []string
	RequestTimeout    time.Duration
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
	Twitch             TwitchConfig
	Google             GoogleConfig
	TokenEncryptionKey string
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
	requestTimeoutSeconds := getPositiveIntEnv("REQUEST_TIMEOUT_SECONDS", defaultRequestTimeoutSec)
	appEnv, appEnvSet := getEnvWithPresence("APP_ENV", "development")
	isProduction := appEnvSet && appEnv == "production"

	defaultEnableSwagger := !isProduction
	enableSwagger, enableSwaggerSet := getBoolEnvWithPresence("ENABLE_SWAGGER", defaultEnableSwagger)
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
			EnableSwagger:     enableSwagger,
			EnableSwaggerSet:  enableSwaggerSet,
			EnableAutoMigrate: getBoolEnv("ENABLE_AUTOMIGRATE", true),
			EnableScheduler:   getBoolEnv("ENABLE_SCHEDULER", true),
			AllowedOrigins:    getCommaEnv("ALLOWED_ORIGINS", defaultAllowedOrigins),
			GinMode:           getEnv("GIN_MODE", defaultGinMode),
			TrustedProxies:    getCommaEnv("TRUSTED_PROXIES", nil),
			RequestTimeout:    time.Duration(requestTimeoutSeconds) * time.Second,
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
			TokenEncryptionKey: getEnv("OAUTH_TOKEN_ENCRYPTION_KEY", ""),
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
		Metrics: MetricsConfig{
			EnableMetrics: getBoolEnv("ENABLE_METRICS", false),
			BearerToken:   strings.TrimSpace(getEnv("METRICS_BEARER_TOKEN", "")),
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

func getBoolEnvWithPresence(key string, fallback bool) (bool, bool) {
	raw, ok := getEnvWithPresence(key, "")
	if !ok {
		return fallback, false
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback, true
	}
	return value, true
}

func ShouldValidateProductionSecrets(cfg *Config) bool {
	if cfg == nil {
		return true
	}
	return !cfg.Server.EnvSet || cfg.Server.Env != "development"
}

func ShouldEnableSwagger(cfg *Config) bool {
	if cfg == nil {
		return true
	}
	if cfg.Server.EnableSwaggerSet {
		return cfg.Server.EnableSwagger
	}

	switch strings.ToLower(cfg.Server.Env) {
	case "", "development", "dev", "local":
		return true
	default:
		return false
	}
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
	if cfg.SMTP.Host == "" {
		return fmt.Errorf("SMTP_HOST must be configured when email-dependent production flows are enabled")
	}
	if cfg.Metrics.EnableMetrics && strings.TrimSpace(cfg.Metrics.BearerToken) == "" {
		return fmt.Errorf("METRICS_BEARER_TOKEN must be configured when ENABLE_METRICS is true in production")
	}

	// These launch flows are mounted unconditionally in production, so fail fast
	// before starting with OAuth, extension, or server-to-server credentials missing.
	requiredValues := []struct {
		name  string
		value string
		flow  string
	}{
		{"TWITCH_CLIENT_ID", cfg.OAuth.Twitch.ClientID, "Twitch OAuth and raffle snapshot flows are enabled"},
		{"TWITCH_CLIENT_SECRET", cfg.OAuth.Twitch.ClientSecret, "Twitch OAuth is enabled"},
		{"TWITCH_EXTENSION_SECRET", cfg.OAuth.Twitch.ExtensionSecret, "Twitch Extension auth is enabled"},
		{"GOOGLE_CLIENT_ID", cfg.OAuth.Google.ClientID, "Google OAuth is enabled"},
		{"GOOGLE_CLIENT_SECRET", cfg.OAuth.Google.ClientSecret, "Google OAuth is enabled"},
		{"TACHIYA_INTERNAL_SHARED_SECRET", cfg.Internal.TachiyaSharedSecret, "Tachiya coupon handoff is enabled"},
		{"OAUTH_TOKEN_ENCRYPTION_KEY", cfg.OAuth.TokenEncryptionKey, "OAuth token persistence is enabled"},
	}
	for _, required := range requiredValues {
		if err := validateRequiredProductionValue(required.name, required.value, required.flow); err != nil {
			return err
		}
	}

	return nil
}

func validateRequiredProductionValue(name, value, flow string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must be configured when %s in production", name, flow)
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

func getPositiveIntEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
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
