package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/vimcolorschemes/worker/internal/date"
)

type JobReport struct {
	Job         string
	Date        time.Time
	ElapsedTime float64
	Status      string
	Data        map[string]interface{}
}

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
	reports, err := GetLatestReports(jobs, day)
	if err != nil {
		return nil, err
	}

	statuses := make(map[string]string, len(jobs))
	for _, job := range jobs {
		report, ok := reports[job]
		if !ok {
			statuses[job] = "missing"
			continue
		}

		statuses[job] = report.Status
	}

	return statuses, nil
}

// GetLatestReports returns each job's latest full report for the provided UTC day.
func GetLatestReports(jobs []string, day time.Time) (map[string]JobReport, error) {
	day = date.RoundTimeToDate(day.UTC())
	nextDay := day.Add(24 * time.Hour)
	reports := make(map[string]JobReport, len(jobs))

	for _, job := range jobs {
		var (
			dateValue   time.Time
			elapsedTime float64
			data        string
		)
		err := db.QueryRow(
			"SELECT date, elapsed_time, data FROM reports WHERE job = ? AND date >= ? AND date < ? ORDER BY date DESC, id DESC LIMIT 1",
			job,
			day,
			nextDay,
		).Scan(&dateValue, &elapsedTime, &data)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}

			return nil, fmt.Errorf("query latest %s report: %w", job, err)
		}

		payload, err := parseReportData(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s report: %w", job, err)
		}

		reports[job] = JobReport{
			Job:         job,
			Date:        dateValue,
			ElapsedTime: elapsedTime,
			Status:      reportStatusFromPayload(payload),
			Data:        payload,
		}
	}

	return reports, nil
}

func reportStatusFromData(data string) (string, error) {
	payload, err := parseReportData(data)
	if err != nil {
		return "", err
	}

	return reportStatusFromPayload(payload), nil
}

// CountRepositoryJobEvents returns counts per status for a repository job on the provided UTC day.
func CountRepositoryJobEvents(job string, day time.Time) (map[string]int, error) {
	day = date.RoundTimeToDate(day.UTC())
	nextDay := day.Add(24 * time.Hour)

	rows, err := db.Query(
		"SELECT status, COUNT(*) FROM repository_job_events WHERE job = ? AND created_at >= ? AND created_at < ? GROUP BY status",
		job,
		day,
		nextDay,
	)
	if err != nil {
		return nil, fmt.Errorf("query %s repository job event counts: %w", job, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	counts := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan %s repository job event count: %w", job, err)
		}
		counts[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s repository job event counts: %w", job, err)
	}

	return counts, nil
}

// ListRepositoryJobEventMessages returns up to limit latest error messages for a repository job on the provided UTC day.
func ListRepositoryJobEventMessages(job string, status string, day time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		return []string{}, nil
	}

	day = date.RoundTimeToDate(day.UTC())
	nextDay := day.Add(24 * time.Hour)

	rows, err := db.Query(
		"SELECT error_message FROM repository_job_events WHERE job = ? AND status = ? AND created_at >= ? AND created_at < ? ORDER BY created_at DESC, id DESC LIMIT ?",
		job,
		status,
		day,
		nextDay,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query %s repository job event messages: %w", job, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	messages := make([]string, 0, limit)
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			return nil, fmt.Errorf("scan %s repository job event message: %w", job, err)
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s repository job event messages: %w", job, err)
	}

	sort.Strings(messages)
	return messages, nil
}

func parseReportData(data string) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func reportStatusFromPayload(payload map[string]interface{}) string {
	status, ok := payload["status"].(string)
	if !ok || status == "" {
		return "success"
	}

	return status
}
