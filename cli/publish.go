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

// Slack lists are all or nothing: a longer list is counted and left to the reports row and the worker logs.
const summaryListLimit = 10

func sendDailyJobSummary(ctx context.Context, publishResult map[string]interface{}, day time.Time) error {
	reports, err := database.GetLatestReports([]string{"import", "update", "generate"}, day)
	if err != nil {
		return fmt.Errorf("load daily reports: %w", err)
	}

	message := buildDailyJobSummary(day, reports, publishResult, frontendURL)

	if err := notify.PostMessage(ctx, message); err != nil {
		return fmt.Errorf("send job summary notification: %w", err)
	}

	return nil
}

func buildDailyJobSummary(day time.Time, reports map[string]database.JobReport, publishResult map[string]interface{}, frontendURL string) notify.Message {
	subtitle := day.Format("Monday, January 2, 2006")
	if frontendURL != "" {
		subtitle += fmt.Sprintf("  ·  <%s|%s>", frontendURL, strings.TrimPrefix(frontendURL, "https://"))
	}

	unhealthy := 0
	for _, job := range publishRequiredJobs {
		if sectionStatus(reports[job]) != "success" {
			unhealthy++
		}
	}

	added := reportInt(reports["import"].Data, "repositoryAddedCount")
	pruned := reportInt(reports["update"].Data, "repositoryDeletedCount")
	errorCount := reportInt(reports["update"].Data, "repositoryErrorCount") + reportInt(reports["generate"].Data, "repositoryErrorCount")

	emoji, status := ":white_check_mark:", "all green"
	if unhealthy > 0 {
		emoji, status = ":x:", fmt.Sprintf("%d of %d jobs need attention", unhealthy, len(publishRequiredJobs))
	} else if errorCount > 0 {
		emoji, status = ":warning:", "needs attention"
	}

	netChange := fmt.Sprintf("%s %s  ·  %s pruned  ·  %s %s",
		signedCount(added, "+"), plural(added, "repo"),
		signedCount(pruned, "-"),
		formatInt(errorCount), plural(errorCount, "error"))

	blocks := []notify.Block{
		notify.Header("vimcolorschemes daily summary"),
		notify.Context(subtitle),
		notify.Section(fmt.Sprintf("%s *%s*  ·  %s", emoji, status, netChange), nil),
	}

	for _, job := range publishRequiredJobs {
		blocks = append(blocks, notify.Divider())
		blocks = append(blocks, jobSectionBlocks(job, reports[job])...)
	}

	blocks = append(blocks, notify.Divider(), publishSectionBlock(publishResult))

	if footer := runTimesFooter(reports); footer != "" {
		blocks = append(blocks, notify.Context(footer))
	}

	text := fmt.Sprintf("vimcolorschemes daily summary %s: %s  ·  %s", day.Format("2006-01-02"), status, netChange)

	return notify.Message{Text: text, Blocks: blocks}
}

type summaryCount struct {
	label  string
	value  int64
	prefix string
}

