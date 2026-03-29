package database

import (
	"database/sql"
	"embed"
	"sync"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var configureMigrationsOnce sync.Once
var configureMigrationsErr error

func applyMigrations(db *sql.DB) error {
	configureMigrationsOnce.Do(func() {
		goose.SetBaseFS(migrationsFS)
		configureMigrationsErr = goose.SetDialect("turso")
	})
	if configureMigrationsErr != nil {
		return configureMigrationsErr
	}

	return goose.Up(db, "migrations")
}
