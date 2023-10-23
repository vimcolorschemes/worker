package main

import "testing"

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
