package main

import (
	"errors"
	"log"
	"os"

	"github.com/vimcolorschemes/worker/cli"
)

func main() {
	job, err := getArgJob()

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	switch job {
	case "import":
		cli.Import()
	case "update":
		cli.Update()
	default:
		log.Print(job, " is not a valid job")
		os.Exit(1)
	}
}

func getArgJob() (string, error) {
	if len(os.Args) > 1 {
		return os.Args[1], nil
	}

	return "", errors.New("Please provide an argument")
}
