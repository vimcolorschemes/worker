package database

import (
	"encoding/json"
	"fmt"
	"time"
)

// CreateReport stores a job report in the database.
func CreateReport(job string, elapsedTime float64, data map[string]interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal report data: %w", err)
	}

	_, err = execWithTransientRetry("INSERT INTO reports (date, job, elapsed_time, data) VALUES (?, ?, ?, ?)",
		time.Now(), job, elapsedTime, string(dataJSON))
	if err != nil {
		return fmt.Errorf("insert report: %w", err)
	}

	return nil
}