func jobSectionBlocks(job string, report database.JobReport) []notify.Block {
	status := sectionStatus(report)
	heading := fmt.Sprintf("%s *%s*", statusEmoji(status), titleCase(job))

	if report.Job == "" {
		return []notify.Block{notify.Section(heading+"  ·  no report recorded today", nil)}
	}

	if status != "success" {
		heading += "  ·  " + status
	}

	data := report.Data
	var counts []summaryCount
	var lists []notify.Block

	switch job {
	case "import":
		counts = []summaryCount{
			{"Candidates", reportInt(data, "repositoryCheckedCount"), ""},
			{"New", reportInt(data, "repositoryAddedCount"), "+"},
			{"Filtered out", reportInt(data, "repositoryDroppedCount"), ""},
		}
		lists = summaryListBlocks(report, "New", reportStrings(data["repositoryAddedNames"]), reportInt(data, "repositoryAddedCount"), repositoryLinks)
	case "update":
		counts = []summaryCount{
			{"Total in DB", reportInt(data, "repositoryCount"), ""},
			{"Errors", reportInt(data, "repositoryErrorCount"), ""},
			{"Pruned (404)", reportInt(data, "repositoryDeletedCount"), ""},
		}
		lists = summaryListBlocks(report, "Pruned (404)", reportStrings(data["repositoryDeletedNames"]), reportInt(data, "repositoryDeletedCount"), repositoryLinks)
	case "generate":
		processed := reportInt(data, "repositoryCount")
		errorCount := reportInt(data, "repositoryErrorCount")
		counts = []summaryCount{
			{"Processed", processed, ""},
			{"OK", processed - errorCount, ""},
			{"Colorschemes generated", reportInt(data, "colorschemeCount"), ""},
			{"Errors", errorCount, ""},
		}
		lists = summaryListBlocks(report, "Errors", reportStrings(data["repositoryErrorSamples"]), errorCount, errorBullets)
	}

	var fields []string
	for _, count := range counts {
		if count.value > 0 {
			fields = append(fields, summaryField(count.label, count.prefix+formatInt(count.value)))
		}
	}

	blocks := []notify.Block{notify.Section(heading, fields)}

	if errValue, ok := data["error"]; ok {
		blocks = append(blocks, notify.Section("*Error*\n"+notify.Escape(fmt.Sprintf("%v", errValue)), nil))
	}

	return append(blocks, lists...)
}

// summaryListBlocks never shows part of a list: a stored list shorter than its count is only counted.
func summaryListBlocks(report database.JobReport, label string, items []string, total int64, render func([]string) string) []notify.Block {
	switch {
	case total == 0:
		return nil
	case total <= summaryListLimit && int64(len(items)) == total:
		return []notify.Block{notify.Section(fmt.Sprintf("*%s*\n%s", label, render(items)), nil)}
	default:
		return []notify.Block{notify.Section(fmt.Sprintf(
			"*%s*\n%s, not listed here. Full list in the `reports` row for %s on %s and in the worker logs.",
			label, formatInt(total), report.Job, report.Date.UTC().Format("2006-01-02"),
		), nil)}
	}
}

func publishSectionBlock(publishResult map[string]interface{}) notify.Block {
	if triggered, _ := publishResult["webhookTriggered"].(bool); triggered {
		return notify.Section(statusEmoji("success")+" *Publish*  ·  deployed", nil)
	}
	return notify.Section(statusEmoji("error")+" *Publish*  ·  blocked", nil)
}

func runTimesFooter(reports map[string]database.JobReport) string {
	var parts []string
	for _, job := range publishRequiredJobs {
		report := reports[job]
		if report.Job == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s at %s", titleCase(job), formatDuration(report.ElapsedTime), report.Date.UTC().Format("15:04 UTC")))
	}
	return strings.Join(parts, "  ·  ")
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

func signedCount(n int64, sign string) string {
	if n == 0 {
		return "0"
	}
	return sign + formatInt(n)
}

func plural(n int64, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// reportInt reads a count from report data, which holds float64 once it has been through JSON.
func reportInt(data map[string]interface{}, key string) int64 {
	switch v := data[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
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

func reportStrings(value interface{}) []string {
	switch typed := value.(type) {
	case []interface{}:
		names := make([]string, len(typed))
		for i, name := range typed {
			names[i] = fmt.Sprintf("%v", name)
		}
		return names
	case []string:
		return typed
	default:
		return nil
	}
}

func repositoryLinks(names []string) string {
	links := make([]string, len(names))
	for i, name := range names {
		escaped := notify.Escape(name)
		links[i] = fmt.Sprintf("<https://github.com/%s|%s>", escaped, escaped)
	}
	return strings.Join(links, "  ·  ")
}

func errorBullets(messages []string) string {
	lines := make([]string, len(messages))
	for i, message := range messages {
		lines[i] = "• " + notify.Escape(message)
	}
	return strings.Join(lines, "\n")
}
