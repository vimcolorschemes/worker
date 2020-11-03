package job_test

import (
	"github.com/vimcolorschemes/worker/job"
	"testing"
)

func TestGetJobNoArgs(t *testing.T) {
	// No args -> job.Import
	args := []string{}
	checkJob(job.GetJob(args), job.Import, t)
}

func TestGetJobTypo(t *testing.T) {
	// typo -> job.Import
	args := []string{"", "updat"}
	checkJob(job.GetJob(args), job.Import, t)
}

func TestGetJobImport(t *testing.T) {
	// import -> job.Import
	args := []string{"", string(job.Import)}
	checkJob(job.GetJob(args), job.Import, t)
}

func TestGetJobUpdate(t *testing.T) {
	// update -> job.Update
	args := []string{"", string(job.Update)}
	checkJob(job.GetJob(args), job.Update, t)
}

func TestGetJobClean(t *testing.T) {
	// clean -> job.Clean
	args := []string{"", string(job.Clean)}
	checkJob(job.GetJob(args), job.Clean, t)
}

func checkJob(got job.JobType, expected job.JobType, t *testing.T) {
	if got != expected {
		t.Errorf("job.GetJob([]job.JobType{\"\", %s}) = %s; want %s", expected, expected, got)
	}
}
