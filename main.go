package main

import (
	"fmt"
	"github.com/vimcolorschemes/worker/job"
	"os"
)

func main() {
	var job = job.GetJob(os.Args)
	fmt.Println(job)
}
