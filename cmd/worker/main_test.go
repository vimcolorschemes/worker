package main

import "testing"

func TestGetJobArg(t *testing.T) {
	t.Run("should return second argument", func(t *testing.T) {
		osArgs := []string{"function", "import"}
		jobArg, err := getJobArg(osArgs)
		if err != nil {
			t.Errorf("Incorrect result for getJobArg; got error: %s", err)
		}

		if jobArg != "import" {
			t.Errorf("Incorrect result for getJobArg; got: %s, want: %s", jobArg, "import")
		}
	})

	t.Run("should return error if second argument is missing", func(t *testing.T) {
		osArgs := []string{"function"}
		jobArg, err := getJobArg(osArgs)

		if err == nil {
			t.Error("Incorrect result for getJobArg; got no error")
		}

		if jobArg != "" {
			t.Errorf("Incorrect result for getJobArg; got: %s, want: %s", jobArg, "")
		}
	})
}
