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

var (
	ErrNotFound     = errors.New("не найдено")
	ErrStepMismatch = errors.New("шаг не совпадает с текущим")
	ErrNoActive     = errors.New("активной сессии нет")
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

type User struct {
	ID        uuid.UUID
	AnonToken string
	Name      string
	AgeGroup  string
	Gender    string
	CreatedAt time.Time
}

func (s *Store) UserByToken(ctx context.Context, token string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, anon_token, name, age_group, gender, created_at
		FROM users WHERE anon_token = $1`, token,
	).Scan(&u.ID, &u.AnonToken, &u.Name, &u.AgeGroup, &u.Gender, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("выборка пользователя: %w", err)
	}
	return u, nil
}

func (s *Store) UpsertUser(ctx context.Context, token, name, ageGroup, gender string) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("начало транзакции: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var u User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (id, anon_token, name, age_group, gender)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (anon_token) DO UPDATE
			SET name = EXCLUDED.name,
			    age_group = EXCLUDED.age_group,
			    gender = EXCLUDED.gender
		RETURNING id, anon_token, name, age_group, gender, created_at`,
		uuid.New(), token, name, ageGroup, gender,
	).Scan(&u.ID, &u.AnonToken, &u.Name, &u.AgeGroup, &u.Gender, &u.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("создание пользователя: %w", err)
	}

	for _, role := range []string{string(RoleBuyer), string(RoleSeller)} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO role_progress (user_id, role) VALUES ($1, $2)
			ON CONFLICT (user_id, role) DO NOTHING`, u.ID, role); err != nil {
			return User{}, fmt.Errorf("создание прогресса: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("коммит: %w", err)
	}
	return u, nil
}

type Role string

const (
	RoleBuyer  Role = "buyer"
	RoleSeller Role = "seller"
)

func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleBuyer:
		return RoleBuyer, nil
	case RoleSeller:
		return RoleSeller, nil
	default:
		return "", fmt.Errorf("неизвестная роль %q: ожидается buyer или seller", s)
	}
}

func (s *Store) TotalSteps(ctx context.Context, role Role) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM training_steps WHERE role = $1`, string(role)).Scan(&n); err != nil {
		return 0, fmt.Errorf("подсчёт шагов: %w", err)
	}
	return n, nil
}

// RoleState — внутреннее состояние роли: публичный status плюс указатель
// обучения, который наружу не отдаётся.
type RoleState struct {
	Status      domain.Status
	CurrentStep int
	TotalSteps  int
}

func (s RoleState) Public() domain.RoleProgress {
	return domain.RoleProgress{Status: s.Status}
}

func (s *Store) Progress(ctx context.Context, userID uuid.UUID, role Role) (RoleState, error) {
	var (
		status string
		step   int
	)
	err := s.pool.QueryRow(ctx, `
		SELECT status, current_step FROM role_progress WHERE user_id = $1 AND role = $2`,
		userID, string(role)).Scan(&status, &step)
	if errors.Is(err, pgx.ErrNoRows) {
		status, step = string(domain.StatusNotStarted), 0
	} else if err != nil {
		return RoleState{}, fmt.Errorf("выборка прогресса: %w", err)
	}

	if step < 0 {
		step = 0
	}

	total, err := s.TotalSteps(ctx, role)
	if err != nil {
		return RoleState{}, err
	}
	if total > 0 && step > total {
		step = total
	}

	return RoleState{
		Status:      domain.Status(status),
		CurrentStep: step,
		TotalSteps:  total,
	}, nil
}

