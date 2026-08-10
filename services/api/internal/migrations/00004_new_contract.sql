-- +goose Up

ALTER TABLE users ADD COLUMN IF NOT EXISTS anon_token text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS name       text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS age_group  text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS gender     text NOT NULL DEFAULT 'male';

UPDATE users SET anon_token = external_id WHERE anon_token IS NULL;

ALTER TABLE users ALTER COLUMN anon_token SET NOT NULL;
ALTER TABLE users ALTER COLUMN external_id DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_anon_token_key ON users (anon_token);

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_gender_check'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_gender_check
            CHECK (gender IN ('male', 'female'));
    END IF;
END $$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS role_progress (
    user_id      uuid     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role         text     NOT NULL CHECK (role IN ('buyer', 'seller')),
    status       text     NOT NULL DEFAULT 'not_started'
                 CHECK (status IN ('not_started', 'training_in_progress', 'training_passed',
                                   'exam_in_progress', 'exam_passed', 'exam_failed')),
    current_step smallint NOT NULL DEFAULT 0,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role)
);

CREATE TABLE IF NOT EXISTS training_steps (
    id           bigserial PRIMARY KEY,
    role         text     NOT NULL CHECK (role IN ('buyer', 'seller')),
    step_no      smallint NOT NULL,
    product_name text     NOT NULL,
    message      text     NOT NULL,
    explanation  text     NOT NULL,
    UNIQUE (role, step_no)
);

CREATE TABLE IF NOT EXISTS training_options (
    id         bigserial PRIMARY KEY,
    step_id    bigint  NOT NULL REFERENCES training_steps (id) ON DELETE CASCADE,
    position   int     NOT NULL DEFAULT 0,
    text       text    NOT NULL,
    is_correct boolean NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_training_options_step ON training_options (step_id, position);

CREATE TABLE IF NOT EXISTS training_answers (
    id         bigserial PRIMARY KEY,
    user_id    uuid    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       text    NOT NULL,
    step_id    bigint  NOT NULL REFERENCES training_steps (id),
    option_id  bigint  NOT NULL REFERENCES training_options (id),
    is_correct boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, role, step_id)
);

CREATE TABLE IF NOT EXISTS exam_sessions (
    id          uuid     PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role        text     NOT NULL CHECK (role IN ('buyer', 'seller')),
    status      text     NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'finished', 'abandoned')),
    persona     jsonb    NOT NULL DEFAULT '{}'::jsonb,
    cycle       smallint NOT NULL DEFAULT 0,
    verdict     text     CHECK (verdict IN ('passed', 'failed')),
    explanation text,
    mistakes    jsonb    NOT NULL DEFAULT '[]'::jsonb,
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_session
    ON exam_sessions (user_id, role) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS exam_messages (
    id         bigserial PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES exam_sessions (id) ON DELETE CASCADE,
    author     text NOT NULL CHECK (author IN ('scammer', 'user')),
    text       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_exam_messages_session ON exam_messages (session_id, id);

CREATE TABLE IF NOT EXISTS results (
    user_id    uuid  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       text  NOT NULL CHECK (role IN ('buyer', 'seller')),
    payload    jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role)
);

INSERT INTO training_steps (role, step_no, product_name, message, explanation)
SELECT
    s.role,
    s.order_index,
    CASE
        WHEN s.role = 'seller' THEN 'iPhone 13, 128 ГБ'
        WHEN s.order_index = 1 THEN 'ноутбук ASUS VivoBook 15'
        ELSE 'iPhone 12, 64 ГБ'
    END,
    s.situation || E'\n\n' || s.question,
    s.explanation
FROM scenarios s
ON CONFLICT (role, step_no) DO NOTHING;

INSERT INTO training_options (step_id, position, text, is_correct)
SELECT
    ts.id,
    (opt.ord - 1)::int,
    opt.value ->> 'text',
    (opt.ord - 1) = s.correct_option
FROM scenarios s
JOIN training_steps ts ON ts.role = s.role AND ts.step_no = s.order_index
CROSS JOIN LATERAL jsonb_array_elements(s.options) WITH ORDINALITY AS opt(value, ord)
WHERE NOT EXISTS (SELECT 1 FROM training_options o WHERE o.step_id = ts.id);

-- +goose StatementBegin
DO $$
DECLARE
    bad record;
BEGIN
    FOR bad IN
        SELECT ts.role, ts.step_no, count(*) FILTER (WHERE o.is_correct) AS correct_count
        FROM training_steps ts
        LEFT JOIN training_options o ON o.step_id = ts.id
        GROUP BY ts.role, ts.step_no
        HAVING count(*) FILTER (WHERE o.is_correct) <> 1
    LOOP
        RAISE EXCEPTION
            'training_steps: у шага role=% step_no=% правильных вариантов %, должен быть ровно 1',
            bad.role, bad.step_no, bad.correct_count;
    END LOOP;

    FOR bad IN
        SELECT ts.role, ts.step_no, count(o.id) AS total
        FROM training_steps ts
        LEFT JOIN training_options o ON o.step_id = ts.id
        GROUP BY ts.role, ts.step_no
        HAVING count(o.id) < 2
    LOOP
        RAISE EXCEPTION
            'training_steps: у шага role=% step_no=% всего вариантов %, нужно минимум 2',
            bad.role, bad.step_no, bad.total;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS results;
DROP TABLE IF EXISTS exam_messages;
DROP TABLE IF EXISTS exam_sessions;
DROP TABLE IF EXISTS training_answers;
DROP TABLE IF EXISTS training_options;
DROP TABLE IF EXISTS training_steps;
DROP TABLE IF EXISTS role_progress;

DROP INDEX IF EXISTS users_anon_token_key;
ALTER TABLE users DROP COLUMN IF EXISTS gender;
ALTER TABLE users DROP COLUMN IF EXISTS age_group;
ALTER TABLE users DROP COLUMN IF EXISTS name;
ALTER TABLE users DROP COLUMN IF EXISTS anon_token;
