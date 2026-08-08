package config_test

import (
	"os"
	"testing"

	"github.com/avito-antifraud/api/internal/config"
)

func TestLoad_APIConfig(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantErr   bool
		wantAddr  string
		wantDBURL string
	}{
		{
			name: "success_default_http",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"HTTP_ADDR":    "",
			},
			wantErr:   false,
			wantAddr:  ":8081",
			wantDBURL: "postgres://user:pass@localhost:5432/db",
		},
		{
			name: "success_custom_http",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"HTTP_ADDR":    ":9090",
			},
			wantErr:   false,
			wantAddr:  ":9090",
			wantDBURL: "postgres://user:pass@localhost:5432/db",
		},
		{
			name: "missing_database_url",
			env: map[string]string{
				"DATABASE_URL": "",
				"HTTP_ADDR":    ":8081",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("HTTP_ADDR")
			os.Unsetenv("DATABASE_URL")

			for k, v := range tt.env {
				if v != "" {
					os.Setenv(k, v)
				}
			}

			cfg, err := config.Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if cfg.HTTPAddr != tt.wantAddr {
					t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, tt.wantAddr)
				}
				if cfg.DatabaseURL != tt.wantDBURL {
					t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, tt.wantDBURL)
				}
			}
		})
	}
}