func (s *Store) SetStatus(ctx context.Context, userID uuid.UUID, role Role, status domain.Status) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO role_progress (user_id, role, status, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, role) DO UPDATE SET status = EXCLUDED.status, updated_at = now()`,
		userID, string(role), string(status))
	if err != nil {
		return fmt.Errorf("обновление статуса: %w", err)
	}
	return nil
}

func (s *Store) ResetProgress(ctx context.Context, userID uuid.UUID, role Role) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("начало транзакции: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stmts := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM training_answers WHERE user_id = $1 AND role = $2`, []any{userID, string(role)}},
		{`UPDATE exam_sessions SET status = 'abandoned', finished_at = now()
		  WHERE user_id = $1 AND role = $2 AND status = 'active'`, []any{userID, string(role)}},
		{`DELETE FROM results WHERE user_id = $1 AND role = $2`, []any{userID, string(role)}},
		{`INSERT INTO role_progress (user_id, role, status, current_step, updated_at)
		  VALUES ($1, $2, 'not_started', 0, now())
		  ON CONFLICT (user_id, role) DO UPDATE
		      SET status = 'not_started', current_step = 0, updated_at = now()`,
			[]any{userID, string(role)}},
	}
	for _, st := range stmts {
		if _, err := tx.Exec(ctx, st.sql, st.args...); err != nil {
			return fmt.Errorf("сброс прогресса: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("коммит: %w", err)
	}
	return nil
}

type Option struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"-"`
}

type Step struct {
	ID          int64
	StepNo      int
	ProductName string
	Message     string
	Explanation string
	Options     []Option
}

func (s *Step) CorrectID() int64 {
	for _, o := range s.Options {
		if o.IsCorrect {
			return o.ID
		}
	}
	return 0
}

func (s *Store) StepByNumber(ctx context.Context, role Role, stepNo int) (Step, error) {
	var st Step
	err := s.pool.QueryRow(ctx, `
		SELECT id, step_no, product_name, message, explanation
		FROM training_steps WHERE role = $1 AND step_no = $2`,
		string(role), stepNo,
	).Scan(&st.ID, &st.StepNo, &st.ProductName, &st.Message, &st.Explanation)
	if errors.Is(err, pgx.ErrNoRows) {
		return Step{}, ErrNotFound
	}
	if err != nil {
		return Step{}, fmt.Errorf("выборка шага: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, text, is_correct FROM training_options
		WHERE step_id = $1 ORDER BY position, id`, st.ID)
	if err != nil {
		return Step{}, fmt.Errorf("выборка вариантов: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var o Option
		if err := rows.Scan(&o.ID, &o.Text, &o.IsCorrect); err != nil {
			return Step{}, fmt.Errorf("чтение варианта: %w", err)
		}
		st.Options = append(st.Options, o)
	}
	if err := rows.Err(); err != nil {
		return Step{}, fmt.Errorf("обход вариантов: %w", err)
	}
	return st, nil
}

