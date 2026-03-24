package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vimcolorschemes/worker/internal/date"
)

// CreateReport stores a job report in the database.
func CreateReport(job string, elapsedTime float64, data map[string]interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal report data: %w", err)
	}

	_, err = execWithTransientRetry("INSERT INTO reports (date, job, elapsed_time, data) VALUES (?, ?, ?, ?)",
		time.Now().UTC(), job, elapsedTime, string(dataJSON))
	if err != nil {
		return fmt.Errorf("insert report: %w", err)
	}

	return nil
}

// GetLatestReportStatuses returns each job's latest status for the provided UTC day.
func GetLatestReportStatuses(jobs []string, day time.Time) (map[string]string, error) {
	statuses := make(map[string]string, len(jobs))
	day = date.RoundTimeToDate(day.UTC())
	nextDay := day.Add(24 * time.Hour)

	for _, job := range jobs {
		var data string
		err := db.QueryRow(
			"SELECT data FROM reports WHERE job = ? AND date >= ? AND date < ? ORDER BY date DESC, id DESC LIMIT 1",
			job,
			day,
			nextDay,
		).Scan(&data)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				statuses[job] = "missing"
				continue
			}

			return nil, fmt.Errorf("query latest %s report: %w", job, err)
		}

		status, err := reportStatusFromData(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s report: %w", job, err)
		}

		statuses[job] = status
	}

	return statuses, nil
}

func reportStatusFromData(data string) (string, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return "", err
	}

	status, ok := payload["status"].(string)
	if !ok || status == "" {
		return "success", nil
	}

	return status, nil
}
