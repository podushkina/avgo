package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-antifraud/api/internal/domain"
)

var ErrNotFound = errors.New("не найдено")

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type User struct {
	ID         uuid.UUID `json:"id"`
	ExternalID string    `json:"external_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) EnsureUser(ctx context.Context, externalID string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (id, external_id)
		VALUES ($1, $2)
		ON CONFLICT (external_id) DO UPDATE SET external_id = EXCLUDED.external_id
		RETURNING id, external_id, created_at`,
		uuid.New(), externalID,
	).Scan(&u.ID, &u.ExternalID, &u.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("создание пользователя: %w", err)
	}
	return u, nil
}

func (s *Store) ScenariosByRole(ctx context.Context, role domain.Role) ([]domain.Scenario, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, role, order_index, title, situation, question,
		       options, correct_option, explanation, red_flags
		FROM scenarios
		WHERE role = $1
		ORDER BY order_index`, string(role))
	if err != nil {
		return nil, fmt.Errorf("выборка сценариев: %w", err)
	}
	defer rows.Close()

	var out []domain.Scenario
	for rows.Next() {
		var (
			sc       domain.Scenario
			roleStr  string
			opts     []byte
			redFlags []byte
		)
		if err := rows.Scan(&sc.ID, &roleStr, &sc.OrderIndex, &sc.Title, &sc.Situation,
			&sc.Question, &opts, &sc.CorrectOption, &sc.Explanation, &redFlags); err != nil {
			return nil, fmt.Errorf("чтение сценария: %w", err)
		}
		if err := json.Unmarshal(opts, &sc.Options); err != nil {
			return nil, fmt.Errorf("разбор вариантов сценария %d: %w", sc.ID, err)
		}
		if err := json.Unmarshal(redFlags, &sc.RedFlags); err != nil {
			return nil, fmt.Errorf("разбор признаков сценария %d: %w", sc.ID, err)
		}
		sc.Role = domain.Role(roleStr)
		out = append(out, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("обход сценариев: %w", err)
	}
	return out, nil
}

func (s *Store) ScenarioByID(ctx context.Context, id int) (domain.Scenario, error) {
	var (
		sc       domain.Scenario
		roleStr  string
		opts     []byte
		redFlags []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, role, order_index, title, situation, question,
		       options, correct_option, explanation, red_flags
		FROM scenarios WHERE id = $1`, id,
	).Scan(&sc.ID, &roleStr, &sc.OrderIndex, &sc.Title, &sc.Situation,
		&sc.Question, &opts, &sc.CorrectOption, &sc.Explanation, &redFlags)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Scenario{}, ErrNotFound
	}
	if err != nil {
		return domain.Scenario{}, fmt.Errorf("выборка сценария: %w", err)
	}
	if err := json.Unmarshal(opts, &sc.Options); err != nil {
		return domain.Scenario{}, fmt.Errorf("разбор вариантов: %w", err)
	}
	if err := json.Unmarshal(redFlags, &sc.RedFlags); err != nil {
		return domain.Scenario{}, fmt.Errorf("разбор признаков: %w", err)
	}
	sc.Role = domain.Role(roleStr)
	return sc, nil
}

type ProgressEntry struct {
	ID           uuid.UUID       `json:"id"`
	Role         domain.Role     `json:"role"`
	CorrectCount int             `json:"correct_count"`
	TotalCount   int             `json:"total_count"`
	Percent      float64         `json:"percent"`
	Score        int             `json:"score"`
	Level        string          `json:"level"`
	Mistakes     []domain.Review `json:"mistakes"`
	CompletedAt  time.Time       `json:"completed_at"`
}

func (s *Store) SaveProgress(
	ctx context.Context, userID uuid.UUID, role domain.Role, res domain.Result,
) (ProgressEntry, error) {
	mistakes, err := json.Marshal(res.Mistakes)
	if err != nil {
		return ProgressEntry{}, fmt.Errorf("сериализация разбора ошибок: %w", err)
	}

	entry := ProgressEntry{
		ID:           uuid.New(),
		Role:         role,
		CorrectCount: res.Correct,
		TotalCount:   res.Total,
		Percent:      res.Percent,
		Score:        res.Score,
		Level:        res.Level,
		Mistakes:     res.Mistakes,
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO progress (id, user_id, role, correct_count, total_count, percent, score, answers)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING completed_at`,
		entry.ID, userID, string(role), res.Correct, res.Total, res.Percent, res.Score, mistakes,
	).Scan(&entry.CompletedAt)
	if err != nil {
		return ProgressEntry{}, fmt.Errorf("сохранение прогресса: %w", err)
	}
	return entry, nil
}

func (s *Store) ProgressByUser(ctx context.Context, userID uuid.UUID) ([]ProgressEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, role, correct_count, total_count, percent, score, answers, completed_at
		FROM progress
		WHERE user_id = $1
		ORDER BY completed_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("выборка прогресса: %w", err)
	}
	defer rows.Close()

	out := []ProgressEntry{}
	for rows.Next() {
		var (
			e        ProgressEntry
			roleStr  string
			mistakes []byte
		)
		if err := rows.Scan(&e.ID, &roleStr, &e.CorrectCount, &e.TotalCount,
			&e.Percent, &e.Score, &mistakes, &e.CompletedAt); err != nil {
			return nil, fmt.Errorf("чтение прогресса: %w", err)
		}
		if err := json.Unmarshal(mistakes, &e.Mistakes); err != nil {
			return nil, fmt.Errorf("разбор ошибок попытки %s: %w", e.ID, err)
		}
		e.Role = domain.Role(roleStr)
		e.Level = domain.LevelFor(e.Percent)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("обход прогресса: %w", err)
	}
	return out, nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
