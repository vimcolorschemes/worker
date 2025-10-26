package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/cli"
	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/store"
)

var jobRunnerMap = map[string]func(force bool, debug bool, repoKey string) database.JSONB{
	"import":   cli.Import,
	"update":   cli.Update,
	"generate": cli.Generate,
}

func main() {
	database := database.Connect()
	jobReportStore := *store.NewJobReportStore(database)

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

	runner, ok := jobRunnerMap[job]
	if !ok {
		log.Print(job, " is not a valid job")
		os.Exit(1)
	}

	startTime := time.Now()

	fmt.Println()

	data := runner(force, debug, repoKey)

	elapsedTime := time.Since(startTime)

	err = jobReportStore.Create(context.TODO(), store.JobReport{
		Job:                  store.Job(job),
		ReportData:           data,
		ElapsedTimeInSeconds: int64(elapsedTime.Seconds()),
		CreatedAt:            time.Now(),
	})
	if err != nil {
		log.Panic(err)
	}

	fmt.Println()
	log.Printf("Elapsed time: %s\n", elapsedTime)
	log.Print(":wq")
}

func getJobArgs(osArgs []string) (string, bool, bool, string, error) {
	if len(osArgs) < 2 {
		return "", false, false, "", errors.New("Please provide an argument")
	}

	job := osArgs[1]

	if len(osArgs) < 3 {
		return job, false, false, "", nil
	}

	args := osArgs[2:]

	force := getArgIndex(args, "--force") != -1
	debug := getArgIndex(args, "--debug") != -1

	repoIndex := getArgIndex(args, "--repo")
	if repoIndex == -1 || len(args) < repoIndex+1 {
		return job, force, debug, "", nil
	}

	repoKey := strings.ToLower(args[repoIndex+1])
	return job, force, debug, repoKey, nil
}

func getArgIndex(args []string, target string) int {
	for index, arg := range args {
		if arg == target {
			return index
		}
	}
	return -1
}
