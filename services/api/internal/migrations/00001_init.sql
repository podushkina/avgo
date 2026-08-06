-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id          uuid PRIMARY KEY,
    external_id text        NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scenarios (
    id             serial PRIMARY KEY,
    role           text   NOT NULL CHECK (role IN ('buyer', 'seller')),
    order_index    int    NOT NULL,
    title          text   NOT NULL,
    situation      text   NOT NULL,
    question       text   NOT NULL,
    options        jsonb  NOT NULL,
    correct_option int    NOT NULL CHECK (correct_option >= 0),
    explanation    text   NOT NULL,
    red_flags      jsonb  NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (role, order_index)
);

CREATE TABLE IF NOT EXISTS progress (
    id            uuid PRIMARY KEY,
    user_id       uuid         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role          text         NOT NULL CHECK (role IN ('buyer', 'seller')),
    correct_count int          NOT NULL,
    total_count   int          NOT NULL,
    percent       numeric(5,1) NOT NULL,
    score         int          NOT NULL DEFAULT 0,
    answers       jsonb        NOT NULL DEFAULT '[]'::jsonb,
    completed_at  timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS progress_user_completed_idx
    ON progress (user_id, completed_at DESC);

-- +goose Down
DROP TABLE IF EXISTS progress;
DROP TABLE IF EXISTS scenarios;
DROP TABLE IF EXISTS users;
