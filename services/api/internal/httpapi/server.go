package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/avito-antifraud/api/internal/domain"
	"github.com/avito-antifraud/api/internal/storage"
)

type Server struct {
	store *storage.Store
	log   *slog.Logger
}

func NewServer(store *storage.Store, log *slog.Logger) *Server {
	return &Server{store: store, log: log}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/healthz", s.handleHealth)
	mux.HandleFunc("POST /api/users", s.handleCreateUser)
	mux.HandleFunc("GET /api/scenarios", s.handleScenarios)
	mux.HandleFunc("POST /api/scenarios/{id}/check", s.handleCheckAnswer)
	mux.HandleFunc("POST /api/attempts", s.handleSubmitAttempt)
	mux.HandleFunc("GET /api/progress", s.handleProgress)
	return logging(s.log, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "база данных недоступна")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExternalID string `json:"external_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "external_id обязателен")
		return
	}

	user, err := s.store.EnsureUser(r.Context(), req.ExternalID)
	if err != nil {
		s.fail(w, "создание пользователя", err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleScenarios(w http.ResponseWriter, r *http.Request) {
	role, err := domain.ParseRole(r.URL.Query().Get("role"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	scenarios, err := s.store.ScenariosByRole(r.Context(), role)
	if err != nil {
		s.fail(w, "выборка сценариев", err)
		return
	}

	out := make([]domain.PublicScenario, 0, len(scenarios))
	for _, sc := range scenarios {
		out = append(out, sc.Public())
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCheckAnswer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный идентификатор сценария")
		return
	}

	var req struct {
		Option int `json:"option"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	sc, err := s.store.ScenarioByID(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "сценарий не найден")
		return
	}
	if err != nil {
		s.fail(w, "выборка сценария", err)
		return
	}

	picked, valid := sc.Option(req.Option)
	if !valid {
		writeError(w, http.StatusBadRequest, "вариант вне диапазона")
		return
	}
	correct, _ := sc.Option(sc.CorrectOption)

	writeJSON(w, http.StatusOK, map[string]any{
		"is_correct":          req.Option == sc.CorrectOption,
		"your_verdict":        picked.Verdict,
		"your_outcome":        picked.Outcome,
		"points":              picked.Verdict.Points(),
		"correct_option":      sc.CorrectOption,
		"correct_option_text": correct.Text,
		"correct_outcome":     correct.Outcome,
		"explanation":         sc.Explanation,
		"red_flags":           sc.RedFlags,
	})
}

func (s *Server) handleSubmitAttempt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string          `json:"user_id"`
		Role    string          `json:"role"`
		Answers []domain.Answer `json:"answers"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный user_id")
		return
	}
	role, err := domain.ParseRole(req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	scenarios, err := s.store.ScenariosByRole(r.Context(), role)
	if err != nil {
		s.fail(w, "выборка сценариев", err)
		return
	}
	if len(scenarios) == 0 {
		writeError(w, http.StatusConflict, "для этой роли не заведены сценарии")
		return
	}

	result := domain.Score(scenarios, req.Answers)

	entry, err := s.store.SaveProgress(r.Context(), userID, role, result)
	if err != nil {
		s.fail(w, "сохранение прогресса", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"attempt_id":           entry.ID,
		"role":                 role,
		"correct":              result.Correct,
		"total":                result.Total,
		"percent":              result.Percent,
		"score":                result.Score,
		"max_score":            result.MaxScore,
		"level":                result.Level,
		"perfect":              result.Perfect,
		"reviews":              result.Reviews,
		"mistakes":             result.Mistakes,
		"missed_red_flags":     result.RedFlags,
		"suggested_difficulty": domain.SuggestDifficulty(result.Percent),
		"completed_at":         entry.CompletedAt,
	})
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный user_id")
		return
	}

	entries, err := s.store.ProgressByUser(r.Context(), userID)
	if err != nil {
		s.fail(w, "выборка прогресса", err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) fail(w http.ResponseWriter, op string, err error) {
	s.log.Error(op, "error", err)
	writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервиса")
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

func logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if r.URL.Path != "/healthz" {
			log.Info("запрос", "method", r.Method, "path", r.URL.Path)
		}
	})
}
