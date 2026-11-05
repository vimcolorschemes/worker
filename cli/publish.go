package cli

import (
	"bytes"
	"context"
	"encoding/json"
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

const frontendURL = "https://vimcolorschemes.com"

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

	dispatchToken, ok := os.LookupEnv("GITHUB_DISPATCH_TOKEN")
	if !ok || dispatchToken == "" {
		log.Panic("GITHUB_DISPATCH_TOKEN not found in env")
	}

	responseStatusCode, err := triggerPublishWebhook(webhookURL, dispatchToken)
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

func triggerPublishWebhook(webhookURL string, dispatchToken string) (int, error) {
	body, err := json.Marshal(map[string]string{"event_type": "database-updated"})
	if err != nil {
		return 0, fmt.Errorf("build github dispatch request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build github dispatch request: %w", err)
	}

	req.Header.Set("User-Agent", "vimcolorschemes-worker/publish")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+dispatchToken)

	response, err := publishHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call github dispatch: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, fmt.Errorf("github dispatch returned status %d", response.StatusCode)
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

	message := buildDailyJobSummary(day, reports, publishResult, generateEventCounts, generateErrorMessages, frontendURL)

	if err := notify.PostMessage(ctx, message); err != nil {
		return fmt.Errorf("send job summary notification: %w", err)
	}

	return nil
}

func buildDailyJobSummary(day time.Time, reports map[string]database.JobReport, publishResult map[string]interface{}, generateEventCounts map[string]int, generateErrorMessages []string, frontendURL string) notify.Message {
	subtitle := day.Format("Monday, January 2, 2006")
	if frontendURL != "" {
		subtitle += fmt.Sprintf("  ·  <%s|%s>", frontendURL, strings.TrimPrefix(frontendURL, "https://"))
	}

	blocks := []notify.Block{
		notify.Header("vimcolorschemes daily summary"),
		notify.Context(subtitle),
	}

	unhealthy := 0
	for _, job := range publishRequiredJobs {
		report := reports[job]
		if sectionStatus(report) != "success" {
			unhealthy++
		}
		blocks = append(blocks, notify.Divider())
		blocks = append(blocks, jobSectionBlocks(job, report, job == "generate", generateEventCounts, generateErrorMessages)...)
	}

	blocks = append(blocks, notify.Divider())
	blocks = append(blocks, jobSectionBlocks("publish", database.JobReport{
		Job:    "publish",
		Status: "success",
		Date:   day,
		Data:   publishResult,
	}, false, nil, nil)...)

	text := fmt.Sprintf("vimcolorschemes daily summary %s: all jobs succeeded", day.Format("2006-01-02"))
	if unhealthy > 0 {
		text = fmt.Sprintf("vimcolorschemes daily summary %s: %d of %d jobs need attention", day.Format("2006-01-02"), unhealthy, len(publishRequiredJobs))
	}

	return notify.Message{Text: text, Blocks: blocks}
}

func jobSectionBlocks(job string, report database.JobReport, includeGenerateEvents bool, generateEventCounts map[string]int, generateErrorMessages []string) []notify.Block {
	status := sectionStatus(report)
	heading := fmt.Sprintf("%s *%s*", statusEmoji(status), titleCase(job))

	if report.Job == "" {
		return []notify.Block{notify.Section(heading+"  ·  no report recorded today", nil)}
	}

	if status != "success" {
		heading += "  ·  " + status
	}
	if job != "publish" {
		heading += fmt.Sprintf("  ·  %s  ·  ran at %s", formatDuration(report.ElapsedTime), report.Date.UTC().Format("15:04 UTC"))
	}

	keyLabels := map[string]string{
		"repositoryCount":        "Repositories",
		"repositoryCheckedCount": "Checked",
		"repositoryDroppedCount": "Dropped",
		"repositoryErrorCount":   "Errors",
		"repositoryDeletedCount": "Pruned",
		"responseStatusCode":     "Status code",
		"webhookTriggered":       "Webhook",
		"notificationStatus":     "Notification",
	}

	var fields []string
	for _, key := range []string{"repositoryCount", "repositoryCheckedCount", "repositoryDroppedCount", "repositoryErrorCount", "repositoryDeletedCount", "responseStatusCode", "webhookTriggered", "notificationStatus"} {
		value, ok := report.Data[key]
		if !ok {
			continue
		}
		fields = append(fields, summaryField(keyLabels[key], formatSummaryValue(key, value)))
	}
	if includeGenerateEvents {
		fields = append(fields, summaryField("Event errors", formatInt(int64(generateEventCounts["error"]))))
	}

	blocks := []notify.Block{notify.Section(heading, fields)}

	if errValue, ok := report.Data["error"]; ok {
		blocks = append(blocks, notify.Section("*Error*\n"+notify.Escape(fmt.Sprintf("%v", errValue)), nil))
	}
	if errValue, ok := report.Data["notificationError"]; ok {
		blocks = append(blocks, notify.Section("*Notification error*\n"+notify.Escape(fmt.Sprintf("%v", errValue)), nil))
	}
	if links := repositoryLinks(report.Data["repositoryDeletedNames"]); links != "" {
		blocks = append(blocks, notify.Section("*Pruned repositories*\n"+links, nil))
	}
	if links := repositoryLinks(report.Data["repositoryDroppedNames"]); links != "" {
		blocks = append(blocks, notify.Section("*Dropped repositories*\n"+links, nil))
	}

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
		lines := make([]string, len(errorSamples))
		for i, message := range errorSamples {
			lines[i] = "• " + notify.Escape(message)
		}
		blocks = append(blocks, notify.Section("*Recent errors*\n"+strings.Join(lines, "\n"), nil))
	}

	return blocks
}

func summaryField(label string, value string) string {
	return fmt.Sprintf("*%s*\n%s", label, notify.Escape(value))
}

func statusEmoji(status string) string {
	switch status {
	case "success":
		return ":white_check_mark:"
	case "error":
		return ":x:"
	case "missing":
		return ":warning:"
	default:
		return ":grey_question:"
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

func repositoryLinks(value interface{}) string {
	var names []string
	switch typed := value.(type) {
	case []interface{}:
		for _, name := range typed {
			names = append(names, fmt.Sprintf("%v", name))
		}
	case []string:
		names = typed
	}

	links := make([]string, len(names))
	for i, name := range names {
		escaped := notify.Escape(name)
		links[i] = fmt.Sprintf("<https://github.com/%s|%s>", escaped, escaped)
	}
	return strings.Join(links, "  ·  ")
}
