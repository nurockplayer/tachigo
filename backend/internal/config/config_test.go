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
			},
		},
		{
			name: "rejects empty access secret",
			cfg: &Config{
				JWT: JWTConfig{
					AccessSecret:  "",
					RefreshSecret: validRefresh,
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
			},
			wantErr: "must be different",
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
