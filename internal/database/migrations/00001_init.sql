-- +goose Up
CREATE TABLE repositories (
    id                       INTEGER PRIMARY KEY,
    owner_name               TEXT NOT NULL DEFAULT '',
    owner_avatar_url         TEXT NOT NULL DEFAULT '',
    name                     TEXT NOT NULL DEFAULT '',
    description              TEXT NOT NULL DEFAULT '',
    github_url               TEXT NOT NULL DEFAULT '',
    stargazers_count         INTEGER NOT NULL DEFAULT 0,
    stargazers_count_history TEXT NOT NULL DEFAULT '[]',
    week_stargazers_count    INTEGER NOT NULL DEFAULT 0,
    github_created_at        DATETIME,
    pushed_at                DATETIME,
    last_generate_event_at   DATETIME,
    is_eligible              BOOLEAN NOT NULL DEFAULT 0,
    updated_at               DATETIME,
    featured_rank            INTEGER,
    has_dark                 INTEGER NOT NULL DEFAULT 0,
    has_light                INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE repository_job_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    job           TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'success',
    error_message TEXT,
    created_at    DATETIME NOT NULL
);

CREATE TABLE colorschemes (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    name          TEXT NOT NULL
);

CREATE TABLE colorscheme_groups (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    colorscheme_id INTEGER NOT NULL REFERENCES colorschemes(id) ON DELETE CASCADE,
    background     TEXT NOT NULL,
    name           TEXT NOT NULL,
    hex_code       TEXT NOT NULL
);

CREATE TABLE reports (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    date         DATETIME NOT NULL,
    job          TEXT NOT NULL,
    elapsed_time REAL NOT NULL,
    data         TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_repositories_owner_name_name_nocase
    ON repositories(owner_name COLLATE NOCASE, name COLLATE NOCASE);

CREATE INDEX idx_repositories_week_stargazers_count_id
    ON repositories(week_stargazers_count DESC, id);

CREATE INDEX idx_repositories_stargazers_count_id
    ON repositories(stargazers_count DESC, id);

CREATE INDEX idx_repositories_github_created_at_id
    ON repositories(github_created_at, id);

CREATE INDEX idx_repositories_owner_week_stars_id_nocase
    ON repositories(owner_name COLLATE NOCASE, week_stargazers_count DESC, id);

CREATE INDEX idx_repositories_generate_due
    ON repositories(is_eligible, pushed_at, last_generate_event_at);

CREATE UNIQUE INDEX idx_repositories_featured_rank
    ON repositories(featured_rank) WHERE featured_rank IS NOT NULL;

CREATE INDEX idx_repositories_has_dark_has_light
    ON repositories(has_dark, has_light);

CREATE INDEX idx_colorschemes_repository_id_id
    ON colorschemes(repository_id, id);

CREATE INDEX idx_colorscheme_groups_scheme_id_id
    ON colorscheme_groups(colorscheme_id, id);

CREATE INDEX idx_colorscheme_groups_background_scheme_id
    ON colorscheme_groups(background, colorscheme_id);

CREATE INDEX idx_repository_job_events_job_repository_created
    ON repository_job_events(job, repository_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_repository_job_events_job_repository_created;
DROP INDEX IF EXISTS idx_colorscheme_groups_background_scheme_id;
DROP INDEX IF EXISTS idx_colorscheme_groups_scheme_id_id;
DROP INDEX IF EXISTS idx_colorschemes_repository_id_id;
DROP INDEX IF EXISTS idx_repositories_has_dark_has_light;
DROP INDEX IF EXISTS idx_repositories_featured_rank;
DROP INDEX IF EXISTS idx_repositories_generate_due;
DROP INDEX IF EXISTS idx_repositories_owner_week_stars_id_nocase;
DROP INDEX IF EXISTS idx_repositories_github_created_at_id;
DROP INDEX IF EXISTS idx_repositories_stargazers_count_id;
DROP INDEX IF EXISTS idx_repositories_week_stargazers_count_id;
DROP INDEX IF EXISTS idx_repositories_owner_name_name_nocase;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS colorscheme_groups;
DROP TABLE IF EXISTS colorschemes;
DROP TABLE IF EXISTS repository_job_events;
DROP TABLE IF EXISTS repositories;
