package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr   string
	Provider   string
	BaseURL    string
	Model      string
	APIKey     string
	Timeout    time.Duration
	MaxTurns   int
	SessionTTL time.Duration
}

const (
	ProviderOpenAICompat = "openai_compat"
	ProviderMock         = "mock"
)

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: envOr("HTTP_ADDR", ":8082"),
		Provider: envOr("LLM_PROVIDER", ProviderOpenAICompat),
		BaseURL:  envOr("LLM_BASE_URL", "http://host.docker.internal:11434/v1"),
		Model:    envOr("LLM_MODEL", "qwen2.5:7b"),
		APIKey:   os.Getenv("LLM_API_KEY"),
	}

	switch cfg.Provider {
	case ProviderOpenAICompat, ProviderMock:
	default:
		return Config{}, fmt.Errorf("неизвестный LLM_PROVIDER %q: ожидается %s или %s",
			cfg.Provider, ProviderOpenAICompat, ProviderMock)
	}

	var err error
	if cfg.Timeout, err = durationOr("LLM_TIMEOUT", 60*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.SessionTTL, err = durationOr("DIALOG_SESSION_TTL", 30*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.MaxTurns, err = intOr("DIALOG_MAX_TURNS", 20); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationOr(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", key, raw, err)
	}
	return d, nil
}

func intOr(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", key, raw, err)
	}
	return n, nil
}
