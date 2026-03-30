package database

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCreateReport(t *testing.T) {
	t.Run("inserts a report row", func(t *testing.T) {
		setupTestDB(t)
		if err := CreateReport("import", 1.5, map[string]interface{}{}); err != nil {
			t.Fatalf("CreateReport returned error: %v", err)
		}

		var count int
		err := db.QueryRow("SELECT count(*) FROM reports").Scan(&count)
		if err != nil {
			t.Fatalf("query count: %v", err)
		}
		if count != 1 {
			t.Fatalf("count = %d, want 1", count)
		}
	})

	t.Run("marshals data map to JSON", func(t *testing.T) {
		setupTestDB(t)
		if err := CreateReport("import", 1.5, map[string]interface{}{"count": 5}); err != nil {
			t.Fatalf("CreateReport returned error: %v", err)
		}

		var data string
		err := db.QueryRow("SELECT data FROM reports").Scan(&data)
		if err != nil {
			t.Fatalf("query data: %v", err)
		}
		if !strings.Contains(data, "count") {
			t.Fatalf("data = %q, want it to contain %q", data, "count")
		}
	})

	t.Run("returns error when database is unavailable", func(t *testing.T) {
		setupTestDB(t)
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}

		err := CreateReport("import", 1.5, map[string]interface{}{})
		if err == nil {
			t.Fatal("CreateReport error = nil, want error")
		}
	})
}

func TestGetLatestReportStatuses(t *testing.T) {
	t.Run("returns latest status for each job on the requested day", func(t *testing.T) {
		setupTestDB(t)

		day := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC)
		insertReport(t, "import", day.Add(time.Hour), `{"repositoryCount":1}`)
		insertReport(t, "update", day.Add(2*time.Hour), `{"repositoryCount":1}`)
		insertReport(t, "generate", day.Add(3*time.Hour), `{"repositoryCount":1}`)

		statuses, err := GetLatestReportStatuses([]string{"import", "update", "generate"}, day)
		if err != nil {
			t.Fatalf("GetLatestReportStatuses returned error: %v", err)
		}

		for _, job := range []string{"import", "update", "generate"} {
			if got := statuses[job]; got != "success" {
				t.Fatalf("statuses[%q] = %q, want %q", job, got, "success")
			}
		}
	})

	t.Run("uses the latest report when a job reruns", func(t *testing.T) {
		setupTestDB(t)

		day := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC)
		insertReport(t, "generate", day.Add(time.Hour), `{"repositoryCount":1}`)
		insertReport(t, "generate", day.Add(2*time.Hour), `{"status":"error","error":"boom"}`)

		statuses, err := GetLatestReportStatuses([]string{"generate"}, day)
		if err != nil {
			t.Fatalf("GetLatestReportStatuses returned error: %v", err)
		}

		if got := statuses["generate"]; got != "error" {
			t.Fatalf("statuses[%q] = %q, want %q", "generate", got, "error")
		}
	})

	t.Run("marks jobs missing when they have no report that day", func(t *testing.T) {
		setupTestDB(t)

		day := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC)
		insertReport(t, "import", day.Add(-time.Hour), `{"repositoryCount":1}`)

		statuses, err := GetLatestReportStatuses([]string{"import", "update"}, day)
		if err != nil {
			t.Fatalf("GetLatestReportStatuses returned error: %v", err)
		}

		for _, job := range []string{"import", "update"} {
			if got := statuses[job]; got != "missing" {
				t.Fatalf("statuses[%q] = %q, want %q", job, got, "missing")
			}
		}
	})

	t.Run("returns error for invalid report data", func(t *testing.T) {
		setupTestDB(t)

		day := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC)
		insertReport(t, "import", day.Add(time.Hour), `not-json`)

		_, err := GetLatestReportStatuses([]string{"import"}, day)
		if err == nil {
			t.Fatal("GetLatestReportStatuses error = nil, want error")
		}
	})
}

