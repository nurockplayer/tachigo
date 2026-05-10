package config

import (
	"strings"
	"testing"
)

func TestValidateProductionSecrets(t *testing.T) {
	validAccess := "access-secret-with-at-least-32-chars!"
	validRefresh := "refresh-secret-with-at-least-32-char"

	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "accepts valid production secrets",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  validAccess,
					RefreshSecret: validRefresh,
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
		},
		{
			name: "rejects empty access secret",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  "",
					RefreshSecret: validRefresh,
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
			wantErr: "JWT_ACCESS_SECRET",
		},
		{
			name: "rejects empty refresh secret",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  validAccess,
					RefreshSecret: "",
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
			wantErr: "JWT_REFRESH_SECRET",
		},
		{
			name: "rejects default access secret",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  defaultJWTAccessSecret,
					RefreshSecret: validRefresh,
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
			wantErr: "default",
		},
		{
			name: "rejects default refresh secret",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  validAccess,
					RefreshSecret: defaultJWTRefreshSecret,
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
			wantErr: "default",
		},
		{
			name: "rejects short access secret",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  "short-secret",
					RefreshSecret: validRefresh,
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
			wantErr: "at least 32 characters",
		},
		{
			name: "rejects short refresh secret",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  validAccess,
					RefreshSecret: "short-secret",
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
			wantErr: "at least 32 characters",
		},
		{
			name: "rejects identical secrets",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  validAccess,
					RefreshSecret: validAccess,
				},
				SMTP: SMTPConfig{
					Host: "smtp.example.com",
				},
			},
			wantErr: "must be different",
		},
		{
			name: "rejects missing SMTP host",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  validAccess,
					RefreshSecret: validRefresh,
				},
			},
			wantErr: "SMTP_HOST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProductionSecrets(tc.cfg)
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
}
