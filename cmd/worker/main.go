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
	job, force, err := getJobArg(os.Args)

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	runner := jobRunnerMap[job]
	if runner == nil {
		log.Print(job, " is not a valid job")
		os.Exit(1)
	}

	runner.(func(force bool))(force)
}

func getJobArg(osArgs []string) (string, bool, error) {
	if len(osArgs) < 2 {
		return "", false, errors.New("Please provide an argument")
	}

	if len(osArgs) < 3 {
		return osArgs[1], false, nil
	}

	return osArgs[1], osArgs[2] == "--force", nil
}
