package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
)

type Job string

const (
	JobImport   Job = "import"
	JobUpdate   Job = "update"
	JobGenerate Job = "generate"
)

type JobReport struct {
	Job                  Job            `db:"job"`
	ReportData           database.JSONB `db:"report_data"`
	ElapsedTimeInSeconds int64          `db:"elapsed_time_in_seconds"`
	CreatedAt            time.Time      `db:"created_at"`
}

type JobReportStore struct {
	database *sql.DB
}

func NewJobReportStore(database *sql.DB) *JobReportStore {
	return &JobReportStore{database: database}
}

func (store *JobReportStore) Create(ctx context.Context, jobReport JobReport) error {
	reportData, err := json.Marshal(jobReport.ReportData)
	if err != nil {
		return err
	}

	_, err = store.database.ExecContext(ctx, `
		INSERT INTO job_reports (job, report_data, elapsed_time_in_seconds, created_at)
		VALUES (?, ?, ?, ?)
	`,
		jobReport.Job,
		reportData,
		jobReport.ElapsedTimeInSeconds,
		jobReport.CreatedAt,
	)
	return err
}
