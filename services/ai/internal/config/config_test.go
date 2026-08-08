package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/avito-antifraud/ai/internal/config"
)

func TestLoad_AIConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(t *testing.T, cfg config.Config)
	}{
		{
			name: "defaults_success",
			env: map[string]string{
				"LLM_PROVIDER":       "openai_compat",
				"LLM_TIMEOUT":        "",
				"DIALOG_SESSION_TTL": "",
				"DIALOG_MAX_TURNS":   "",
			},
			wantErr: false,
			check: func(t *testing.T, cfg config.Config) {
				if cfg.HTTPAddr != ":8082" {
					t.Errorf("expected default HTTPAddr :8082, got %s", cfg.HTTPAddr)
				}
				if cfg.Provider != config.ProviderOpenAICompat {
					t.Errorf("expected provider openai_compat, got %s", cfg.Provider)
				}
				if cfg.Timeout != 60*time.Second {
					t.Errorf("expected timeout 60s, got %v", cfg.Timeout)
				}
				if cfg.SessionTTL != 30*time.Minute {
					t.Errorf("expected session TTL 30m, got %v", cfg.SessionTTL)
				}
				if cfg.MaxTurns != 20 {
					t.Errorf("expected max turns 20, got %d", cfg.MaxTurns)
				}
			},
		},
		{
			name: "mock_provider_success",
			env: map[string]string{
				"LLM_PROVIDER": "mock",
			},
			wantErr: false,
			check: func(t *testing.T, cfg config.Config) {
				if cfg.Provider != config.ProviderMock {
					t.Errorf("expected provider mock, got %s", cfg.Provider)
				}
			},
		},
		{
			name: "invalid_provider",
			env: map[string]string{
				"LLM_PROVIDER": "invalid_provider",
			},
			wantErr: true,
		},
		{
			name: "invalid_timeout_format",
			env: map[string]string{
				"LLM_PROVIDER": "openai_compat",
				"LLM_TIMEOUT":  "not_a_duration",
			},
			wantErr: true,
		},
		{
			name: "invalid_session_ttl_format",
			env: map[string]string{
				"LLM_PROVIDER":       "openai_compat",
				"DIALOG_SESSION_TTL": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid_max_turns_format",
			env: map[string]string{
				"LLM_PROVIDER":     "openai_compat",
				"DIALOG_MAX_TURNS": "abc",
			},
			wantErr: true,
		},
	}

	envKeys := []string{
		"HTTP_ADDR", "LLM_PROVIDER", "LLM_BASE_URL", "LLM_MODEL",
		"LLM_API_KEY", "LLM_TIMEOUT", "DIALOG_SESSION_TTL", "DIALOG_MAX_TURNS",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range envKeys {
				os.Unsetenv(k)
			}
			for k, v := range tt.env {
				if v != "" {
					os.Setenv(k, v)
				}
			}

			cfg, err := config.Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
