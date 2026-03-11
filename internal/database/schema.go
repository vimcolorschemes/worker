package database

import (
	"database/sql"
)

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS repositories (
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
		is_eligible              BOOLEAN NOT NULL DEFAULT 0,
		updated_at               DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS repository_job_events (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
		job           TEXT NOT NULL,
		status        TEXT NOT NULL DEFAULT 'success',
		error_message TEXT,
		created_at    DATETIME NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS colorschemes (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
		name          TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS colorscheme_groups (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		colorscheme_id  INTEGER NOT NULL REFERENCES colorschemes(id) ON DELETE CASCADE,
		background      TEXT NOT NULL,
		name            TEXT NOT NULL,
		hex_code        TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS reports (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		date         DATETIME NOT NULL,
		job          TEXT NOT NULL,
		elapsed_time REAL NOT NULL,
		data         TEXT NOT NULL DEFAULT '{}'
	)`,

	// Case-insensitive lookup for repository detail pages.
	`CREATE INDEX IF NOT EXISTS idx_repositories_owner_name_name_nocase
		ON repositories(owner_name COLLATE NOCASE, name COLLATE NOCASE)`,

	// Trending sort for repository lists.
	`CREATE INDEX IF NOT EXISTS idx_repositories_week_stargazers_count_id
		ON repositories(week_stargazers_count DESC, id)`,

	// Top sort for repository lists.
	`CREATE INDEX IF NOT EXISTS idx_repositories_stargazers_count_id
		ON repositories(stargazers_count DESC, id)`,

	// New/old sort for repository lists.
	`CREATE INDEX IF NOT EXISTS idx_repositories_github_created_at_id
		ON repositories(github_created_at, id)`,

	// Owner page listing sorted by trending.
	`CREATE INDEX IF NOT EXISTS idx_repositories_owner_week_stars_id_nocase
		ON repositories(owner_name COLLATE NOCASE, week_stargazers_count DESC, id)`,

	// Fast existence checks and joins for repository colorschemes.
	`CREATE INDEX IF NOT EXISTS idx_colorschemes_repository_id_id
		ON colorschemes(repository_id, id)`,

	// Fast loading of groups for each selected colorscheme.
	`CREATE INDEX IF NOT EXISTS idx_colorscheme_groups_scheme_id_id
		ON colorscheme_groups(colorscheme_id, id)`,

	// Background filter support (dark/light/both).
	`CREATE INDEX IF NOT EXISTS idx_colorscheme_groups_background_scheme_id
		ON colorscheme_groups(background, colorscheme_id)`,

	// Generate queue lookup by latest generate event per repository.
	`CREATE INDEX IF NOT EXISTS idx_repository_job_events_job_repository_created
		ON repository_job_events(job, repository_id, created_at DESC)`,
}

func initializeSchema(db *sql.DB) error {
	for _, statement := range schemaStatements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}
