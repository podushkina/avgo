package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/avito-antifraud/ai/internal/httpapi"
	"github.com/avito-antifraud/ai/internal/llm"
	"github.com/avito-antifraud/ai/internal/session"
)

type MockLLMClient struct {
	tokens []string
	err    error
}

func (m *MockLLMClient) Stream(ctx context.Context, history []llm.Message, yield func(string) error) error {
	if m.err != nil {
		return m.err
	}
	for _, tok := range m.tokens {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := yield(tok); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestServer_CreateAndGetSession(t *testing.T) {
	mockLLM := &MockLLMClient{tokens: []string{"ok"}}
	store := session.NewStore(time.Hour, 5)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpapi.NewServer(mockLLM, store, logger, 5)

	t.Run("create session success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"role": "buyer", "difficulty": "easy"})
		req := httptest.NewRequest(http.MethodPost, "/api/dialog/sessions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", rec.Code)
		}
	})

	t.Run("create session invalid role", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"role": "unknown", "difficulty": "easy"})
		req := httptest.NewRequest(http.MethodPost, "/api/dialog/sessions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestServer_HandleMessage_SSE_Streaming(t *testing.T) {
	mockLLM := &MockLLMClient{
		tokens: []string{"Привет", ", ", "это ", "тест."},
	}
	store := session.NewStore(time.Hour, 5)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpapi.NewServer(mockLLM, store, logger, 5)

	sess := store.Create("buyer", "easy")

	body, _ := json.Marshal(map[string]string{"text": "Привет!"})
	req := httptest.NewRequest(http.MethodPost, "/api/dialog/sessions/"+sess.ID+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Errorf("expected Content-Type text/event-stream, got %s", contentType)
	}

	resBody := rec.Body.String()
	if !strings.Contains(resBody, "event: token") {
		t.Errorf("expected SSE token event in response, got: %s", resBody)
	}
	if !strings.Contains(resBody, "event: done") {
		t.Errorf("expected SSE done event in response, got: %s", resBody)
	}
}

func TestServer_HandleMessage_Validation(t *testing.T) {
	mockLLM := &MockLLMClient{tokens: []string{"ok"}}
	store := session.NewStore(time.Hour, 5)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpapi.NewServer(mockLLM, store, logger, 5)

	t.Run("empty message", func(t *testing.T) {
		sess := store.Create("buyer", "easy")
		body, _ := json.Marshal(map[string]string{"text": "   "})
		req := httptest.NewRequest(http.MethodPost, "/api/dialog/sessions/"+sess.ID+"/messages", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		server.Routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty text, got %d", rec.Code)
		}
	})

	t.Run("session not found", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"text": "Hello"})
		req := httptest.NewRequest(http.MethodPost, "/api/dialog/sessions/non-existent-id/messages", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		server.Routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for missing session, got %d", rec.Code)
		}
	})
}
