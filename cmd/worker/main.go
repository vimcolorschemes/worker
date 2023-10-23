package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/vimcolorschemes/worker/cli"
	"github.com/vimcolorschemes/worker/internal/database"
)

var jobRunnerMap = map[string]interface{}{
	"import":   cli.Import,
	"update":   cli.Update,
	"generate": cli.Generate,
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

	data := runner.(func(force bool, debug bool, repoKey string) bson.M)(force, debug, repoKey)

	elapsedTime := time.Since(startTime)
	database.CreateReport(job, elapsedTime.Seconds(), data)

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
