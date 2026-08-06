package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/avito-antifraud/ai/internal/analysis"
	"github.com/avito-antifraud/ai/internal/llm"
	"github.com/avito-antifraud/ai/internal/prompt"
	"github.com/avito-antifraud/ai/internal/sanitize"
	"github.com/avito-antifraud/ai/internal/session"
)

const maxMessageLen = 2000

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
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"sessions": s.sessions.Len(),
	})
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role       string `json:"role"`
		Difficulty string `json:"difficulty"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	role, err := prompt.ParseRole(req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	difficulty, err := prompt.ParseDifficulty(req.Difficulty)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sess := s.sessions.Create(role, difficulty)
	writeJSON(w, http.StatusCreated, map[string]any{
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
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
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
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "текст сообщения пуст")
		return
	}
	if len([]rune(req.Text)) > maxMessageLen {
		writeError(w, http.StatusBadRequest, "сообщение слишком длинное")
		return
	}

	id := r.PathValue("id")
	sess, err := s.sessions.AppendUser(id, req.Text)
	switch {
	case errors.Is(err, session.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
		return
	case errors.Is(err, session.ErrTurnLimit):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		s.log.Error("добавление реплики", "error", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервиса")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "стриминг не поддерживается")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var full strings.Builder
	streamer := sanitize.NewStreamer(func(chunk string) error {
		full.WriteString(chunk)
		if err := writeEvent(w, "token", map[string]string{"text": chunk}); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})

	history := append(sess.History(), llm.Message{
		Role:    llm.RoleSystem,
		Content: prompt.TurnDirective(sess.Difficulty, sess.UserTurns),
	})

	err = s.client.Stream(r.Context(), history, streamer.Push)
	if errors.Is(err, sanitize.ErrRepeat) {
		s.log.Warn("ответ обрезан: модель начала повторяться", "session", id)
		err = nil
	}
	if err == nil {
		err = streamer.Close()
	}
	if err != nil {
		s.log.Error("ответ модели", "error", err)
		_ = writeEvent(w, "error", map[string]string{
			"error": "не удалось получить ответ модели: " + err.Error(),
		})
		flusher.Flush()
		return
	}

	reply := sanitize.TrimToSentence(full.String())
	s.sessions.AppendAssistant(id, reply)

	_ = writeEvent(w, "done", map[string]any{"text": reply})
	flusher.Flush()
}

func (s *Server) handleFinish(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessions.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	report := analysis.Analyze(sess.History())
	writeJSON(w, http.StatusOK, map[string]any{
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

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
