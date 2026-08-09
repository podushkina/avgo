package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/avito-antifraud/api/internal/apierr"
	"github.com/avito-antifraud/api/internal/config"
	"github.com/avito-antifraud/api/internal/exam"
	"github.com/avito-antifraud/api/internal/llm"
	"github.com/avito-antifraud/api/internal/storage"
)

const (
	sessionCookie = "antiscam_session"
	cookieMaxAge  = 180 * 24 * 60 * 60
)

type Server struct {
	store      *storage.Store
	client     llm.Client
	classifier *exam.Classifier
	reviewer   *exam.Reviewer
	cfg        config.Config
	log        *slog.Logger
	msgLimiter *RateLimiter
}

func NewServer(store *storage.Store, client llm.Client, cfg config.Config, log *slog.Logger) *Server {
	return &Server{
		store:      store,
		client:     client,
		classifier: exam.NewClassifier(client, log),
		reviewer:   exam.NewReviewer(client, log),
		cfg:        cfg,
		log:        log,
		msgLimiter: NewRateLimiter(20.0/60.0, 20),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	routes := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodGet, "/healthz", s.handleHealth},
		{http.MethodGet, "/me", s.handleMe},
		{http.MethodPost, "/users", s.handleCreateUser},
		{http.MethodPost, "/progress/reset", s.handleResetProgress},
		{http.MethodPost, "/progress/reset/{role}", s.handleResetProgress},
		{http.MethodGet, "/training/current-step", s.handleCurrentStep},
		{http.MethodPost, "/training/answer", s.handleAnswer},
		{http.MethodGet, "/exam/start", s.handleExamStart},
		{http.MethodPost, "/exam/start", s.handleExamStart},
		{http.MethodPost, "/exam/restart", s.handleExamRestart},
		{http.MethodPost, "/exam/message", s.handleExamMessage},
		{http.MethodPost, "/exam/finish", s.handleExamFinish},
		{http.MethodGet, "/results", s.handleResults},
		{http.MethodPost, "/results", s.handleResults},
	}

	for _, r := range routes {
		for _, prefix := range []string{"/api/v1", "/api"} {
			mux.HandleFunc(r.method+" "+prefix+r.path, r.handler)
		}
	}
	mux.HandleFunc("GET /healthz", s.handleHealth)

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		apierr.Write(w, apierr.New(http.StatusServiceUnavailable,
			apierr.CodeInternal, "база данных недоступна"))
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *Server) setCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   cookieMaxAge,
	})
}

func (s *Server) token(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func (s *Server) currentUser(r *http.Request) (storage.User, error) {
	token := s.token(r)
	if token == "" {
		return storage.User{}, apierr.ErrUserNotFound
	}
	u, err := s.store.UserByToken(r.Context(), token)
	if errors.Is(err, storage.ErrNotFound) {
		return storage.User{}, apierr.ErrUserNotFound
	}
	if err != nil {
		s.log.Error("выборка пользователя", "error", err)
		return storage.User{}, apierr.ErrInternal
	}
	return u, nil
}

func (s *Server) roleFrom(r *http.Request, body string) (storage.Role, error) {
	raw := body
	if raw == "" {
		raw = r.PathValue("role")
	}
	if raw == "" {
		raw = r.URL.Query().Get("role")
	}
	role, err := storage.ParseRole(raw)
	if err != nil {
		return "", apierr.BadRequest(err.Error())
	}
	return role, nil
}

func (s *Server) fail(w http.ResponseWriter, op string, err error) {
	var apiErr *apierr.Error
	if errors.As(err, &apiErr) {
		apierr.Write(w, apiErr)
		return
	}
	s.log.Error(op, "error", err)
	apierr.Write(w, apierr.ErrInternal)
}

func (s *Server) progressPair(
	ctx context.Context, userID uuid.UUID,
) (buyer, seller any, err error) {
	b, err := s.store.Progress(ctx, userID, storage.RoleBuyer)
	if err != nil {
		return nil, nil, err
	}
	sl, err := s.store.Progress(ctx, userID, storage.RoleSeller)
	if err != nil {
		return nil, nil, err
	}
	return b, sl, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil || r.ContentLength == 0 {
		return true
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(dst); err != nil {
		apierr.Write(w, apierr.BadRequest("некорректный JSON: "+err.Error()))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
