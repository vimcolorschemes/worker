package main

import (
	"errors"
	"log"
	"os"

	"github.com/vimcolorschemes/worker/cli"
)

var jobRunnerMap = map[string]interface{}{
	"import":   cli.Import,
	"update":   cli.Update,
	"generate": cli.Generate,
}

func main() {
	job, err := getJobArg(os.Args)

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	runner := jobRunnerMap[job]
	if runner == nil {
		log.Print(job, " is not a valid job")
		os.Exit(1)
	}

	runner.(func())()
}

func getJobArg(osArgs []string) (string, error) {
	if len(osArgs) > 1 {
		return osArgs[1], nil
	}

	return "", errors.New("Please provide an argument")
}
