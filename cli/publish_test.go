package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
