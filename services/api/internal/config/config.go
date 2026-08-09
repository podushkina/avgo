package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	ProviderOpenAICompat = "openai_compat"
	ProviderMock         = "mock"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string

	LLMEnabled  bool
	LLMProvider string
	LLMBaseURL  string
	LLMModel    string
	LLMAPIKey   string
	LLMTimeout  time.Duration

	ExamMaxCycles  int
	ExamSessionTTL time.Duration
	MessageLimit   int
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    envOr("HTTP_ADDR", ":8081"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LLMProvider: envOr("LLM_PROVIDER", ProviderOpenAICompat),
		LLMBaseURL:  envOr("LLM_BASE_URL", "http://host.docker.internal:11434/v1"),
		LLMModel:    envOr("LLM_MODEL", "qwen2.5:7b"),
		LLMAPIKey:   os.Getenv("LLM_API_KEY"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL не задан")
	}

	switch cfg.LLMProvider {
	case ProviderOpenAICompat, ProviderMock:
	default:
		return Config{}, fmt.Errorf("неизвестный LLM_PROVIDER %q: ожидается %s или %s",
			cfg.LLMProvider, ProviderOpenAICompat, ProviderMock)
	}

	var err error
	if cfg.LLMEnabled, err = boolOr("LLM_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.LLMTimeout, err = durationOr("LLM_TIMEOUT", 30*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.ExamSessionTTL, err = durationOr("EXAM_SESSION_TTL", 24*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.ExamMaxCycles, err = intOr("EXAM_MAX_CYCLES", 8); err != nil {
		return Config{}, err
	}
	if cfg.ExamMaxCycles < 1 {
		return Config{}, fmt.Errorf("EXAM_MAX_CYCLES должен быть больше нуля, получено %d", cfg.ExamMaxCycles)
	}
	if cfg.MessageLimit, err = intOr("EXAM_MESSAGE_LIMIT", 1000); err != nil {
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

func boolOr(key string, fallback bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s=%q: %w", key, raw, err)
	}
	return b, nil
}