func (s *Store) AnsweredCount(ctx context.Context, userID uuid.UUID, role Role) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM training_answers WHERE user_id = $1 AND role = $2`,
		userID, string(role)).Scan(&n); err != nil {
		return 0, fmt.Errorf("подсчёт ответов: %w", err)
	}
	return n, nil
}

type TrainingAnswer struct {
	StepNo    int  `json:"stepNumber"`
	IsCorrect bool `json:"isCorrect"`
}

func (s *Store) Answers(ctx context.Context, userID uuid.UUID, role Role) ([]TrainingAnswer, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ts.step_no, ta.is_correct
		FROM training_answers ta
		JOIN training_steps ts ON ts.id = ta.step_id
		WHERE ta.user_id = $1 AND ta.role = $2
		ORDER BY ts.step_no`, userID, string(role))
	if err != nil {
		return nil, fmt.Errorf("выборка ответов: %w", err)
	}
	defer rows.Close()

	out := []TrainingAnswer{}
	for rows.Next() {
		var a TrainingAnswer
		if err := rows.Scan(&a.StepNo, &a.IsCorrect); err != nil {
			return nil, fmt.Errorf("чтение ответа: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("обход ответов: %w", err)
	}
	return out, nil
}

func (s *Store) RecordAnswer(
	ctx context.Context, userID uuid.UUID, role Role, expectedStep int, optionID int64,
) (step Step, isCorrect bool, answered int, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Step{}, false, 0, fmt.Errorf("начало транзакции: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current int
	err = tx.QueryRow(ctx, `
		SELECT current_step FROM role_progress
		WHERE user_id = $1 AND role = $2 FOR UPDATE`, userID, string(role)).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err = tx.Exec(ctx, `
			INSERT INTO role_progress (user_id, role) VALUES ($1, $2)`,
			userID, string(role)); err != nil {
			return Step{}, false, 0, fmt.Errorf("создание прогресса: %w", err)
		}
		current = 0
	} else if err != nil {
		return Step{}, false, 0, fmt.Errorf("выборка указателя: %w", err)
	}

	stepNo := current + 1
	if expectedStep > 0 && expectedStep != stepNo {
		return Step{}, false, current, ErrStepMismatch
	}

	st, err := s.stepTx(ctx, tx, role, stepNo)
	if err != nil {
		return Step{}, false, current, err
	}

	var correct bool
	found := false
	for _, o := range st.Options {
		if o.ID == optionID {
			correct, found = o.IsCorrect, true
			break
		}
	}
	if !found {
		return Step{}, false, current, ErrNotFound
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO training_answers (user_id, role, step_id, option_id, is_correct)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, role, step_id) DO NOTHING`,
		userID, string(role), st.ID, optionID, correct); err != nil {
		return Step{}, false, current, fmt.Errorf("сохранение ответа: %w", err)
	}

	var total int
	if err = tx.QueryRow(ctx,
		`SELECT count(*) FROM training_steps WHERE role = $1`, string(role)).Scan(&total); err != nil {
		return Step{}, false, current, fmt.Errorf("подсчёт шагов: %w", err)
	}

	status := domain.NextStatusAfterAnswer(domain.StatusTrainingInProgress, stepNo, total)
	if _, err = tx.Exec(ctx, `
		UPDATE role_progress SET current_step = $3, status = $4, updated_at = now()
		WHERE user_id = $1 AND role = $2`,
		userID, string(role), stepNo, string(status)); err != nil {
		return Step{}, false, current, fmt.Errorf("сдвиг указателя: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return Step{}, false, current, fmt.Errorf("коммит: %w", err)
	}
	return st, correct, stepNo, nil
}

func (s *Store) stepTx(ctx context.Context, tx pgx.Tx, role Role, stepNo int) (Step, error) {
	var st Step
	err := tx.QueryRow(ctx, `
		SELECT id, step_no, product_name, message, explanation
		FROM training_steps WHERE role = $1 AND step_no = $2`,
		string(role), stepNo,
	).Scan(&st.ID, &st.StepNo, &st.ProductName, &st.Message, &st.Explanation)
	if errors.Is(err, pgx.ErrNoRows) {
		return Step{}, ErrNotFound
	}
	if err != nil {
		return Step{}, fmt.Errorf("выборка шага: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT id, text, is_correct FROM training_options
		WHERE step_id = $1 ORDER BY position, id`, st.ID)
	if err != nil {
		return Step{}, fmt.Errorf("выборка вариантов: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var o Option
		if err := rows.Scan(&o.ID, &o.Text, &o.IsCorrect); err != nil {
			return Step{}, fmt.Errorf("чтение варианта: %w", err)
		}
		st.Options = append(st.Options, o)
	}
	return st, rows.Err()
}

type Persona struct {
	Name     string `json:"name"`
	AgeGroup string `json:"ageGroup"`
	Gender   string `json:"gender"`
}

type ExamMessage struct {
	ID        int64     `json:"id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExamSession struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Role        Role
	Status      string
	Persona     Persona
	Cycle       int
	Verdict     *string
	Explanation *string
	Mistakes    []domain.Mistake
	StartedAt   time.Time
}

func (s *Store) ActiveSession(ctx context.Context, userID uuid.UUID, role Role) (ExamSession, error) {
	var (
		sess     ExamSession
		persona  []byte
		mistakes []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, role, status, persona, cycle, verdict, explanation, mistakes, started_at
		FROM exam_sessions
		WHERE user_id = $1 AND role = $2 AND status = 'active'`, userID, string(role),
	).Scan(&sess.ID, &sess.UserID, &sess.Role, &sess.Status, &persona, &sess.Cycle,
		&sess.Verdict, &sess.Explanation, &mistakes, &sess.StartedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExamSession{}, ErrNoActive
	}
	if err != nil {
		return ExamSession{}, fmt.Errorf("выборка сессии: %w", err)
	}
	if err := json.Unmarshal(persona, &sess.Persona); err != nil {
		return ExamSession{}, fmt.Errorf("разбор персоны: %w", err)
	}
	if err := json.Unmarshal(mistakes, &sess.Mistakes); err != nil {
		return ExamSession{}, fmt.Errorf("разбор ошибок: %w", err)
	}
	return sess, nil
}

