package config

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestValidateProductionSecrets(t *testing.T) {
	validAccess := "access-secret-with-at-least-32-chars!"
	validRefresh := "refresh-secret-with-at-least-32-char"
	validConfig := func() *Config {
		return &Config{
			JWT: JWTConfig{
				AccessSecret:  validAccess,
				RefreshSecret: validRefresh,
			},
			SMTP: SMTPConfig{
				Host: "smtp.example.com",
			},
			OAuth: OAuthConfig{
				TokenEncryptionKey: "oauth-token-encryption-secret",
				Twitch: TwitchConfig{
					ClientID:        "twitch-client-id",
					ClientSecret:    "twitch-client-secret",
					ExtensionSecret: "twitch-extension-secret",
				},
				Google: GoogleConfig{
					ClientID:     "google-client-id",
					ClientSecret: "google-client-secret",
				},
			},
			Internal: InternalConfig{
				TachiyaSharedSecret: "tachiya-shared-secret",
			},
		}
	}
	withConfig := func(mutate func(*Config)) func() *Config {
		return func() *Config {
			cfg := validConfig()
			mutate(cfg)
			return cfg
		}
	}

	tests := []struct {
		name    string
		cfg     func() *Config
		wantErr string
	}{
		{
			name: "accepts valid production secrets",
			cfg:  validConfig,
		},
		{
			name: "rejects empty access secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.JWT.AccessSecret = ""
			}),
			wantErr: "JWT_ACCESS_SECRET",
		},
		{
			name: "rejects empty refresh secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.JWT.RefreshSecret = ""
			}),
			wantErr: "JWT_REFRESH_SECRET",
		},
		{
			name: "rejects default access secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.JWT.AccessSecret = defaultJWTAccessSecret
			}),
			wantErr: "default",
		},
		{
			name: "rejects default refresh secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.JWT.RefreshSecret = defaultJWTRefreshSecret
			}),
			wantErr: "default",
		},
		{
			name: "rejects short access secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.JWT.AccessSecret = "short-secret"
			}),
			wantErr: "at least 32 characters",
		},
		{
			name: "rejects short refresh secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.JWT.RefreshSecret = "short-secret"
			}),
			wantErr: "at least 32 characters",
		},
		{
			name: "rejects identical secrets",
			cfg: withConfig(func(cfg *Config) {
				cfg.JWT.RefreshSecret = cfg.JWT.AccessSecret
			}),
			wantErr: "must be different",
		},
		{
			name: "rejects missing SMTP host",
			cfg: withConfig(func(cfg *Config) {
				cfg.SMTP.Host = ""
			}),
			wantErr: "SMTP_HOST",
		},
		{
			name: "rejects missing Twitch OAuth client id",
			cfg: withConfig(func(cfg *Config) {
				cfg.OAuth.Twitch.ClientID = ""
			}),
			wantErr: "TWITCH_CLIENT_ID",
		},
		{
			name: "rejects missing Twitch OAuth client secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.OAuth.Twitch.ClientSecret = ""
			}),
			wantErr: "TWITCH_CLIENT_SECRET",
		},
		{
			name: "rejects missing Twitch extension secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.OAuth.Twitch.ExtensionSecret = ""
			}),
			wantErr: "TWITCH_EXTENSION_SECRET",
		},
		{
			name: "rejects missing Google OAuth client id",
			cfg: withConfig(func(cfg *Config) {
				cfg.OAuth.Google.ClientID = ""
			}),
			wantErr: "GOOGLE_CLIENT_ID",
		},
		{
			name: "rejects missing Google OAuth client secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.OAuth.Google.ClientSecret = ""
			}),
			wantErr: "GOOGLE_CLIENT_SECRET",
		},
		{
			name: "rejects missing Tachiya shared secret",
			cfg: withConfig(func(cfg *Config) {
				cfg.Internal.TachiyaSharedSecret = ""
			}),
			wantErr: "TACHIYA_INTERNAL_SHARED_SECRET",
		},
		{
			name: "rejects missing OAuth token encryption key",
			cfg: withConfig(func(cfg *Config) {
				cfg.OAuth.TokenEncryptionKey = ""
			}),
			wantErr: "OAUTH_TOKEN_ENCRYPTION_KEY",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProductionSecrets(tc.cfg())
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestShouldValidateProductionSecrets(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "validates when APP_ENV is missing and defaults to development",
			cfg: &Config{
				Server: ServerConfig{
					Env: "development",
				},
			},
			want: true,
		},
		{
			name: "skips validation only for explicit development",
			cfg: &Config{
				Server: ServerConfig{
					Env:    "development",
					EnvSet: true,
				},
			},
			want: false,
		},
		{
			name: "validates production",
			cfg: &Config{
				Server: ServerConfig{
					Env:    "production",
					EnvSet: true,
				},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldValidateProductionSecrets(tc.cfg)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestShouldEnableSwagger(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "defaults to enabled when config is missing",
			want: true,
		},
		{
			name: "enabled by default in development",
			cfg: &Config{
				Server: ServerConfig{Env: "development"},
			},
			want: true,
		},
		{
			name: "disabled by default in production",
			cfg: &Config{
				Server: ServerConfig{Env: "production"},
			},
			want: false,
		},
		{
			name: "explicit flag enables production",
			cfg: &Config{
				Server: ServerConfig{Env: "production", EnableSwagger: true, EnableSwaggerSet: true},
			},
			want: true,
		},
		{
			name: "explicit flag disables development",
			cfg: &Config{
				Server: ServerConfig{Env: "development", EnableSwagger: false, EnableSwaggerSet: true},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldEnableSwagger(tc.cfg)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestLoad_EnableSwagger(t *testing.T) {
	t.Run("records explicit true", func(t *testing.T) {
		t.Setenv("ENABLE_SWAGGER", "true")

		cfg := Load()
		if !cfg.Server.EnableSwaggerSet {
			t.Fatal("expected EnableSwaggerSet=true")
		}
		if !cfg.Server.EnableSwagger {
			t.Fatal("expected EnableSwagger=true")
		}
	})

	t.Run("records explicit false", func(t *testing.T) {
		t.Setenv("ENABLE_SWAGGER", "false")

		cfg := Load()
		if !cfg.Server.EnableSwaggerSet {
			t.Fatal("expected EnableSwaggerSet=true")
		}
		if cfg.Server.EnableSwagger {
			t.Fatal("expected EnableSwagger=false")
		}
	})
}

func TestLoad_ContractRPCEndpoint(t *testing.T) {
	t.Run("defaults to public Sepolia endpoint", func(t *testing.T) {
		t.Setenv("SEPOLIA_RPC_URL", "")

		cfg := Load()
		if cfg == nil {
			t.Fatalf("expected config, got nil")
		}
		if cfg.Contract.RPCEndpoint != defaultSepoliaRPCEndpoint {
			t.Fatalf("expected default %q, got %q", defaultSepoliaRPCEndpoint, cfg.Contract.RPCEndpoint)
		}
	})

	t.Run("supports env override", func(t *testing.T) {
		override := "https://example.invalid"
		t.Setenv("SEPOLIA_RPC_URL", override)

		cfg := Load()
		if cfg == nil {
			t.Fatalf("expected config, got nil")
		}
		if cfg.Contract.RPCEndpoint != override {
			t.Fatalf("expected %q, got %q", override, cfg.Contract.RPCEndpoint)
		}
	})
}

func TestLoad_RequestTimeoutSecondsValidation(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     time.Duration
	}{
		{
			name:     "falls back to default when invalid",
			envValue: "invalid",
			want:     30 * time.Second,
		},
		{
			name:     "falls back to default when zero",
			envValue: "0",
			want:     30 * time.Second,
		},
		{
			name:     "falls back to default when negative",
			envValue: "-1",
			want:     30 * time.Second,
		},
		{
			name:     "uses positive override",
			envValue: "45",
			want:     45 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("REQUEST_TIMEOUT_SECONDS", tc.envValue)

			cfg := Load()

			if cfg.Server.RequestTimeout != tc.want {
				t.Fatalf("RequestTimeout: want %v, got %v", tc.want, cfg.Server.RequestTimeout)
			}
		})
	}
}

func TestLoad_TracingConfig(t *testing.T) {
	t.Setenv("TRACING_ENABLED", "true")
	t.Setenv("OTEL_SERVICE_NAME", "tachigo-api-staging")
	t.Setenv("OTEL_ENVIRONMENT", "staging")
	t.Setenv("OTEL_TRACES_SAMPLE_RATIO", "0.25")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://otel-collector.example.com/v1/traces")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "x-api-key=abc123,tenant=tachigo")

	cfg := Load()

	if !cfg.Tracing.Enabled {
		t.Fatal("Tracing.Enabled: want true")
	}
	if cfg.Tracing.ServiceName != "tachigo-api-staging" {
		t.Fatalf("Tracing.ServiceName: want override, got %q", cfg.Tracing.ServiceName)
	}
	if cfg.Tracing.Environment != "staging" {
		t.Fatalf("Tracing.Environment: want staging, got %q", cfg.Tracing.Environment)
	}
	if cfg.Tracing.SampleRatio != 0.25 {
		t.Fatalf("Tracing.SampleRatio: want 0.25, got %v", cfg.Tracing.SampleRatio)
	}
	if cfg.Tracing.OTLPTracesEndpoint != "https://otel-collector.example.com/v1/traces" {
		t.Fatalf("Tracing.OTLPTracesEndpoint: got %q", cfg.Tracing.OTLPTracesEndpoint)
	}
	if !cfg.Tracing.OTLPInsecure {
		t.Fatal("Tracing.OTLPInsecure: want true")
	}
	if cfg.Tracing.OTLPHeaders["x-api-key"] != "abc123" || cfg.Tracing.OTLPHeaders["tenant"] != "tachigo" {
		t.Fatalf("Tracing.OTLPHeaders: got %v", cfg.Tracing.OTLPHeaders)
	}
}

func TestLoad_TracingEnvironmentFallsBackToAppEnv(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("OTEL_ENVIRONMENT", "")

	cfg := Load()

	if cfg.Tracing.Environment != "staging" {
		t.Fatalf("Tracing.Environment: want APP_ENV fallback, got %q", cfg.Tracing.Environment)
	}
}

func TestValidateTracing(t *testing.T) {
	tests := []struct {
		name    string
		cfg     TracingConfig
		wantErr string
	}{
		{
			name: "accepts disabled tracing without endpoint",
			cfg: TracingConfig{
				Enabled:     false,
				Environment: "production",
				SampleRatio: 1,
			},
		},
		{
			name: "accepts enabled staging config",
			cfg: TracingConfig{
				Enabled:            true,
				Environment:        "staging",
				SampleRatio:        0.10,
				OTLPTracesEndpoint: "https://otel-collector.example.com/v1/traces",
				ServiceName:        "tachigo-api",
			},
		},
		{
			name: "rejects enabled staging without endpoint",
			cfg: TracingConfig{
				Enabled:     true,
				Environment: "staging",
				SampleRatio: 0.10,
			},
			wantErr: "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		},
		{
			name: "rejects enabled production sample ratio below zero",
			cfg: TracingConfig{
				Enabled:            true,
				Environment:        "production",
				SampleRatio:        -0.01,
				OTLPTracesEndpoint: "https://otel-collector.example.com/v1/traces",
			},
			wantErr: "OTEL_TRACES_SAMPLE_RATIO",
		},
		{
			name: "rejects enabled production sample ratio above one",
			cfg: TracingConfig{
				Enabled:            true,
				Environment:        "production",
				SampleRatio:        1.01,
				OTLPTracesEndpoint: "https://otel-collector.example.com/v1/traces",
			},
			wantErr: "OTEL_TRACES_SAMPLE_RATIO",
		},
		{
			name: "rejects enabled production sample ratio NaN",
			cfg: TracingConfig{
				Enabled:            true,
				Environment:        "production",
				SampleRatio:        math.NaN(),
				OTLPTracesEndpoint: "https://otel-collector.example.com/v1/traces",
				ServiceName:        "tachigo-api",
			},
			wantErr: "OTEL_TRACES_SAMPLE_RATIO",
		},
		{
			name: "rejects enabled production sample ratio infinity",
			cfg: TracingConfig{
				Enabled:            true,
				Environment:        "production",
				SampleRatio:        math.Inf(1),
				OTLPTracesEndpoint: "https://otel-collector.example.com/v1/traces",
				ServiceName:        "tachigo-api",
			},
			wantErr: "OTEL_TRACES_SAMPLE_RATIO",
		},
		{
			name: "rejects enabled production without service name",
			cfg: TracingConfig{
				Enabled:            true,
				Environment:        "production",
				SampleRatio:        0.10,
				OTLPTracesEndpoint: "https://otel-collector.example.com/v1/traces",
			},
			wantErr: "OTEL_SERVICE_NAME",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTracing(tc.cfg)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateTracing() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateTracing() error = nil, want %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ValidateTracing() error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		fallback bool
		want     bool
	}{
		{"uses fallback when unset", "", true, true},
		{"uses fallback when unset false", "", false, false},
		{"parses true", "true", false, true},
		{"parses 1", "1", false, true},
		{"parses false", "false", true, false},
		{"parses 0", "0", true, false},
		{"falls back on invalid", "yes", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TEST_BOOL_ENV", tc.envValue)
			got := getBoolEnv("TEST_BOOL_ENV", tc.fallback)
			if got != tc.want {
				t.Fatalf("getBoolEnv(%q, %v) = %v, want %v", tc.envValue, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestGetCommaEnv(t *testing.T) {
	fallback := []string{"http://localhost:3000", "http://localhost:5173"}

	t.Run("uses fallback when unset", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != len(fallback) || got[0] != fallback[0] {
			t.Fatalf("expected fallback %v, got %v", fallback, got)
		}
	})

	t.Run("splits single value", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "https://example.com")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != 1 || got[0] != "https://example.com" {
			t.Fatalf("expected [https://example.com], got %v", got)
		}
	})

	t.Run("splits multiple values", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "https://a.com,https://b.com")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != 2 || got[0] != "https://a.com" || got[1] != "https://b.com" {
			t.Fatalf("expected [https://a.com https://b.com], got %v", got)
		}
	})

	t.Run("trims whitespace around tokens", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "10.0.0.1, 10.0.0.2 , 10.0.0.3")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != 3 || got[0] != "10.0.0.1" || got[1] != "10.0.0.2" || got[2] != "10.0.0.3" {
			t.Fatalf("expected trimmed tokens, got %v", got)
		}
	})

	t.Run("returns nil fallback when unset and fallback is nil", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "")
		got := getCommaEnv("TEST_COMMA_ENV", nil)
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})
}

func TestLoad_ServerConfig_Defaults(t *testing.T) {
	envVars := []string{
		"APP_ENV", "LOG_LEVEL", "ENABLE_SWAGGER",
		"ENABLE_AUTOMIGRATE", "ENABLE_SCHEDULER",
		"ALLOWED_ORIGINS", "GIN_MODE", "TRUSTED_PROXIES",
		"REQUEST_TIMEOUT_SECONDS",
	}
	for _, k := range envVars {
		t.Setenv(k, "")
	}

	cfg := Load()

	if cfg.Server.LogLevel != "info" {
		t.Errorf("LogLevel: want %q, got %q", "info", cfg.Server.LogLevel)
	}
	if !cfg.Server.EnableSwagger {
		t.Errorf("EnableSwagger: want true when APP_ENV unset, got false")
	}
	if !cfg.Server.EnableAutoMigrate {
		t.Errorf("EnableAutoMigrate: want true, got false")
	}
	if !cfg.Server.EnableScheduler {
		t.Errorf("EnableScheduler: want true, got false")
	}
	if len(cfg.Server.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins: want 2 defaults, got %v", cfg.Server.AllowedOrigins)
	}
	if cfg.Server.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins[0]: want http://localhost:3000, got %q", cfg.Server.AllowedOrigins[0])
	}
	if cfg.Server.GinMode != "debug" {
		t.Errorf("GinMode: want %q, got %q", "debug", cfg.Server.GinMode)
	}
	if cfg.Server.TrustedProxies != nil {
		t.Errorf("TrustedProxies: want nil, got %v", cfg.Server.TrustedProxies)
	}
	if cfg.Server.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout: want 30s, got %v", cfg.Server.RequestTimeout)
	}
}

