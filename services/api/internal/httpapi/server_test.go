package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/avito-antifraud/api/internal/domain"
	"github.com/avito-antifraud/api/internal/httpapi"
	"github.com/avito-antifraud/api/internal/storage"
)

type MockStore struct {
	scenarios map[int]domain.Scenario
	roleScen  map[domain.Role][]domain.Scenario
}

func newMockStore() *MockStore {
	sc := domain.Scenario{
		ID:            101,
		Title:         "Секретный фишинг",
		Role:          domain.RoleBuyer,
		CorrectOption: 0,
		Options: []domain.Option{
			{Text: "Безопасно", Verdict: domain.VerdictSafe, Outcome: "Всё ок"},
			{Text: "Опасно", Verdict: domain.VerdictDangerous, Outcome: "Скам"},
		},
		Explanation: "Тестовое объяснение",
		RedFlags:    []string{"fake_url"},
	}

	return &MockStore{
		scenarios: map[int]domain.Scenario{101: sc},
		roleScen:  map[domain.Role][]domain.Scenario{domain.RoleBuyer: {sc}},
	}
}

func (m *MockStore) Ping(ctx context.Context) error { return nil }
func (m *MockStore) EnsureUser(ctx context.Context, extID string) (storage.User, error) {
	return storage.User{ID: uuid.New(), ExternalID: extID}, nil
}
func (m *MockStore) ScenariosByRole(ctx context.Context, role domain.Role) ([]domain.Scenario, error) {
	return m.roleScen[role], nil
}
func (m *MockStore) ScenarioByID(ctx context.Context, id int) (domain.Scenario, error) {
	sc, ok := m.scenarios[id]
	if !ok {
		return domain.Scenario{}, errors.New("not found")
	}
	return sc, nil
}
func (m *MockStore) SaveProgress(ctx context.Context, uID uuid.UUID, r domain.Role, res domain.Result) (storage.ProgressEntry, error) {
	return storage.ProgressEntry{
		ID:          uuid.New(),
		Role:        r,
		CompletedAt: time.Now(),
	}, nil
}
func (m *MockStore) ProgressByUser(ctx context.Context, uID uuid.UUID) ([]storage.ProgressEntry, error) {
	return nil, nil
}
func (m *MockStore) ProgressByUserPaginated(ctx context.Context, uID uuid.UUID, l, o int) ([]storage.ProgressEntry, int, error) {
	return nil, 0, nil
}
func (m *MockStore) StartAttempt(ctx context.Context, uID uuid.UUID, r domain.Role) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil
}
func (m *MockStore) RecordAnswer(ctx context.Context, attID uuid.UUID, scID, optID int) (int, error) {
	return optID, nil
}
func (m *MockStore) GetAttemptAnswers(ctx context.Context, attID uuid.UUID) ([]domain.Answer, domain.Role, uuid.UUID, error) {
	return []domain.Answer{{ScenarioID: 101, Option: 0}}, domain.RoleBuyer, uuid.New(), nil
}
func (m *MockStore) MarkAttemptCompleted(ctx context.Context, attID uuid.UUID) error { return nil }

func TestHandleScenarios_HidesCorrectOption(t *testing.T) {
	mock := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpapi.NewServer(mock, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/scenarios?role=buyer", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	rawJSON := rec.Body.String()
	if bytes.Contains([]byte(rawJSON), []byte("correct_option")) {
		t.Errorf("found correct_option in public API response: %s", rawJSON)
	}
}

func TestHandleStartAttempt_Validation(t *testing.T) {
	mock := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httpapi.NewServer(mock, logger)

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name:       "success",
			body:       map[string]string{"user_id": uuid.New().String(), "role": "buyer"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid_uuid",
			body:       map[string]string{"user_id": "invalid-uuid", "role": "buyer"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid_role",
			body:       map[string]string{"user_id": uuid.New().String(), "role": "admin"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/attempts/start", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			server.Routes().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.wantStatus, rec.Code)
			}
		})
	}
}
