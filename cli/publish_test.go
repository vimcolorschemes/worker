package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/notify"
)

func TestValidatePublishPrerequisites(t *testing.T) {
	t.Run("accepts all successful jobs", func(t *testing.T) {
		statuses := map[string]string{
			"import":   "success",
			"update":   "success",
			"generate": "success",
		}

		if err := validatePublishPrerequisites(statuses); err != nil {
			t.Fatalf("validatePublishPrerequisites returned error: %v", err)
		}
	})

	t.Run("rejects missing or failed jobs", func(t *testing.T) {
		statuses := map[string]string{
			"import":   "success",
			"update":   "error",
			"generate": "missing",
		}

		err := validatePublishPrerequisites(statuses)
		if err == nil {
			t.Fatal("validatePublishPrerequisites error = nil, want error")
		}

		for _, want := range []string{"update=error", "generate=missing"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %q, want it to contain %q", err.Error(), want)
			}
		}
	})
}

func TestTriggerPublishWebhook(t *testing.T) {
	originalClient := publishHTTPClient
	t.Cleanup(func() {
		publishHTTPClient = originalClient
	})

	t.Run("posts to webhook", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
			}

			if got := r.Header.Get("User-Agent"); got != "vimcolorschemes-worker/publish" {
				t.Fatalf("User-Agent = %q, want %q", got, "vimcolorschemes-worker/publish")
			}

			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want %q", got, "application/json")
			}

			if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
				t.Fatalf("Accept = %q, want %q", got, "application/vnd.github+json")
			}

			if got := r.Header.Get("Authorization"); got != "Bearer test-dispatch-token" {
				t.Fatalf("Authorization = %q, want %q", got, "Bearer test-dispatch-token")
			}

			rawBody, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			var payload map[string]string
			if err := json.Unmarshal(rawBody, &payload); err != nil {
				t.Fatalf("unmarshal body %q: %v", rawBody, err)
			}

			if payload["event_type"] != "database-updated" {
				t.Fatalf("event_type = %q, want %q", payload["event_type"], "database-updated")
			}

			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		publishHTTPClient = &http.Client{Timeout: time.Second}

		statusCode, err := triggerPublishWebhook(server.URL, "test-dispatch-token")
		if err != nil {
			t.Fatalf("triggerPublishWebhook returned error: %v", err)
		}

		if statusCode != http.StatusCreated {
			t.Fatalf("statusCode = %d, want %d", statusCode, http.StatusCreated)
		}
	})

	t.Run("returns error for non-2xx response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		publishHTTPClient = &http.Client{Timeout: time.Second}

		statusCode, err := triggerPublishWebhook(server.URL, "test-dispatch-token")
		if err == nil {
			t.Fatal("triggerPublishWebhook error = nil, want error")
		}

		if statusCode != http.StatusBadGateway {
			t.Fatalf("statusCode = %d, want %d", statusCode, http.StatusBadGateway)
		}
	})
}

