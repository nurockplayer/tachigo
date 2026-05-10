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
