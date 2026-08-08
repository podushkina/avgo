package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/avito-antifraud/ai/internal/analysis"
	"github.com/avito-antifraud/ai/internal/llm"
	"github.com/avito-antifraud/ai/internal/prompt"
	"github.com/avito-antifraud/ai/internal/sanitize"
	"github.com/avito-antifraud/ai/internal/session"
	"github.com/avito-antifraud/httpx"
)

const (
	maxMessageLen     = 2000
	inactivityTimeout = 15 * time.Second
)

type Server struct {
	client   llm.Client
	sessions *session.Store
	log      *slog.Logger
	maxTurns int
}

func NewServer(client llm.Client, sessions *session.Store, log *slog.Logger, maxTurns int) *Server {
	return &Server{client: client, sessions: sessions, log: log, maxTurns: maxTurns}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/dialog/healthz", s.handleHealth)
	mux.HandleFunc("POST /api/dialog/sessions", s.handleCreateSession)
	mux.HandleFunc("GET /api/dialog/sessions/{id}", s.handleGetSession)
	mux.HandleFunc("POST /api/dialog/sessions/{id}/messages", s.handleMessage)
	mux.HandleFunc("POST /api/dialog/sessions/{id}/finish", s.handleFinish)

	return httpx.TimeoutAndLog(s.log, 120*time.Second)(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"sessions": s.sessions.Len(),
	})
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role       string `json:"role"`
		Difficulty string `json:"difficulty"`
	}
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	role, err := prompt.ParseRole(req.Role)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	difficulty, err := prompt.ParseDifficulty(req.Difficulty)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	sess := s.sessions.Create(role, difficulty)
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"session_id":      sess.ID,
		"role":            sess.Role,
		"difficulty":      sess.Difficulty,
		"opening_message": prompt.Opening(role),
		"max_turns":       s.maxTurns,
	})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessions.Get(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"session_id": sess.ID,
		"role":       sess.Role,
		"difficulty": sess.Difficulty,
		"turns_used": sess.UserTurns,
		"max_turns":  s.maxTurns,
		"messages":   visible(sess.History()),
	})
}

func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		httpx.WriteError(w, http.StatusBadRequest, "текст сообщения пуст")
		return
	}
	if len([]rune(req.Text)) > maxMessageLen {
		httpx.WriteError(w, http.StatusBadRequest, "сообщение слишком длинное")
		return
	}

	id := r.PathValue("id")
	sess, err := s.sessions.AppendUser(id, req.Text)
	switch {
	case errors.Is(err, session.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	case errors.Is(err, session.ErrTurnLimit):
		httpx.WriteError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		s.log.Error("добавление реплики", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "внутренняя ошибка сервиса")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "стриминг не поддерживается")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	history := append(sess.History(), llm.Message{
		Role:    llm.RoleSystem,
		Content: prompt.TurnDirective(sess.Difficulty, sess.UserTurns),
	})

	type chunkResult struct {
		chunk string
		err   error
	}

	chunks := make(chan chunkResult)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		defer close(chunks)
		var full strings.Builder
		streamer := sanitize.NewStreamer(func(chunk string) error {
			full.WriteString(chunk)
			select {
			case chunks <- chunkResult{chunk: chunk}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

		streamErr := s.client.Stream(ctx, history, streamer.Push)
		if errors.Is(streamErr, sanitize.ErrRepeat) {
			s.log.Warn("ответ обрезан: модель начала повторяться", "session", id)
			streamErr = nil
		}
		if streamErr == nil {
			streamErr = streamer.Close()
		}

		select {
		case chunks <- chunkResult{err: streamErr}:
		case <-ctx.Done():
		}
	}()

	var fullResponse strings.Builder
	timer := time.NewTimer(inactivityTimeout)
	defer timer.Stop()

	for {
		timer.Reset(inactivityTimeout)
		select {
		case <-r.Context().Done():
			return
		case <-timer.C:
			s.log.Error("зависание генерации: не было токенов в течение 15 сек", "session", id)
			_ = writeEvent(w, "error", map[string]string{
				"error": "превышено время ожидания ответа от модели (inactivity timeout)",
			})
			flusher.Flush()
			return
		case res, ok := <-chunks:
			if !ok {
				return
			}
			if res.err != nil {
				s.log.Error("ошибка стриминга модели", "error", res.err)
				_ = writeEvent(w, "error", map[string]string{
					"error": "не удалось получить ответ модели: " + res.err.Error(),
				})
				flusher.Flush()
				return
			}
			if res.chunk == "" {
				// Завершение потока без ошибок
				reply := sanitize.TrimToSentence(fullResponse.String())
				s.sessions.AppendAssistant(id, reply)
				_ = writeEvent(w, "done", map[string]any{"text": reply})
				flusher.Flush()
				return
			}

			fullResponse.WriteString(res.chunk)
			if err := writeEvent(w, "token", map[string]string{"text": res.chunk}); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleFinish(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessions.Get(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	report := analysis.Analyze(sess.History())
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"session_id": sess.ID,
		"role":       sess.Role,
		"difficulty": sess.Difficulty,
		"report":     report,
	})
}

func visible(msgs []llm.Message) []llm.Message {
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == llm.RoleSystem {
			continue
		}
		out = append(out, m)
	}
	return out
}

func writeEvent(w http.ResponseWriter, event string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body)
	return err
}
