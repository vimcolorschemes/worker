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

// PublishJobNotification posts a job notification to the Slack incoming webhook.
func PublishJobNotification(ctx context.Context, subject string, body string) error {
	webhookURL := strings.TrimSpace(os.Getenv("SLACK_WEBHOOK_URL"))
	if webhookURL == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL not found in env")
	}

	text := fmt.Sprintf("*%s*\n```\n%s\n```", slackEscaper.Replace(subject), slackEscaper.Replace(body))
	payload, err := json.Marshal(map[string]string{"text": text})
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
