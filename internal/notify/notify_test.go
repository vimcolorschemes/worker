package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublishJobNotification(t *testing.T) {
	t.Run("posts subject and body to the slack webhook", func(t *testing.T) {
		var gotText string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("Method = %q, want POST", r.Method)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			gotText = payload["text"]
			_, _ = w.Write([]byte("ok"))
		}))
		t.Cleanup(server.Close)
		t.Setenv("SLACK_WEBHOOK_URL", server.URL)

		if err := PublishJobNotification(context.Background(), "Daily Summary", "a < b & c > d"); err != nil {
			t.Fatalf("PublishJobNotification returned error: %v", err)
		}

		want := "*Daily Summary*\n```\na &lt; b &amp; c &gt; d\n```"
		if gotText != want {
			t.Fatalf("text = %q, want %q", gotText, want)
		}
	})

	t.Run("returns error when webhook url is missing", func(t *testing.T) {
		t.Setenv("SLACK_WEBHOOK_URL", "")

		if err := PublishJobNotification(context.Background(), "Daily Summary", "hello"); err == nil {
			t.Fatal("PublishJobNotification error = nil, want error")
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("invalid_token"))
		}))
		t.Cleanup(server.Close)
		t.Setenv("SLACK_WEBHOOK_URL", server.URL)

		if err := PublishJobNotification(context.Background(), "Daily Summary", "hello"); err == nil {
			t.Fatal("PublishJobNotification error = nil, want error")
		}
	})
}
