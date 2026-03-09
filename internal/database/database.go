package database

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vimcolorschemes/worker/internal/dotenv"

	_ "github.com/tursodatabase/go-libsql"
)

var db *sql.DB

func init() {
	if strings.HasSuffix(os.Args[0], ".test") {
		return
	}

	databaseURL, exists := dotenv.Get("DATABASE_URL")
	if !exists {
		panic("DATABASE_URL not found in env")
	}

	authToken, _ := dotenv.Get("DATABASE_AUTH_TOKEN")
	databaseURL, err := buildDatabaseURL(databaseURL, authToken)
	if err != nil {
		panic(err)
	}

	if strings.HasPrefix(databaseURL, "file:") {
		filePath := strings.TrimPrefix(databaseURL, "file:")
		if idx := strings.IndexByte(filePath, '?'); idx >= 0 {
			filePath = filePath[:idx]
		}
		if dir := filepath.Dir(filePath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				panic(err)
			}
		}
	}

	db, err = sql.Open("libsql", databaseURL)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		panic(err)
	}

	err = initializeSchema(db)
	if err != nil {
		panic(err)
	}
}

func buildDatabaseURL(databaseURL string, authToken string) (string, error) {
	if authToken == "" {
		return databaseURL, nil
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}

	query := parsedURL.Query()
	query.Set("authToken", authToken)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}
