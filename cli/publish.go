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

const (
	frontendURL    = "https://vimcolorschemes.com"
	summaryDivider = "----------------------------------------"
)

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

	subject := fmt.Sprintf("vimcolorschemes daily summary %s", day.Format("2006-01-02"))
	body := buildDailyJobSummary(day, reports, publishResult, generateEventCounts, generateErrorMessages, frontendURL)

	if err := notify.PublishJobNotification(ctx, subject, body); err != nil {
		return fmt.Errorf("send job summary notification: %w", err)
	}

	return nil
}

func buildDailyJobSummary(day time.Time, reports map[string]database.JobReport, publishResult map[string]interface{}, generateEventCounts map[string]int, generateErrorMessages []string, frontendURL string) string {
	var b strings.Builder

	b.WriteString("vimcolorschemes — Daily Summary\n")
	b.WriteString(day.Format("2006-01-02"))
	b.WriteString("\n")
	if frontendURL != "" {
		b.WriteString(frontendURL)
		b.WriteString("\n")
	}

	for _, job := range []string{"import", "update", "generate"} {
		b.WriteString("\n")
		writeJobSection(&b, job, reports[job], job == "generate", generateEventCounts, generateErrorMessages)
	}

	b.WriteString("\n")
	writeJobSection(&b, "publish", database.JobReport{
		Job:    "publish",
		Status: "success",
		Date:   day,
		Data:   publishResult,
	}, false, nil, nil)

	return strings.TrimRight(b.String(), "\n")
}

func writeJobSection(b *strings.Builder, job string, report database.JobReport, includeGenerateEvents bool, generateEventCounts map[string]int, generateErrorMessages []string) {
	b.WriteString(summaryDivider)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s · %s\n", titleCase(job), sectionStatus(report)))
	b.WriteString(summaryDivider)
	b.WriteString("\n")

	if report.Job == "" {
		b.WriteString("  (no report recorded today)\n")
		return
	}

	rows := [][2]string{}
	if job != "publish" {
		rows = append(rows, [2]string{"Ran at", report.Date.UTC().Format("2006-01-02 15:04:05 UTC")})
		rows = append(rows, [2]string{"Duration", formatDuration(report.ElapsedTime)})
	}

	keyLabels := map[string]string{
		"repositoryCount":        "Repositories",
		"repositoryErrorCount":   "Errors",
		"repositoryDeletedCount": "Pruned",
		"responseStatusCode":     "Status code",
		"webhookTriggered":       "Webhook",
		"notificationStatus":     "Notification",
	}

	for _, key := range []string{"repositoryCount", "repositoryErrorCount", "repositoryDeletedCount", "responseStatusCode", "webhookTriggered", "notificationStatus"} {
		value, ok := report.Data[key]
		if !ok {
			continue
		}
		rows = append(rows, [2]string{keyLabels[key], formatSummaryValue(key, value)})
	}

	if statuses, ok := report.Data["jobStatuses"].(map[string]string); ok {
		rows = append(rows, [2]string{"Job statuses", formatStringMap(statuses)})
	} else if statuses, ok := report.Data["jobStatuses"].(map[string]interface{}); ok {
		rows = append(rows, [2]string{"Job statuses", formatInterfaceMap(statuses)})
	}

	if errValue, ok := report.Data["error"]; ok {
		rows = append(rows, [2]string{"Error", fmt.Sprintf("%v", errValue)})
	}
	if errValue, ok := report.Data["notificationError"]; ok {
		rows = append(rows, [2]string{"Notification error", fmt.Sprintf("%v", errValue)})
	}

	if includeGenerateEvents {
		rows = append(rows, [2]string{"Event errors", formatInt(int64(generateEventCounts["error"]))})
	}

	writeSummaryRows(b, rows)

	var errorSamples []string
	if includeGenerateEvents {
		errorSamples = append(errorSamples, generateErrorMessages...)
	}
	if samples, ok := report.Data["repositoryErrorSamples"].([]interface{}); ok {
		for _, sample := range samples {
			errorSamples = append(errorSamples, fmt.Sprintf("%v", sample))
		}
	}
	if len(errorSamples) > 0 {
		b.WriteString("\n  Recent errors:\n")
		for _, message := range errorSamples {
			b.WriteString(fmt.Sprintf("    - %s\n", message))
		}
	}
}

func writeSummaryRows(b *strings.Builder, rows [][2]string) {
	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("  %-*s  %s\n", width+1, row[0]+":", row[1]))
	}
}

func sectionStatus(report database.JobReport) string {
	if report.Job == "" {
		return "missing"
	}
	if report.Status == "" {
		return "unknown"
	}
	return report.Status
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	d := time.Duration(seconds * float64(time.Second))
	hours := int(d / time.Hour)
	minutes := int((d % time.Hour) / time.Minute)
	secs := int((d % time.Minute) / time.Second)
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	}
	return fmt.Sprintf("%dm %ds", minutes, secs)
}

func formatSummaryValue(key string, value interface{}) string {
	if v, ok := value.(bool); ok {
		if key == "webhookTriggered" {
			if v {
				return "triggered"
			}
			return "not triggered"
		}
		return fmt.Sprintf("%t", v)
	}

	useThousands := key == "repositoryCount" || key == "repositoryErrorCount" || key == "repositoryDeletedCount"

	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			if useThousands {
				return formatInt(int64(v))
			}
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case int:
		if useThousands {
			return formatInt(int64(v))
		}
		return fmt.Sprintf("%d", v)
	case int64:
		if useThousands {
			return formatInt(v)
		}
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatInt(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	raw := fmt.Sprintf("%d", n)
	if len(raw) <= 3 {
		return sign + raw
	}
	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	return sign + strings.Join(parts, ",")
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
