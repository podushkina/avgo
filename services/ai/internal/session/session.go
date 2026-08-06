package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/avito-antifraud/ai/internal/llm"
	"github.com/avito-antifraud/ai/internal/prompt"
)

var (
	ErrNotFound  = errors.New("сессия не найдена")
	ErrTurnLimit = errors.New("достигнут лимит реплик в сессии")
)

type Session struct {
	ID         string
	Role       prompt.Role
	Difficulty prompt.Difficulty
	Messages   []llm.Message
	UserTurns  int
	CreatedAt  time.Time
	LastSeen   time.Time
}

func (s *Session) History() []llm.Message {
	out := make([]llm.Message, len(s.Messages))
	copy(out, s.Messages)
	return out
}

type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	ttl      time.Duration
	maxTurns int
	now      func() time.Time
}

func NewStore(ttl time.Duration, maxTurns int) *Store {
	return &Store{
		sessions: make(map[string]*Session),
		ttl:      ttl,
		maxTurns: maxTurns,
		now:      time.Now,
	}
}

func (s *Store) Create(role prompt.Role, difficulty prompt.Difficulty) *Session {
	now := s.now()
	sess := &Session{
		ID:         uuid.NewString(),
		Role:       role,
		Difficulty: difficulty,
		CreatedAt:  now,
		LastSeen:   now,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: prompt.System(role, difficulty)},
			{Role: llm.RoleAssistant, Content: prompt.Opening(role)},
		},
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
	return sess
}

func (s *Store) Get(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	if s.now().Sub(sess.LastSeen) > s.ttl {
		delete(s.sessions, id)
		return nil, ErrNotFound
	}
	return sess, nil
}

func (s *Store) AppendUser(id, text string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	if sess.UserTurns >= s.maxTurns {
		return nil, ErrTurnLimit
	}

	sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleUser, Content: text})
	sess.UserTurns++
	sess.LastSeen = s.now()
	return sess, nil
}

func (s *Store) AppendAssistant(id, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return
	}
	sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleAssistant, Content: text})
	sess.LastSeen = s.now()
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

func (s *Store) evict() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	removed := 0
	for id, sess := range s.sessions {
		if now.Sub(sess.LastSeen) > s.ttl {
			delete(s.sessions, id)
			removed++
		}
	}
	return removed
}

func (s *Store) StartJanitor(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.evict()
			}
		}
	}()
}
