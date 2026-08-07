-- +goose Up
CREATE TABLE IF NOT EXISTS attempts (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       text NOT NULL CHECK (role IN ('buyer', 'seller')),
    status     text NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed')),
    answers    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS attempts_user_idx ON attempts (user_id);

-- +goose Down
DROP TABLE IF EXISTS attempts;