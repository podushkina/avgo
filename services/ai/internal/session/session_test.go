package session

import (
	"errors"
	"testing"
	"time"

	"github.com/avito-antifraud/ai/internal/llm"
	"github.com/avito-antifraud/ai/internal/prompt"
)

func newTestStore(ttl time.Duration, maxTurns int) (*Store, *time.Time) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	s := NewStore(ttl, maxTurns)
	s.now = func() time.Time { return now }
	return s, &now
}

func TestCreateSeedsSystemAndOpening(t *testing.T) {
	s, _ := newTestStore(time.Hour, 10)
	sess := s.Create(prompt.RoleSeller, prompt.DifficultyMedium)

	if len(sess.Messages) != 2 {
		t.Fatalf("сообщений = %d, ожидалось 2", len(sess.Messages))
	}
	if sess.Messages[0].Role != llm.RoleSystem {
		t.Errorf("первое сообщение должно быть системным, получено %q", sess.Messages[0].Role)
	}
	if sess.Messages[1].Role != llm.RoleAssistant {
		t.Errorf("второе сообщение должно быть от мошенника, получено %q", sess.Messages[1].Role)
	}
}

func TestGetReturnsNotFoundForUnknownID(t *testing.T) {
	s, _ := newTestStore(time.Hour, 10)
	if _, err := s.Get("нет-такой"); !errors.Is(err, ErrNotFound) {
		t.Errorf("ошибка = %v, ожидалась ErrNotFound", err)
	}
}

func TestExpiredSessionIsGoneOnGet(t *testing.T) {
	s, now := newTestStore(30*time.Minute, 10)
	sess := s.Create(prompt.RoleBuyer, prompt.DifficultyEasy)

	*now = now.Add(31 * time.Minute)
	if _, err := s.Get(sess.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("протухшая сессия должна быть недоступна, получено %v", err)
	}
	if s.Len() != 0 {
		t.Errorf("протухшая сессия должна удаляться, осталось %d", s.Len())
	}
}

func TestActivityExtendsLifetime(t *testing.T) {
	s, now := newTestStore(30*time.Minute, 10)
	sess := s.Create(prompt.RoleBuyer, prompt.DifficultyEasy)

	*now = now.Add(20 * time.Minute)
	if _, err := s.AppendUser(sess.ID, "привет"); err != nil {
		t.Fatalf("AppendUser: %v", err)
	}

	*now = now.Add(20 * time.Minute)
	if _, err := s.Get(sess.ID); err != nil {
		t.Errorf("активность должна продлевать жизнь сессии, получено %v", err)
	}
}

func TestTurnLimitIsEnforced(t *testing.T) {
	s, _ := newTestStore(time.Hour, 2)
	sess := s.Create(prompt.RoleSeller, prompt.DifficultyHard)

	for i := 0; i < 2; i++ {
		if _, err := s.AppendUser(sess.ID, "реплика"); err != nil {
			t.Fatalf("реплика %d: %v", i+1, err)
		}
	}
	if _, err := s.AppendUser(sess.ID, "лишняя"); !errors.Is(err, ErrTurnLimit) {
		t.Errorf("ошибка = %v, ожидалась ErrTurnLimit", err)
	}
}

func TestEvictRemovesOnlyExpired(t *testing.T) {
	s, now := newTestStore(30*time.Minute, 10)
	old := s.Create(prompt.RoleBuyer, prompt.DifficultyEasy)

	*now = now.Add(31 * time.Minute)
	fresh := s.Create(prompt.RoleSeller, prompt.DifficultyEasy)

	if removed := s.evict(); removed != 1 {
		t.Errorf("удалено %d сессий, ожидалась 1", removed)
	}
	if _, err := s.Get(fresh.ID); err != nil {
		t.Errorf("свежая сессия не должна удаляться: %v", err)
	}
	if _, err := s.Get(old.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("старая сессия должна быть удалена, получено %v", err)
	}
}

func TestHistoryIsACopy(t *testing.T) {
	s, _ := newTestStore(time.Hour, 10)
	sess := s.Create(prompt.RoleSeller, prompt.DifficultyMedium)

	h := sess.History()
	h[0].Content = "подменено"

	if sess.Messages[0].Content == "подменено" {
		t.Error("History должна возвращать копию, иначе вызывающий код портит промпт")
	}
}
