package database

import (
	"database/sql"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type JSONB map[string]any

func Connect() *sql.DB {
	url := os.Getenv("TURSO_DATABASE_URL")
	token := os.Getenv("TURSO_AUTH_TOKEN")

	var driver, dsn string

	if strings.HasPrefix(url, "libsql://") {
		driver = "libsql"
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		dsn = url
		if token != "" {
			dsn = dsn + sep + "authToken=" + token
		}
	} else {
		driver, dsn = "sqlite3", "./db/vimcolorschemes.db"
	}

	log.Printf("Connecting to database with driver %s and dsn %s", driver, dsn)

	db, err := sql.Open(driver, dsn)
	if err != nil {
		log.Fatal(err)
	}

	return db
}
