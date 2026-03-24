package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
)

var publishRequiredJobs = []string{"import", "update", "generate"}

var publishNow = func() time.Time {
	return time.Now().UTC()
}

var publishHTTPClient = &http.Client{Timeout: 30 * time.Second}

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

	return map[string]interface{}{
		"jobStatuses":        statuses,
		"responseStatusCode": responseStatusCode,
		"webhookTriggered":   true,
	}
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
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, fmt.Errorf("vercel webhook returned status %d", response.StatusCode)
	}

	return response.StatusCode, nil
}