func TestLoad_ServerConfig_ProductionDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("ENABLE_SWAGGER", "")
	t.Setenv("GIN_MODE", "")

	cfg := Load()

	if cfg.Server.EnableSwagger {
		t.Errorf("EnableSwagger: want false in production, got true")
	}
	if cfg.Server.GinMode != "release" {
		t.Errorf("GinMode: want %q in production, got %q", "release", cfg.Server.GinMode)
	}
}

func TestLoad_ServerConfig_EnvOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("ENABLE_SWAGGER", "true")
	t.Setenv("ENABLE_AUTOMIGRATE", "false")
	t.Setenv("ENABLE_SCHEDULER", "false")
	t.Setenv("ALLOWED_ORIGINS", "https://app.tachigo.io,https://admin.tachigo.io")
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1,10.0.0.2")
	t.Setenv("REQUEST_TIMEOUT_SECONDS", "45")

	cfg := Load()

	if cfg.Server.LogLevel != "warn" {
		t.Errorf("LogLevel: want %q, got %q", "warn", cfg.Server.LogLevel)
	}
	if !cfg.Server.EnableSwagger {
		t.Errorf("EnableSwagger: want true (overridden), got false")
	}
	if cfg.Server.EnableAutoMigrate {
		t.Errorf("EnableAutoMigrate: want false (overridden), got true")
	}
	if cfg.Server.EnableScheduler {
		t.Errorf("EnableScheduler: want false (overridden), got true")
	}
	if len(cfg.Server.AllowedOrigins) != 2 || cfg.Server.AllowedOrigins[1] != "https://admin.tachigo.io" {
		t.Errorf("AllowedOrigins: got %v", cfg.Server.AllowedOrigins)
	}
	if cfg.Server.GinMode != "debug" {
		t.Errorf("GinMode: want %q, got %q", "debug", cfg.Server.GinMode)
	}
	if len(cfg.Server.TrustedProxies) != 2 || cfg.Server.TrustedProxies[0] != "10.0.0.1" {
		t.Errorf("TrustedProxies: got %v", cfg.Server.TrustedProxies)
	}
	if cfg.Server.RequestTimeout != 45*time.Second {
		t.Errorf("RequestTimeout: want 45s, got %v", cfg.Server.RequestTimeout)
	}
}
