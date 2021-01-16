package main

import "testing"

func TestGetJobArg(t *testing.T) {
	t.Run("should return job and false force", func(t *testing.T) {
		osArgs := []string{"function", "import"}
		jobArg, force, repoKey, err := getJobArgs(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArgs; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "import")
		}

		if force == true {
			t.Errorf("Incorrect result for getJobArgs; got force: %v, want force: %v", force, false)
		}

		if repoKey != "" {
			t.Errorf("Incorrect result for getJobArgs; got repo key: %s, want empty repo key", repoKey)
		}
	})

	t.Run("should accept force option", func(t *testing.T) {
		osArgs := []string{"function", "import", "--force"}
		jobArg, force, repoKey, err := getJobArgs(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArgs; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "import")
		}

		if force == false {
			t.Errorf("Incorrect result for getJobArgs; got force: %v, want force: %v", force, true)
		}

		if repoKey != "" {
			t.Errorf("Incorrect result for getJobArgs; got repo key: %s, want empty repo key", repoKey)
		}
	})

	t.Run("should accept repo option", func(t *testing.T) {
		osArgs := []string{"function", "import", "--repo", "test/test"}
		jobArg, force, repoKey, err := getJobArgs(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArgs; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "import")
		}

		if force == true {
			t.Errorf("Incorrect result for getJobArgs; got force: %v, want force: %v", force, false)
		}

		if repoKey != "test/test" {
			t.Errorf("Incorrect result for getJobArgs; got repo key: %s, want repo key: %s", repoKey, "test/test")
		}
	})

	t.Run("should return error if second argument is missing", func(t *testing.T) {
		osArgs := []string{"function"}
		jobArg, _, _, err := getJobArgs(osArgs)

		if err == nil {
			t.Error("Incorrect result for getJobArgs; got no error")
		}

		if jobArg != "" {
			t.Errorf("Incorrect result for getJobArgs; got: %s, want: %s", jobArg, "")
		}
	})
}
