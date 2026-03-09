package database

import (
	"strings"
	"testing"
)

func TestCreateReport(t *testing.T) {
	t.Run("inserts a report row", func(t *testing.T) {
		setupTestDB(t)
		CreateReport("import", 1.5, map[string]interface{}{})

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
		CreateReport("import", 1.5, map[string]interface{}{"count": 5})

		var data string
		err := db.QueryRow("SELECT data FROM reports").Scan(&data)
		if err != nil {
			t.Fatalf("query data: %v", err)
		}
		if !strings.Contains(data, "count") {
			t.Fatalf("data = %q, want it to contain %q", data, "count")
		}
	})
}
