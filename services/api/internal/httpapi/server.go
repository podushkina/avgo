package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/avito-antifraud/api/internal/domain"
	"github.com/avito-antifraud/api/internal/storage"
)

type AntiFraudStore interface {
	EnsureUser(ctx context.Context, externalID string) (storage.User, error)
	ScenariosByRole(ctx context.Context, role domain.Role) ([]domain.Scenario, error)
	ScenarioByID(ctx context.Context, id int) (domain.Scenario, error)
	SaveProgress(ctx context.Context, userID uuid.UUID, role domain.Role, res domain.Result) (storage.ProgressEntry, error)
	ProgressByUser(ctx context.Context, userID uuid.UUID) ([]storage.ProgressEntry, error)
	Ping(ctx context.Context) error

	StartAttempt(ctx context.Context, userID uuid.UUID, role domain.Role) (uuid.UUID, error)
	RecordAnswer(ctx context.Context, attemptID uuid.UUID, scenarioID int, optionID int) (int, error)
	GetAttemptAnswers(ctx context.Context, attemptID uuid.UUID) ([]domain.Answer, domain.Role, uuid.UUID, error)
	MarkAttemptCompleted(ctx context.Context, attemptID uuid.UUID) error
}

type Server struct {
	store AntiFraudStore
	log   *slog.Logger
}

func NewServer(store AntiFraudStore, log *slog.Logger) *Server {
	return &Server{store: store, log: log}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/healthz", s.handleHealth)
	mux.HandleFunc("POST /api/users", s.handleCreateUser)
	mux.HandleFunc("GET /api/scenarios", s.handleScenarios)

	mux.HandleFunc("POST /api/attempts/start", s.handleStartAttempt)
	mux.HandleFunc("POST /api/attempts/{id}/check", s.handleCheckAnswer)
	mux.HandleFunc("POST /api/attempts/{id}/finish", s.handleFinishAttempt)

	mux.HandleFunc("GET /api/progress", s.handleProgress)

	limiter := NewRateLimiter(5, 10)

	return limiter.Middleware(logging(s.log, mux))
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

func (s *Server) handleStartAttempt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
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

	attemptID, err := s.store.StartAttempt(r.Context(), userID, role)
	if err != nil {
		s.fail(w, "старт попытки", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"attempt_id": attemptID.String()})
}

func (s *Server) handleCheckAnswer(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный attempt_id")
		return
	}

	var req struct {
		ScenarioID int `json:"scenario_id"`
		Option     int `json:"option"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	savedOption, err := s.store.RecordAnswer(r.Context(), attemptID, req.ScenarioID, req.Option)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sc, err := s.store.ScenarioByID(r.Context(), req.ScenarioID)
	if err != nil {
		s.fail(w, "выборка сценария", err)
		return
	}

	picked, _ := sc.Option(savedOption)
	correct, _ := sc.Option(sc.CorrectOption)

	writeJSON(w, http.StatusOK, map[string]any{
		"is_correct":          savedOption == sc.CorrectOption,
		"your_verdict":        picked.Verdict,
		"your_outcome":        picked.Outcome,
		"points":              picked.Verdict.Points(),
		"correct_option":      sc.CorrectOption,
		"correct_option_text": correct.Text,
		"correct_outcome":     correct.Outcome,
		"explanation":         sc.Explanation,
		"red_flags":           sc.RedFlags,
		"was_overwritten":     savedOption != req.Option,
	})
}

func (s *Server) handleFinishAttempt(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный attempt_id")
		return
	}

	answers, role, userID, err := s.store.GetAttemptAnswers(r.Context(), attemptID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "попытка не найдена или уже завершена")
		return
	}

	scenarios, err := s.store.ScenariosByRole(r.Context(), role)
	if err != nil {
		s.fail(w, "выборка сценариев", err)
		return
	}

	result := domain.Score(scenarios, answers)

	entry, err := s.store.SaveProgress(r.Context(), userID, role, result)
	if err != nil {
		s.fail(w, "сохранение прогресса", err)
		return
	}

	_ = s.store.MarkAttemptCompleted(r.Context(), attemptID)

	writeJSON(w, http.StatusOK, map[string]any{
		"progress_id":          entry.ID,
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
