package main

import (
	"strings"
	"testing"
)

func TestGetJobArg(t *testing.T) {
	t.Run("should return job and false force", func(t *testing.T) {
		osArgs := []string{"function", "import"}
		jobArg, force, debug, repoKey, err := getJobArgs(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArgs; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "import")
		}

		if force == true {
			t.Errorf("Incorrect result for getJobArgs; got force: %v, want force: %v", force, false)
		}

		if debug == true {
			t.Errorf("Incorrect result for getJobArgs; got debug: %v, want debug: %v", debug, false)
		}

		if repoKey != "" {
			t.Errorf("Incorrect result for getJobArgs; got repo key: %s, want empty repo key", repoKey)
		}
	})

	t.Run("should accept force option", func(t *testing.T) {
		osArgs := []string{"function", "import", "--force"}
		jobArg, force, debug, repoKey, err := getJobArgs(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArgs; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "import")
		}

		if force == false {
			t.Errorf("Incorrect result for getJobArgs; got force: %v, want force: %v", force, true)
		}

		if debug == true {
			t.Errorf("Incorrect result for getJobArgs; got debug: %v, want debug: %v", debug, false)
		}

		if repoKey != "" {
			t.Errorf("Incorrect result for getJobArgs; got repo key: %s, want empty repo key", repoKey)
		}
	})

	t.Run("should accept debug option", func(t *testing.T) {
		osArgs := []string{"function", "import", "--debug"}
		jobArg, force, debug, repoKey, err := getJobArgs(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArgs; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "import")
		}

		if force == true {
			t.Errorf("Incorrect result for getJobArgs; got force: %v, want force: %v", force, false)
		}

		if debug == false {
			t.Errorf("Incorrect result for getJobArgs; got debug: %v, want debug: %v", debug, true)
		}

		if repoKey != "" {
			t.Errorf("Incorrect result for getJobArgs; got repo key: %s, want empty repo key", repoKey)
		}
	})

	t.Run("should accept repo option", func(t *testing.T) {
		osArgs := []string{"function", "import", "--repo", "test/test"}
		jobArg, force, debug, repoKey, err := getJobArgs(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArgs; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "import")
		}

		if force == true {
			t.Errorf("Incorrect result for getJobArgs; got force: %v, want force: %v", force, false)
		}

		if debug == true {
			t.Errorf("Incorrect result for getJobArgs; got debug: %v, want debug: %v", debug, false)
		}

		if repoKey != "test/test" {
			t.Errorf("Incorrect result for getJobArgs; got repo key: %s, want repo key: %s", repoKey, "test/test")
		}
	})

	t.Run("should return error if second argument is missing", func(t *testing.T) {
		osArgs := []string{"function"}
		jobArg, _, _, _, err := getJobArgs(osArgs)

		if err == nil {
			t.Error("Incorrect result for getJobArgs; got no error")
		}

		if jobArg != "" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "")
		}
	})
}

func TestRunJobWithRecovery(t *testing.T) {
	t.Run("returns runner data on success", func(t *testing.T) {
		runner := func(_force bool, _debug bool, _repoKey string) map[string]interface{} {
			return map[string]interface{}{"repositoryCount": 2}
		}

		data, err, stackTrace := runJobWithRecovery(runner, false, false, "")

		if err != nil {
			t.Fatalf("runJobWithRecovery error = %v, want nil", err)
		}

		if stackTrace != "" {
			t.Fatalf("stackTrace = %q, want empty", stackTrace)
		}

		if data["repositoryCount"] != 2 {
			t.Fatalf("repositoryCount = %v, want 2", data["repositoryCount"])
		}
	})

	t.Run("captures panic and returns stack trace", func(t *testing.T) {
		runner := func(_force bool, _debug bool, _repoKey string) map[string]interface{} {
			panic("boom")
		}

		data, err, stackTrace := runJobWithRecovery(runner, false, false, "")

		if err == nil {
			t.Fatal("runJobWithRecovery error = nil, want error")
		}

		if !strings.Contains(err.Error(), "panic: boom") {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), "panic: boom")
		}

		if stackTrace == "" {
			t.Fatal("stackTrace = empty, want stack trace")
		}

		if data != nil {
			t.Fatalf("data = %v, want nil", data)
		}
	})
}