func (s *Store) CreateSession(
	ctx context.Context, userID uuid.UUID, role Role, persona Persona,
) (ExamSession, error) {
	raw, err := json.Marshal(persona)
	if err != nil {
		return ExamSession{}, fmt.Errorf("сериализация персоны: %w", err)
	}

	var sess ExamSession
	err = s.pool.QueryRow(ctx, `
		INSERT INTO exam_sessions (user_id, role, persona) VALUES ($1, $2, $3)
		RETURNING id, user_id, role, status, cycle, started_at`,
		userID, string(role), raw,
	).Scan(&sess.ID, &sess.UserID, &sess.Role, &sess.Status, &sess.Cycle, &sess.StartedAt)
	if err != nil {
		return ExamSession{}, fmt.Errorf("создание сессии: %w", err)
	}
	sess.Persona = persona
	return sess, nil
}

func (s *Store) AbandonActive(ctx context.Context, userID uuid.UUID, role Role) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE exam_sessions SET status = 'abandoned', finished_at = now()
		WHERE user_id = $1 AND role = $2 AND status = 'active'`, userID, string(role))
	if err != nil {
		return fmt.Errorf("закрытие сессии: %w", err)
	}
	return nil
}

func (s *Store) ExpireStaleSessions(ctx context.Context, ttl time.Duration) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE exam_sessions SET status = 'abandoned', finished_at = now()
		WHERE status = 'active' AND started_at < now() - $1::interval`,
		fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	if err != nil {
		return fmt.Errorf("протухшие сессии: %w", err)
	}
	return nil
}

func (s *Store) AddMessage(ctx context.Context, sessionID uuid.UUID, author, text string) (ExamMessage, error) {
	var m ExamMessage
	err := s.pool.QueryRow(ctx, `
		INSERT INTO exam_messages (session_id, author, text) VALUES ($1, $2, $3)
		RETURNING id, author, text, created_at`, sessionID, author, text,
	).Scan(&m.ID, &m.Author, &m.Text, &m.CreatedAt)
	if err != nil {
		return ExamMessage{}, fmt.Errorf("сохранение сообщения: %w", err)
	}
	return m, nil
}

func (s *Store) Messages(ctx context.Context, sessionID uuid.UUID) ([]ExamMessage, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, author, text, created_at FROM exam_messages
		WHERE session_id = $1 ORDER BY id`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("выборка сообщений: %w", err)
	}
	defer rows.Close()

	out := []ExamMessage{}
	for rows.Next() {
		var m ExamMessage
		if err := rows.Scan(&m.ID, &m.Author, &m.Text, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("чтение сообщения: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("обход сообщений: %w", err)
	}
	return out, nil
}

func (s *Store) BumpCycle(ctx context.Context, sessionID uuid.UUID, mistakes []domain.Mistake) (int, error) {
	raw, err := json.Marshal(mistakes)
	if err != nil {
		return 0, fmt.Errorf("сериализация ошибок: %w", err)
	}
	var cycle int
	if err := s.pool.QueryRow(ctx, `
		UPDATE exam_sessions SET cycle = cycle + 1, mistakes = $2
		WHERE id = $1 RETURNING cycle`, sessionID, raw).Scan(&cycle); err != nil {
		return 0, fmt.Errorf("счётчик цикла: %w", err)
	}
	return cycle, nil
}

func (s *Store) FinishSession(ctx context.Context, sessionID uuid.UUID, o domain.ExamOutcome) error {
	raw, err := json.Marshal(o.Mistakes)
	if err != nil {
		return fmt.Errorf("сериализация ошибок: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE exam_sessions
		SET status = 'finished', verdict = $2, explanation = $3, mistakes = $4, finished_at = now()
		WHERE id = $1`, sessionID, string(o.Verdict), o.Explanation, raw)
	if err != nil {
		return fmt.Errorf("завершение сессии: %w", err)
	}
	return nil
}

func (s *Store) SaveResult(ctx context.Context, userID uuid.UUID, role Role, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("сериализация результата: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO results (user_id, role, payload) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role) DO UPDATE SET payload = EXCLUDED.payload, created_at = now()`,
		userID, string(role), raw)
	if err != nil {
		return fmt.Errorf("сохранение результата: %w", err)
	}
	return nil
}

func (s *Store) Result(ctx context.Context, userID uuid.UUID, role Role) (json.RawMessage, error) {
	var raw json.RawMessage
	err := s.pool.QueryRow(ctx,
		`SELECT payload FROM results WHERE user_id = $1 AND role = $2`,
		userID, string(role)).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("выборка результата: %w", err)
	}
	return raw, nil
}
