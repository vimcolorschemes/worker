-- +goose Up
ALTER TABLE repositories ADD COLUMN is_disabled BOOLEAN NOT NULL DEFAULT 0;

DROP INDEX IF EXISTS idx_repositories_generate_due;
CREATE INDEX idx_repositories_generate_due
    ON repositories(is_disabled, is_eligible, pushed_at, last_generate_event_at);

-- +goose Down
DROP INDEX IF EXISTS idx_repositories_generate_due;

ALTER TABLE repositories DROP COLUMN is_disabled;

CREATE INDEX idx_repositories_generate_due
    ON repositories(is_eligible, pushed_at, last_generate_event_at);
