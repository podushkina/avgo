package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avito-antifraud/ai/internal/config"
	"github.com/avito-antifraud/ai/internal/httpapi"
	"github.com/avito-antifraud/ai/internal/llm"
	"github.com/avito-antifraud/ai/internal/session"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("сервис остановлен с ошибкой", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var client llm.Client
	if cfg.Provider == config.ProviderMock {
		client = llm.NewMock()
		log.Info("используется mock-провайдер: обращений к модели не будет")
	} else {
		client = llm.NewOpenAICompat(cfg.BaseURL, cfg.Model, cfg.APIKey, cfg.Timeout)
		log.Info("провайдер модели", "base_url", cfg.BaseURL, "model", cfg.Model)
	}

	store := session.NewStore(cfg.SessionTTL, cfg.MaxTurns)
	store.StartJanitor(ctx, time.Minute)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewServer(client, store, log, cfg.MaxTurns).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("ai-service слушает", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("получен сигнал остановки")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
