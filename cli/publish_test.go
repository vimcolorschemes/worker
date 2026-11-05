package cli

import (
	"encoding/json"
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
	reports := map[string]database.JobReport{
		"import": {
			Job:         "import",
			Date:        day.Add(time.Hour),
			ElapsedTime: 12.5,
			Status:      "success",
			Data: map[string]interface{}{
				"repositoryCount":        float64(2934),
				"repositoryCheckedCount": float64(40),
				"repositoryDroppedCount": float64(2),
				"repositoryDroppedNames": []string{"owner4/dotfiles", "owner5/tutorial"},
			},
		},
		"update": {
			Job:         "update",
			Date:        day.Add(2 * time.Hour),
			ElapsedTime: 18.2,
			Status:      "success",
			Data: map[string]interface{}{
				"repositoryCount":        float64(2934),
				"repositoryErrorCount":   float64(2),
				"repositoryDeletedCount": float64(3),
				"repositoryDeletedNames": []interface{}{"owner1/repo1", "owner2/repo2", "owner3/repo3"},
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
			},
		},
	}
	publishResult := map[string]interface{}{
		"responseStatusCode": float64(201),
		"webhookTriggered":   true,
		"notificationStatus": "sent",
		"jobStatuses": map[string]interface{}{
			"import":   "success",
			"update":   "success",
			"generate": "success",
		},
	}

	message := buildDailyJobSummary(day, reports, publishResult, map[string]int{"error": 1}, []string{"clone failed"}, frontendURL)

	if message.Text != "vimcolorschemes daily summary 2026-03-29: all jobs succeeded" {
		t.Fatalf("Text = %q", message.Text)
	}
	if message.Blocks[0]["type"] != "header" {
		t.Fatalf("first block = %v, want a header", message.Blocks[0])
	}

	body := strings.Join(blockTexts(message.Blocks), "\n")
	for _, want := range []string{
		"vimcolorschemes daily summary",
		"Sunday, March 29, 2026  ·  <https://vimcolorschemes.com|vimcolorschemes.com>",
		":white_check_mark: *Import*  ·  12.5s  ·  ran at 01:00 UTC",
		"*Repositories*\n2,934",
		"*Checked*\n40",
		"*Dropped*\n2",
		"*Dropped repositories*\n<https://github.com/owner4/dotfiles|owner4/dotfiles>  ·  <https://github.com/owner5/tutorial|owner5/tutorial>",
		":white_check_mark: *Update*  ·  18.2s  ·  ran at 02:00 UTC",
		"*Errors*\n2",
		"*Pruned*\n3",
		"*Pruned repositories*\n<https://github.com/owner1/repo1|owner1/repo1>",
		":white_check_mark: *Generate*",
		"*Event errors*\n1",
		"*Recent errors*\n• clone failed\n• owner/repo: boom",
		":white_check_mark: *Publish*",
		"*Status code*\n201",
		"*Webhook*\ntriggered",
		"*Notification*\nsent",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("summary = %q, want it to contain %q", body, want)
		}
	}

	t.Run("flags missing reports", func(t *testing.T) {
		delete(reports, "update")

		message := buildDailyJobSummary(day, reports, publishResult, nil, nil, frontendURL)

		if message.Text != "vimcolorschemes daily summary 2026-03-29: 1 of 3 jobs need attention" {
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
