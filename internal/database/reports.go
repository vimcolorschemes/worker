package database

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// CreateReport stores a job report in the database.
func CreateReport(job string, elapsedTime float64, data map[string]interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal report data: %w", err)
	}

	_, err = db.Exec("INSERT INTO reports (date, job, elapsed_time, data) VALUES (?, ?, ?, ?)",
		time.Now(), job, elapsedTime, string(dataJSON))
	if err == nil {
		return nil
	}

	if !isTransientLibSQLError(err) {
		return fmt.Errorf("insert report: %w", err)
	}

	log.Printf("Report insert failed with transient libsql error, retrying once: %s", err)
	if pingErr := db.Ping(); pingErr != nil {
		return fmt.Errorf("ping before report retry: %w", pingErr)
	}

	_, retryErr := db.Exec("INSERT INTO reports (date, job, elapsed_time, data) VALUES (?, ?, ?, ?)",
		time.Now(), job, elapsedTime, string(dataJSON))
	if retryErr != nil {
		return fmt.Errorf("insert report retry: %w", retryErr)
	}

	return nil
}

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
