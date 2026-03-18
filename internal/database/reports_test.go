package database

import (
	"strings"
	"testing"
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
