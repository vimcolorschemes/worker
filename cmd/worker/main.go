package main

import (
	"log"
	"os"

	"github.com/vimcolorschemes/worker/cli"
)

func main() {
	job := getArgJob()

	switch job {
	case "import":
		cli.Import()
	case "update":
		cli.Update()
	default:
		if job == "" {
			log.Print("Please Provide an argument")
			os.Exit(1)
		} else {
			log.Print(job, " is not a valid job")
			os.Exit(1)
		}
	}
}

func getArgJob() string {
	var job = ""

	if len(os.Args) > 1 {
		job = os.Args[1]
	}

	return job
}
