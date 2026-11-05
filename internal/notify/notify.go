package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Slack treats these as control characters in message text.
var slackEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// Slack rejects section text longer than this.
const sectionTextLimit = 3000

// Block is a Slack Block Kit block.
type Block map[string]any

// Message is a Slack message. Text is the notification fallback shown when blocks cannot render.
type Message struct {
	Text   string  `json:"text"`
	Blocks []Block `json:"blocks,omitempty"`
}

// Escape makes arbitrary text safe to embed in Slack mrkdwn.
func Escape(s string) string {
	return slackEscaper.Replace(s)
}

func Header(text string) Block {
	return Block{"type": "header", "text": Block{"type": "plain_text", "text": text, "emoji": true}}
}

func Context(markdown string) Block {
	return Block{"type": "context", "elements": []Block{mrkdwn(markdown)}}
}

func Divider() Block {
	return Block{"type": "divider"}
}

func Section(markdown string, fields []string) Block {
	block := Block{"type": "section", "text": mrkdwn(truncate(markdown, sectionTextLimit))}
	if len(fields) > 0 {
		fieldBlocks := make([]Block, len(fields))
		for i, field := range fields {
			fieldBlocks[i] = mrkdwn(field)
		}
		block["fields"] = fieldBlocks
	}
	return block
}

func mrkdwn(text string) Block {
	return Block{"type": "mrkdwn", "text": text}
}

func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit-1]) + "…"
}

// PostMessage posts a message to the Slack incoming webhook in SLACK_WEBHOOK_URL.
func PostMessage(ctx context.Context, message Message) error {
	webhookURL := strings.TrimSpace(os.Getenv("SLACK_WEBHOOK_URL"))
	if webhookURL == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL not found in env")
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("build slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post slack notification: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("slack webhook returned status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}
