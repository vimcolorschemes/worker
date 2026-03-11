package database

import (
	"database/sql"
	"path/filepath"
	"testing"
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
	if err := initializeSchema(db); err != nil {
		t.Fatalf("initializeSchema returned error: %v", err)
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

func TestInitializeSchemaCreatesAllTables(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := initializeSchema(db); err != nil {
		t.Fatalf("initializeSchema returned error: %v", err)
	}

	for _, tableName := range []string{"repositories", "repository_job_events", "colorschemes", "colorscheme_groups", "reports"} {
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

func TestInitializeSchemaCreatesExpectedIndexes(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("libsql", "file:"+databasePath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := initializeSchema(db); err != nil {
		t.Fatalf("initializeSchema returned error: %v", err)
	}

	for _, indexName := range []string{
		"idx_repositories_owner_name_name_nocase",
		"idx_repositories_week_stargazers_count_id",
		"idx_repositories_stargazers_count_id",
		"idx_repositories_github_created_at_id",
		"idx_repositories_owner_week_stars_id_nocase",
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
