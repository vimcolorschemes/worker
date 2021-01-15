package main

import "testing"

func TestGetJobArg(t *testing.T) {
	t.Run("should return job and false force", func(t *testing.T) {
		osArgs := []string{"function", "import"}
		jobArg, force, err := getJobArg(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArg; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArg; got: %s, want: %s", jobArg, "import")
		}

		if force == true {
			t.Errorf("Incorrect result for getJobArg; got force: %v, want force: %v", force, false)
		}
	})

	t.Run("should return job and true force if option is present", func(t *testing.T) {
		osArgs := []string{"function", "import", "--force"}
		jobArg, force, err := getJobArg(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArg; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArg; got: %s, want: %s", jobArg, "import")
		}

		if force == false {
			t.Errorf("Incorrect result for getJobArg; got force: %v, want force: %v", force, true)
		}
	})

	t.Run("should return error if second argument is missing", func(t *testing.T) {
		osArgs := []string{"function"}
		jobArg, _, err := getJobArg(osArgs)

		if err == nil {
			t.Error("Incorrect result for getJobArg; got no error")
		}

		if jobArg != "" {
			t.Errorf("Incorrect result for getJobArg; got: %s, want: %s", jobArg, "")
		}
	})
}
