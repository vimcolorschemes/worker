package dotenv

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	if strings.HasSuffix(os.Args[0], ".test") {
		// Running in test mode
		return
	}

	log.Print("Loading .env file")

	err := godotenv.Load()
	if err != nil {
		log.Print("No .env file found")
	}
}

// Get returns the string value of an environment variable
func Get(key string) (string, bool) {
	value, exists := os.LookupEnv(key)

	if !exists {
		log.Printf("%s not found in .env", key)
		return "", false
	}

	return value, true
}

// GetInt returns the int value of an environment variable
func GetInt(key string) (int, error) {
	value, exists := Get(key)

	if !exists {
		return 0, fmt.Errorf("%s does not exist", key)
	}

	result, err := strconv.Atoi(value)

	if err != nil {
		return 0, fmt.Errorf("Error parsing %s to int with value %s", key, value)
	}

	return result, nil
}
