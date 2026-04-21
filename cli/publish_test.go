package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
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

			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		publishHTTPClient = &http.Client{Timeout: time.Second}

		statusCode, err := triggerPublishWebhook(server.URL)
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

		statusCode, err := triggerPublishWebhook(server.URL)
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
				"repositoryCount": float64(2934),
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

	body := buildDailyJobSummary(day, reports, publishResult, map[string]int{"error": 1}, []string{"clone failed"}, frontendURL)

	for _, want := range []string{
		"vimcolorschemes — Daily Summary",
		"2026-03-29",
		"https://vimcolorschemes.com",
		"Import · success",
		"Repositories:  2,934",
		"Duration:      12.5s",
		"Update · success",
		"Errors:        2",
		"Pruned:        3",
		"Generate · success",
		"Event errors:  1",
		"Recent errors:",
		"- clone failed",
		"- owner/repo: boom",
		"Publish · success",
		"Status code:   201",
		"Webhook:       triggered",
		"Notification:  sent",
		"Job statuses:  generate=success, import=success, update=success",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("summary = %q, want it to contain %q", body, want)
		}
	}
}
