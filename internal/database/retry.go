package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

func isTransientLibSQLError(err error) bool {
	errMsg := strings.ToLower(err.Error())

	for _, marker := range []string{
		"invalid token",
		"stream not found: generation mismatch",
		"generation mismatch",
		"stream not found",
	} {
		if strings.Contains(errMsg, marker) {
			return true
		}
	}

	return false
}

func execWithTransientRetry(query string, args ...any) (sql.Result, error) {
	result, err := db.Exec(query, args...)
	if err == nil {
		return result, nil
	}

	if !isTransientLibSQLError(err) {
		return nil, err
	}

	log.Printf("DB exec failed with transient libsql error, retrying once: %s", err)
	if pingErr := db.Ping(); pingErr != nil {
		return nil, fmt.Errorf("ping before exec retry: %w", pingErr)
	}

	return db.Exec(query, args...)
}

func queryWithTransientRetry(query string, args ...any) (*sql.Rows, error) {
	rows, err := db.Query(query, args...)
	if err == nil {
		return rows, nil
	}

	if !isTransientLibSQLError(err) {
		return nil, err
	}

	log.Printf("DB query failed with transient libsql error, retrying once: %s", err)
	if pingErr := db.Ping(); pingErr != nil {
		return nil, fmt.Errorf("ping before query retry: %w", pingErr)
	}

	return db.Query(query, args...)
}

func beginWithTransientRetry() (*sql.Tx, error) {
	tx, err := db.Begin()
	if err == nil {
		return tx, nil
	}

	if !isTransientLibSQLError(err) {
		return nil, err
	}

	log.Printf("DB begin failed with transient libsql error, retrying once: %s", err)
	if pingErr := db.Ping(); pingErr != nil {
		return nil, fmt.Errorf("ping before begin retry: %w", pingErr)
	}

	return db.Begin()
}
