package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/avito-antifraud/api/internal/config"
	"github.com/avito-antifraud/api/internal/httpapi"
	"github.com/avito-antifraud/api/internal/httpx"
	"github.com/avito-antifraud/api/internal/llm"
	"github.com/avito-antifraud/api/internal/migrations"
	"github.com/avito-antifraud/api/internal/storage"
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

	if err := migrate(ctx, cfg.DatabaseURL, log); err != nil {
		return fmt.Errorf("миграции: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("пул соединений: %w", err)
	}
	defer pool.Close()

	var client llm.Client
	switch {
	case !cfg.LLMEnabled:
		client = llm.NewScripted()
		log.Info("LLM выключен: используется сценарный стаб")
	case cfg.LLMProvider == config.ProviderMock:
		client = llm.NewMock()
		log.Info("используется mock-провайдер модели")
	default:
		client = llm.NewOpenAICompat(cfg.LLMBaseURL, cfg.LLMModel, cfg.LLMAPIKey, cfg.LLMTimeout)
		log.Info("провайдер модели", "base_url", cfg.LLMBaseURL, "model", cfg.LLMModel)
	}

	handler := httpapi.NewServer(storage.New(pool), client, cfg, log).Routes()
	handler = httpx.TimeoutAndLog(log, cfg.LLMTimeout+30*time.Second)(handler)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api-service слушает", "addr", cfg.HTTPAddr)
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

func migrate(ctx context.Context, dsn string, log *slog.Logger) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("открытие соединения: %w", err)
	}
	defer func() { _ = db.Close() }()

	const attempts = 15
	for i := 1; i <= attempts; i++ {
		if err = db.PingContext(ctx); err == nil {
			break
		}
		if i == attempts {
			return fmt.Errorf("база недоступна после %d попыток: %w", attempts, err)
		}
		log.Info("жду базу данных", "attempt", i)

		select {
		case <-ctx.Done():
			return fmt.Errorf("отмена ожидания базы данных: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("диалект goose: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("накат миграций: %w", err)
	}

	log.Info("миграции применены")
	return nil
}
