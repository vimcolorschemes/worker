package database

import (
	"database/sql"
	"fmt"
)

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS repositories (
		id                       INTEGER PRIMARY KEY,
		owner_name               TEXT NOT NULL DEFAULT '',
		owner_avatar_url         TEXT NOT NULL DEFAULT '',
		name                     TEXT NOT NULL DEFAULT '',
		github_url               TEXT NOT NULL DEFAULT '',
		stargazers_count         INTEGER NOT NULL DEFAULT 0,
		stargazers_count_history TEXT NOT NULL DEFAULT '[]',
		week_stargazers_count    INTEGER NOT NULL DEFAULT 0,
		github_created_at        DATETIME,
		pushed_at                DATETIME,
		is_eligible              BOOLEAN NOT NULL DEFAULT 0,
		updated_at               DATETIME,
		generated_at             DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS color_schemes (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
		name          TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS color_scheme_groups (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		color_scheme_id INTEGER NOT NULL REFERENCES color_schemes(id) ON DELETE CASCADE,
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
}

func initializeSchema(db *sql.DB) error {
	if err := migrateColorSchemeTables(db); err != nil {
		return err
	}

	for _, statement := range schemaStatements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	if err := migrateRepositoriesSchema(db); err != nil {
		return err
	}

	return nil
}

func migrateColorSchemeTables(db *sql.DB) error {
	vimColorSchemesTableExists, err := tableExists(db, "vim_color_schemes")
	if err != nil {
		return err
	}
	colorSchemesTableExists, err := tableExists(db, "color_schemes")
	if err != nil {
		return err
	}

	if vimColorSchemesTableExists && !colorSchemesTableExists {
		if _, err := db.Exec("ALTER TABLE vim_color_schemes RENAME TO color_schemes"); err != nil {
			return fmt.Errorf("rename vim_color_schemes to color_schemes: %w", err)
		}
	}

	vimColorSchemeGroupsTableExists, err := tableExists(db, "vim_color_scheme_groups")
	if err != nil {
		return err
	}
	colorSchemeGroupsTableExists, err := tableExists(db, "color_scheme_groups")
	if err != nil {
		return err
	}

	if vimColorSchemeGroupsTableExists && !colorSchemeGroupsTableExists {
		if _, err := db.Exec("ALTER TABLE vim_color_scheme_groups RENAME TO color_scheme_groups"); err != nil {
			return fmt.Errorf("rename vim_color_scheme_groups to color_scheme_groups: %w", err)
		}
	}

	return nil
}

func tableExists(db *sql.DB, tableName string) (bool, error) {
	var actual string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", tableName).Scan(&actual)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func migrateRepositoriesSchema(db *sql.DB) error {
	columns, err := getTableColumns(db, "repositories")
	if err != nil {
		return err
	}

	if columns["update_valid"] && !columns["is_eligible"] {
		if _, err := db.Exec("ALTER TABLE repositories RENAME COLUMN update_valid TO is_eligible"); err != nil {
			return fmt.Errorf("rename update_valid to is_eligible: %w", err)
		}
		columns["is_eligible"] = true
		delete(columns, "update_valid")
	}

	if columns["generate_valid"] {
		if _, err := db.Exec("ALTER TABLE repositories DROP COLUMN generate_valid"); err != nil {
			return fmt.Errorf("drop generate_valid: %w", err)
		}
	}

	return nil
}

func getTableColumns(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}
