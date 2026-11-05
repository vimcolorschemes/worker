package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostMessage(t *testing.T) {
	t.Run("posts text and blocks to the slack webhook", func(t *testing.T) {
		var got Message
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("Method = %q, want POST", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			_, _ = w.Write([]byte("ok"))
		}))
		t.Cleanup(server.Close)
		t.Setenv("SLACK_WEBHOOK_URL", server.URL)

		message := Message{Text: "fallback", Blocks: []Block{Header("Daily Summary"), Divider()}}
		if err := PostMessage(context.Background(), message); err != nil {
			t.Fatalf("PostMessage returned error: %v", err)
		}

		if got.Text != "fallback" {
			t.Fatalf("text = %q, want %q", got.Text, "fallback")
		}
		if len(got.Blocks) != 2 || got.Blocks[0]["type"] != "header" || got.Blocks[1]["type"] != "divider" {
			t.Fatalf("blocks = %v, want header and divider", got.Blocks)
		}
	})

	t.Run("returns error when webhook url is missing", func(t *testing.T) {
		t.Setenv("SLACK_WEBHOOK_URL", "")

		if err := PostMessage(context.Background(), Message{Text: "hello"}); err == nil {
			t.Fatal("PostMessage error = nil, want error")
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid_blocks"))
		}))
		t.Cleanup(server.Close)
		t.Setenv("SLACK_WEBHOOK_URL", server.URL)

		err := PostMessage(context.Background(), Message{Text: "hello"})
		if err == nil || !strings.Contains(err.Error(), "invalid_blocks") {
			t.Fatalf("PostMessage error = %v, want it to include the slack response", err)
		}
	})
}

func TestEscape(t *testing.T) {
	if got := Escape("a < b & c > d"); got != "a &lt; b &amp; c &gt; d" {
		t.Fatalf("Escape = %q", got)
	}
}

func TestSection(t *testing.T) {
	block := Section(strings.Repeat("x", sectionTextLimit+10), []string{"*Label*\nvalue"})

	text := block["text"].(Block)["text"].(string)
	if n := len([]rune(text)); n != sectionTextLimit {
		t.Fatalf("section text length = %d, want %d", n, sectionTextLimit)
	}
	fields := block["fields"].([]Block)
	if len(fields) != 1 || fields[0]["text"] != "*Label*\nvalue" {
		t.Fatalf("fields = %v", fields)
	}
}