func TestGetLatestReports(t *testing.T) {
	t.Run("returns the latest full report payloads", func(t *testing.T) {
		setupTestDB(t)

		day := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC)
		insertReport(t, "import", day.Add(time.Hour), `{"repositoryCount":1}`)
		insertReport(t, "import", day.Add(2*time.Hour), `{"repositoryCount":2}`)

		reports, err := GetLatestReports([]string{"import", "generate"}, day)
		if err != nil {
			t.Fatalf("GetLatestReports returned error: %v", err)
		}

		importReport, ok := reports["import"]
		if !ok {
			t.Fatal("reports[\"import\"] missing")
		}

		if importReport.Status != "success" {
			t.Fatalf("status = %q, want %q", importReport.Status, "success")
		}

		if got := importReport.Data["repositoryCount"]; got != float64(2) {
			t.Fatalf("repositoryCount = %v, want 2", got)
		}

		if _, ok := reports["generate"]; ok {
			t.Fatal("reports[\"generate\"] present, want missing")
		}
	})
}

func TestCountRepositoryJobEvents(t *testing.T) {
	t.Run("counts events by status for the requested day", func(t *testing.T) {
		setupTestDB(t)

		day := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC)
		insertRepositoryJobEvent(t, 1, "generate", "error", "boom", day.Add(time.Hour))
		insertRepositoryJobEvent(t, 2, "generate", "error", "bang", day.Add(2*time.Hour))
		insertRepositoryJobEvent(t, 3, "generate", "success", "", day.Add(3*time.Hour))
		insertRepositoryJobEvent(t, 4, "generate", "error", "old", day.Add(-time.Hour))

		counts, err := CountRepositoryJobEvents("generate", day)
		if err != nil {
			t.Fatalf("CountRepositoryJobEvents returned error: %v", err)
		}

		if counts["error"] != 2 {
			t.Fatalf("counts[\"error\"] = %d, want 2", counts["error"])
		}

		if counts["success"] != 1 {
			t.Fatalf("counts[\"success\"] = %d, want 1", counts["success"])
		}
	})
}

func TestListRepositoryJobEventMessages(t *testing.T) {
	t.Run("returns latest messages for the requested day", func(t *testing.T) {
		setupTestDB(t)

		day := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC)
		insertRepositoryJobEvent(t, 1, "generate", "error", "second", day.Add(2*time.Hour))
		insertRepositoryJobEvent(t, 2, "generate", "error", "first", day.Add(time.Hour))
		insertRepositoryJobEvent(t, 3, "generate", "success", "", day.Add(3*time.Hour))

		messages, err := ListRepositoryJobEventMessages("generate", "error", day, 5)
		if err != nil {
			t.Fatalf("ListRepositoryJobEventMessages returned error: %v", err)
		}

		got := strings.Join(messages, ",")
		if got != "first,second" {
			t.Fatalf("messages = %q, want %q", got, "first,second")
		}
	})
}

func insertReport(t *testing.T, job string, reportTime time.Time, data string) {
	t.Helper()

	_, err := db.Exec("INSERT INTO reports (date, job, elapsed_time, data) VALUES (?, ?, ?, ?)", reportTime, job, 1.5, data)
	if err != nil {
		t.Fatalf("insert report: %v", err)
	}
}

func insertRepositoryJobEvent(t *testing.T, repositoryID int64, job string, status string, errorMessage string, createdAt time.Time) {
	t.Helper()

	_, err := db.Exec(`INSERT INTO repositories (id, owner_name, name) VALUES (?, ?, ?)`, repositoryID, "owner", fmt.Sprintf("repo-%d", repositoryID))
	if err != nil {
		t.Fatalf("insert repository: %v", err)
	}

	_, err = db.Exec(`INSERT INTO repository_job_events (repository_id, job, status, error_message, created_at) VALUES (?, ?, ?, ?, ?)`, repositoryID, job, status, errorMessage, createdAt)
	if err != nil {
		t.Fatalf("insert repository job event: %v", err)
	}
}
