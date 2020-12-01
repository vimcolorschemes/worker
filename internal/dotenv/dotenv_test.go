package dotenv

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	if !strings.HasSuffix(os.Args[0], ".test") {
		// Not running in test mode
		return
	}

	err := godotenv.Load("./.env.test")
	if err != nil {
		log.Printf("Error setting up test dotenv variables: %s", err)
	}
}

func TestGet(t *testing.T) {
	t.Run("should return string value if it exists", func(t *testing.T) {
		key := "WORKER_TEST_VALUE"
		expectedValue := "value"

		value, exists := Get(key)

		if !exists {
			t.Errorf("Incorrect result for Get, %s does not exist", key)
		}

		if value != expectedValue {
			t.Errorf("Incorrect result for Get, got: %s, want, %s", value, expectedValue)
		}
	})

	t.Run("should return error if value does not exist", func(t *testing.T) {
		key := "WORKER_TEST_NOT_EXIST"

		value, exists := Get(key)

		if exists {
			t.Errorf("Incorrect result for Get, %s exists: %s", key, value)
		}

		if value != "" {
			t.Errorf("Incorrect result for Get, got: %s, want, %s", value, "")
		}
	})
}

func TestGetInt(t *testing.T) {
	t.Run("should return int value if it exists", func(t *testing.T) {
		value, err := GetInt("WORKER_TEST_INT_1")

		if err != nil {
			t.Errorf("Incorrect result for GetInt, got error: %s", err)
		}

		if value != 1 {
			t.Errorf("Incorrect result for GetInt, got: %d, want, %d", value, 1)
		}
	})

	t.Run("should return error if value does not exist", func(t *testing.T) {
		value, err := GetInt("WORKER_TEST_NOT_EXIST")

		if err == nil {
			t.Error("Incorrect result for GetInt, got no error")
		}

		if value != 0 {
			t.Errorf("Incorrect result for GetInt, got: %d, want, %d", value, 0)
		}
	})

	t.Run("should return error if value is not parsable as an int", func(t *testing.T) {
		value, err := GetInt("WORKER_TEST_NOT_INT")

		if err == nil {
			t.Error("Incorrect result for GetInt, got no error")
		}

		if value != 0 {
			t.Errorf("Incorrect result for GetInt, got: %d, want, %d", value, 0)
		}
	})
}
