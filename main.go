package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/vimcolorschemes/worker/job"
	"github.com/vimcolorschemes/worker/runner"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	var jobToRun = job.GetJob(os.Args)

	switch jobToRun {
	case job.Import:
		import_runner.Run()
	default:
		fmt.Println("Error running", jobToRun)
	}
}
