-- +goose Up
CREATE TABLE projects (
    source_key    text PRIMARY KEY,
    title         text NOT NULL,
    description   text NOT NULL,
    detail_url    text,
    status        text NOT NULL CHECK (status IN ('active', 'completed')),
    first_seen_at timestamptz NOT NULL,
    last_seen_at  timestamptz NOT NULL
);

-- +goose Down
DROP TABLE projects;
