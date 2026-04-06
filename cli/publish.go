package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/notify"
)

var publishRequiredJobs = []string{"import", "update", "generate"}

var publishNow = func() time.Time {
	return time.Now().UTC()
}

var publishHTTPClient = &http.Client{Timeout: 30 * time.Second}

var publishNotificationNow = func() time.Time {
	return time.Now().UTC()
}

// Publish triggers a frontend deployment after the daily jobs succeed.
func Publish(_force bool, _debug bool, _repoKey string) map[string]interface{} {
	statuses, err := database.GetLatestReportStatuses(publishRequiredJobs, publishNow())
	if err != nil {
		log.Panic(err)
	}

	if err := validatePublishPrerequisites(statuses); err != nil {
		log.Panic(err)
	}

	webhookURL, ok := os.LookupEnv("PUBLISH_WEBHOOK_URL")
	if !ok || webhookURL == "" {
		log.Panic("PUBLISH_WEBHOOK_URL not found in env")
	}

	responseStatusCode, err := triggerPublishWebhook(webhookURL)
	if err != nil {
		log.Panic(err)
	}

	result := map[string]interface{}{
		"jobStatuses":        statuses,
		"responseStatusCode": responseStatusCode,
		"webhookTriggered":   true,
	}

	if err := sendDailyJobSummary(context.Background(), result, publishNotificationNow()); err != nil {
		log.Printf("Error sending daily job summary: %s", err)
		result["notificationStatus"] = "error"
		result["notificationError"] = err.Error()
	} else {
		result["notificationStatus"] = "sent"
	}

	return result
}

func validatePublishPrerequisites(statuses map[string]string) error {
	var incompleteJobs []string

	for _, job := range publishRequiredJobs {
		if statuses[job] != "success" {
			incompleteJobs = append(incompleteJobs, fmt.Sprintf("%s=%s", job, statuses[job]))
		}
	}

	if len(incompleteJobs) == 0 {
		return nil
	}

	sort.Strings(incompleteJobs)
	return fmt.Errorf("publish requires successful reports today for import, update, and generate; got %s", strings.Join(incompleteJobs, ", "))
}

func triggerPublishWebhook(webhookURL string) (int, error) {
	req, err := http.NewRequest(http.MethodPost, webhookURL, nil)
	if err != nil {
		return 0, fmt.Errorf("build vercel webhook request: %w", err)
	}

	req.Header.Set("User-Agent", "vimcolorschemes-worker/publish")

	response, err := publishHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call vercel webhook: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, fmt.Errorf("vercel webhook returned status %d", response.StatusCode)
	}

	return response.StatusCode, nil
}

func sendDailyJobSummary(ctx context.Context, publishResult map[string]interface{}, day time.Time) error {
	reports, err := database.GetLatestReports([]string{"import", "update", "generate"}, day)
	if err != nil {
		return fmt.Errorf("load daily reports: %w", err)
	}

	generateEventCounts, err := database.CountRepositoryJobEvents("generate", day)
	if err != nil {
		return fmt.Errorf("load generate event counts: %w", err)
	}

	generateErrorMessages, err := database.ListRepositoryJobEventMessages("generate", "error", day, 5)
	if err != nil {
		return fmt.Errorf("load generate event messages: %w", err)
	}

	subject := fmt.Sprintf("vimcolorschemes prod daily summary %s", day.Format("2006-01-02"))
	body := buildDailyJobSummary(day, reports, publishResult, generateEventCounts, generateErrorMessages)

	if err := notify.PublishJobNotification(ctx, subject, body); err != nil {
		return fmt.Errorf("send job summary notification: %w", err)
	}

	return nil
}

func buildDailyJobSummary(day time.Time, reports map[string]database.JobReport, publishResult map[string]interface{}, generateEventCounts map[string]int, generateErrorMessages []string) string {
	lines := []string{
		fmt.Sprintf("vimcolorschemes production daily summary for %s", day.Format("2006-01-02")),
		"",
	}

	for _, job := range []string{"import", "update", "generate"} {
		lines = append(lines, formatJobSummary(job, reports[job], job == "generate", generateEventCounts, generateErrorMessages)...)
		lines = append(lines, "")
	}

	publishReport := database.JobReport{
		Job:    "publish",
		Status: "success",
		Date:   day,
		Data:   publishResult,
	}
	lines = append(lines, formatJobSummary("publish", publishReport, false, nil, nil)...)

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatJobSummary(job string, report database.JobReport, includeGenerateEvents bool, generateEventCounts map[string]int, generateErrorMessages []string) []string {
	if report.Job == "" {
		return []string{fmt.Sprintf("%s: missing", job)}
	}

	lines := []string{fmt.Sprintf("%s: %s", job, report.Status)}
	lines = append(lines, fmt.Sprintf("  ran_at: %s", report.Date.UTC().Format(time.RFC3339)))
	lines = append(lines, fmt.Sprintf("  elapsed_seconds: %.3f", report.ElapsedTime))

	for _, key := range []string{"repositoryCount", "repositoryErrorCount", "repositoryDeletedCount", "responseStatusCode", "webhookTriggered", "notificationStatus"} {
		if value, ok := report.Data[key]; ok {
			lines = append(lines, fmt.Sprintf("  %s: %v", key, value))
		}
	}

	if statuses, ok := report.Data["jobStatuses"].(map[string]string); ok {
		lines = append(lines, fmt.Sprintf("  jobStatuses: %s", formatStringMap(statuses)))
	} else if statuses, ok := report.Data["jobStatuses"].(map[string]interface{}); ok {
		lines = append(lines, fmt.Sprintf("  jobStatuses: %s", formatInterfaceMap(statuses)))
	}

	if errValue, ok := report.Data["error"]; ok {
		lines = append(lines, fmt.Sprintf("  error: %v", errValue))
	}
	if errValue, ok := report.Data["notificationError"]; ok {
		lines = append(lines, fmt.Sprintf("  notificationError: %v", errValue))
	}

	if includeGenerateEvents {
		lines = append(lines, fmt.Sprintf("  repositoryEventErrors: %d", generateEventCounts["error"]))
		for _, message := range generateErrorMessages {
			lines = append(lines, fmt.Sprintf("  sampleError: %s", message))
		}
	}

	if samples, ok := report.Data["repositoryErrorSamples"].([]interface{}); ok {
		for _, sample := range samples {
			lines = append(lines, fmt.Sprintf("  sampleError: %v", sample))
		}
	}

	return lines
}

func formatStringMap(values map[string]string) string {
	parts := make([]string, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		parts = append(parts, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

func formatInterfaceMap(values map[string]interface{}) string {
	parts := make([]string, 0, len(values))
	for _, key := range sortedInterfaceMapKeys(values) {
		parts = append(parts, fmt.Sprintf("%s=%v", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedInterfaceMapKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
