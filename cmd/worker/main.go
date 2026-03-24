package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	rdebug "runtime/debug"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/cli"
	"github.com/vimcolorschemes/worker/internal/database"
)

type jobRunner func(force bool, debug bool, repoKey string) map[string]interface{}

var jobRunnerMap = map[string]jobRunner{
	"import":   cli.Import,
	"update":   cli.Update,
	"generate": cli.Generate,
	"publish":  cli.Publish,
}

func main() {
	job, force, debug, repoKey, err := getJobArgs(os.Args)

	log.Printf("Running %s", job)

	if force {
		log.Print("--force option activated")
	}

	if repoKey != "" {
		log.Printf("--repo %s option activated", repoKey)
	}

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	runner := jobRunnerMap[job]
	if runner == nil {
		log.Print(job, " is not a valid job")
		os.Exit(1)
	}

	startTime := time.Now()

	fmt.Println()

	data, stackTrace, runErr := runJobWithRecovery(runner, force, debug, repoKey)

	elapsedTime := time.Since(startTime)
	if runErr != nil {
		reportData := map[string]interface{}{
			"status": "error",
			"error":  runErr.Error(),
		}
		if stackTrace != "" {
			reportData["stackTrace"] = stackTrace
		}

		if err := database.CreateReport(job, elapsedTime.Seconds(), reportData); err != nil {
			log.Printf("Error creating report: %s", err)
		}

		fmt.Println()
		log.Printf("Elapsed time: %s\n", elapsedTime)
		log.Print(":wq")
		log.Printf("Job failed: %s", runErr)
		if stackTrace != "" {
			log.Print(stackTrace)
		}
		os.Exit(1)
	}

	if err := database.CreateReport(job, elapsedTime.Seconds(), data); err != nil {
		log.Printf("Error creating report: %s", err)
	}

	fmt.Println()
	log.Printf("Elapsed time: %s\n", elapsedTime)
	log.Print(":wq")
}

func runJobWithRecovery(runner jobRunner, force bool, debug bool, repoKey string) (data map[string]interface{}, stackTrace string, runErr error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		runErr = fmt.Errorf("panic: %v", recovered)
		stackTrace = string(rdebug.Stack())
	}()

	data = runner(force, debug, repoKey)
	return data, "", nil
}

func getJobArgs(osArgs []string) (string, bool, bool, string, error) {
	if len(osArgs) < 2 {
		return "", false, false, "", errors.New("please provide an argument")
	}

	job := osArgs[1]

	if len(osArgs) < 3 {
		return job, false, false, "", nil
	}

	args := osArgs[2:]

	forceIndex := getArgIndex(args, "--force")
	force := forceIndex != -1

	debugIndex := getArgIndex(args, "--debug")
	debug := debugIndex != -1

	repoIndex := getArgIndex(args, "--repo")
	if repoIndex == -1 || len(args) < repoIndex+1 {
		return osArgs[1], force, debug, "", nil
	}

	repoKey := strings.ToLower(args[repoIndex+1])
	return osArgs[1], force, debug, repoKey, nil
}

func getArgIndex(args []string, target string) int {
	for index, arg := range args {
		if arg == target {
			return index
		}
	}
	return -1
}
