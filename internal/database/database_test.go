package database

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	databasePath := filepath.Join(t.TempDir(), "test.db")
	var err error
	db, err = sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("PRAGMA foreign_keys returned error: %v", err)
	}
	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		db = nil
	})
}

func TestBuildDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		authToken   string
		want        string
	}{
		{
			name:        "keeps URL without token",
			databaseURL: "file:./vimcolorschemes.db",
			want:        "file:./vimcolorschemes.db",
		},
		{
			name:        "adds token to file URL",
			databaseURL: "file:./vimcolorschemes.db",
			authToken:   "secret token",
			want:        "file:./vimcolorschemes.db?authToken=secret+token",
		},
		{
			name:        "merges existing query params",
			databaseURL: "libsql://example.turso.io?syncUrl=file%3A.%2Freplica.db",
			authToken:   "secret token",
			want:        "libsql://example.turso.io?authToken=secret+token&syncUrl=file%3A.%2Freplica.db",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildDatabaseURL(test.databaseURL, test.authToken)
			if err != nil {
				t.Fatalf("buildDatabaseURL returned error: %v", err)
			}

			if got != test.want {
				t.Fatalf("buildDatabaseURL returned %q, want %q", got, test.want)
			}
		})
	}
}

func TestApplyMigrationsCreatesAllTables(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations returned error: %v", err)
	}

	for _, tableName := range []string{"repositories", "repositories_search", "repository_job_events", "colorschemes", "colorscheme_groups", "reports", "goose_db_version"} {
		var actual string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", tableName).Scan(&actual)
		if err != nil {
			t.Fatalf("table %q was not created: %v", tableName, err)
		}
		if actual != tableName {
			t.Fatalf("created table %q, want %q", actual, tableName)
		}
	}
}

func TestApplyMigrationsAddsRepositoryDisabledColumn(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations returned error: %v", err)
	}

	var name string
	var notNull int
	var defaultValue sql.NullString
	err = db.QueryRow("SELECT name, \"notnull\", dflt_value FROM pragma_table_info('repositories') WHERE name = 'is_disabled'").Scan(&name, &notNull, &defaultValue)
	if err != nil {
		t.Fatalf("query column info: %v", err)
	}
	if name != "is_disabled" {
		t.Fatalf("column name = %q, want %q", name, "is_disabled")
	}
	if notNull != 1 {
		t.Fatalf("notnull = %d, want 1", notNull)
	}
	if !defaultValue.Valid || defaultValue.String != "0" {
		t.Fatalf("defaultValue = %q, want %q", defaultValue.String, "0")
	}

	if _, err := db.Exec(`INSERT INTO repositories (id, owner_name, name) VALUES (1, 'owner', 'repo')`); err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	var isDisabled int
	err = db.QueryRow(`SELECT is_disabled FROM repositories WHERE id = 1`).Scan(&isDisabled)
	if err != nil {
		t.Fatalf("query repo: %v", err)
	}
	if isDisabled != 0 {
		t.Fatalf("is_disabled = %d, want 0", isDisabled)
	}
}

func TestRepositoryDisabledMigrationRoundTrip(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations returned error: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO repositories (id, owner_name, name, is_disabled) VALUES (1, 'owner', 'repo', 1)`); err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	var columnCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('repositories') WHERE name = 'is_disabled'").Scan(&columnCount)
	if err != nil {
		t.Fatalf("query column count after up: %v", err)
	}
	if columnCount != 1 {
		t.Fatalf("columnCount after up = %d, want 1", columnCount)
	}

	if err := goose.DownTo(db, "migrations", 20260329211101); err != nil {
		t.Fatalf("goose.DownTo returned error: %v", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('repositories') WHERE name = 'is_disabled'").Scan(&columnCount)
	if err != nil {
		t.Fatalf("query column count after down: %v", err)
	}
	if columnCount != 0 {
		t.Fatalf("columnCount after down = %d, want 0", columnCount)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("goose.Up returned error: %v", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('repositories') WHERE name = 'is_disabled'").Scan(&columnCount)
	if err != nil {
		t.Fatalf("query column count after re-up: %v", err)
	}
	if columnCount != 1 {
		t.Fatalf("columnCount after re-up = %d, want 1", columnCount)
	}

	var isDisabled int
	err = db.QueryRow(`SELECT is_disabled FROM repositories WHERE id = 1`).Scan(&isDisabled)
	if err != nil {
		t.Fatalf("query repo after re-up: %v", err)
	}
	if isDisabled != 0 {
		t.Fatalf("is_disabled after re-up = %d, want 0", isDisabled)
	}
}

func TestApplyMigrationsCreatesExpectedTriggers(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations returned error: %v", err)
	}

	for _, triggerName := range []string{
		"repositories_search_ai",
		"repositories_search_ad",
		"repositories_search_au",
	} {
		var actual string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'trigger' AND name = ?", triggerName).Scan(&actual)
		if err != nil {
			t.Fatalf("trigger %q was not created: %v", triggerName, err)
		}
		if actual != triggerName {
			t.Fatalf("created trigger %q, want %q", actual, triggerName)
		}
	}
}

func TestApplyMigrationsCreatesExpectedIndexes(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations returned error: %v", err)
	}

	for _, indexName := range []string{
		"idx_repositories_owner_name_name_nocase",
		"idx_repositories_week_stargazers_count_id",
		"idx_repositories_stargazers_count_id",
		"idx_repositories_github_created_at_id",
		"idx_repositories_owner_week_stars_id_nocase",
		"idx_repositories_generate_due",
		"idx_repositories_featured_rank",
		"idx_repositories_has_dark_has_light",
		"idx_colorschemes_repository_id_id",
		"idx_colorscheme_groups_scheme_id_id",
		"idx_colorscheme_groups_background_scheme_id",
		"idx_repository_job_events_job_repository_created",
	} {
		var actual string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?", indexName).Scan(&actual)
		if err != nil {
			t.Fatalf("index %q was not created: %v", indexName, err)
		}
		if actual != indexName {
			t.Fatalf("created index %q, want %q", actual, indexName)
		}
	}
}

func TestApplyMigrationsIsIdempotent(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("first applyMigrations returned error: %v", err)
	}

	if err := applyMigrations(db); err != nil {
		t.Fatalf("second applyMigrations returned error: %v", err)
	}
}