func TestBuildDailyJobSummary(t *testing.T) {
	day := time.Date(2026, time.March, 29, 0, 0, 0, 0, time.UTC)
	prunedNames := []interface{}{}
	for i := 1; i <= 11; i++ {
		prunedNames = append(prunedNames, fmt.Sprintf("owner%d/repo%d", i, i))
	}
	reports := map[string]database.JobReport{
		"import": {
			Job:         "import",
			Date:        day.Add(time.Hour),
			ElapsedTime: 12.5,
			Status:      "success",
			Data: map[string]interface{}{
				"repositoryCount":        float64(2934),
				"repositoryCheckedCount": float64(40),
				"repositoryAddedCount":   float64(2),
				"repositoryAddedNames":   []interface{}{"alice/moss", "bob/ember"},
				"repositoryDroppedCount": float64(38),
				"repositoryDroppedNames": []interface{}{"owner4/dotfiles"},
			},
		},
		"update": {
			Job:         "update",
			Date:        day.Add(2 * time.Hour),
			ElapsedTime: 918.2,
			Status:      "success",
			Data: map[string]interface{}{
				"repositoryCount":        float64(2934),
				"repositoryErrorCount":   float64(0),
				"repositoryDeletedCount": float64(11),
				"repositoryDeletedNames": prunedNames,
			},
		},
		"generate": {
			Job:         "generate",
			Date:        day.Add(3 * time.Hour),
			ElapsedTime: 25.4,
			Status:      "success",
			Data: map[string]interface{}{
				"repositoryCount":        float64(120),
				"repositoryErrorCount":   float64(1),
				"repositoryErrorSamples": []interface{}{"owner/repo: boom"},
				"colorschemeCount":       float64(1043),
			},
		},
	}
	publishResult := map[string]interface{}{
		"responseStatusCode": float64(201),
		"webhookTriggered":   true,
	}

	message := buildDailyJobSummary(day, reports, publishResult, frontendURL)

	if message.Text != "vimcolorschemes daily summary 2026-03-29: needs attention  ·  +2 repos  ·  -11 pruned  ·  1 error" {
		t.Fatalf("Text = %q", message.Text)
	}
	if message.Blocks[0]["type"] != "header" {
		t.Fatalf("first block = %v, want a header", message.Blocks[0])
	}

	body := strings.Join(blockTexts(message.Blocks), "\n")
	for _, want := range []string{
		"Sunday, March 29, 2026  ·  <https://vimcolorschemes.com|vimcolorschemes.com>",
		":warning: *needs attention*  ·  +2 repos  ·  -11 pruned  ·  1 error",
		":white_check_mark: *Import*",
		"*Candidates*\n40",
		"*New*\n+2",
		"*Filtered out*\n38",
		"*New*\n<https://github.com/alice/moss|alice/moss>  ·  <https://github.com/bob/ember|bob/ember>",
		":white_check_mark: *Update*",
		"*Total in DB*\n2,934",
		"*Pruned (404)*\n11",
		"*Pruned (404)*\n11, not listed here. Full list in the `reports` row for update on 2026-03-29 and in the worker logs.",
		":white_check_mark: *Generate*",
		"*Processed*\n120",
		"*OK*\n119",
		"*Colorschemes generated*\n1,043",
		"*Errors*\n• owner/repo: boom",
		":white_check_mark: *Publish*  ·  deployed",
		"Import 12.5s at 01:00 UTC  ·  Update 15m 18s at 02:00 UTC  ·  Generate 25.4s at 03:00 UTC",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("summary = %q, want it to contain %q", body, want)
		}
	}

	for _, unwanted := range []string{"*Errors*\n0", "owner4/dotfiles", "owner1/repo1", "Status code"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("summary = %q, want it not to contain %q", body, unwanted)
		}
	}

	t.Run("shows a quiet day as all green", func(t *testing.T) {
		quiet := map[string]database.JobReport{
			"import":   {Job: "import", Date: day, Status: "success", Data: map[string]interface{}{"repositoryCheckedCount": float64(3), "repositoryDroppedCount": float64(3)}},
			"update":   {Job: "update", Date: day, Status: "success", Data: map[string]interface{}{"repositoryCount": float64(10)}},
			"generate": {Job: "generate", Date: day, Status: "success", Data: map[string]interface{}{}},
		}

		message := buildDailyJobSummary(day, quiet, publishResult, frontendURL)

		if message.Text != "vimcolorschemes daily summary 2026-03-29: all green  ·  0 repos  ·  0 pruned  ·  0 errors" {
			t.Fatalf("Text = %q", message.Text)
		}
	})

	t.Run("counts a capped list instead of showing part of it", func(t *testing.T) {
		capped := map[string]database.JobReport{
			"generate": {Job: "generate", Date: day, Status: "success", Data: map[string]interface{}{
				"repositoryCount":        float64(10),
				"repositoryErrorCount":   float64(7),
				"repositoryErrorSamples": []interface{}{"a/one: boom", "a/two: boom", "a/three: boom", "a/four: boom", "a/five: boom"},
			}},
		}

		body := strings.Join(blockTexts(buildDailyJobSummary(day, capped, publishResult, frontendURL).Blocks), "\n")

		if strings.Contains(body, "a/one: boom") {
			t.Fatalf("summary = %q, want no partial error list", body)
		}
		if !strings.Contains(body, "*Errors*\n7, not listed here.") {
			t.Fatalf("summary = %q, want the error count with a pointer", body)
		}
	})

	t.Run("flags missing reports", func(t *testing.T) {
		delete(reports, "update")

		message := buildDailyJobSummary(day, reports, publishResult, frontendURL)

		if message.Text != "vimcolorschemes daily summary 2026-03-29: 1 of 3 jobs need attention  ·  +2 repos  ·  0 pruned  ·  1 error" {
			t.Fatalf("Text = %q", message.Text)
		}
		if body := strings.Join(blockTexts(message.Blocks), "\n"); !strings.Contains(body, ":warning: *Update*  ·  no report recorded today") {
			t.Fatalf("summary = %q, want the missing update report flagged", body)
		}
	})
}

func blockTexts(value interface{}) []string {
	var texts []string
	switch typed := value.(type) {
	case []notify.Block:
		for _, block := range typed {
			texts = append(texts, blockTexts(block)...)
		}
	case notify.Block:
		for key, child := range typed {
			if text, ok := child.(string); ok && key == "text" {
				texts = append(texts, text)
				continue
			}
			texts = append(texts, blockTexts(child)...)
		}
	}
	return texts
}
