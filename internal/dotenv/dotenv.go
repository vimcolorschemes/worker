package dotenv

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	log.Print("Loading .env file")
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func Get(key string, doPanic bool) string {
	value, exists := os.LookupEnv(key)

	if doPanic && !exists {
		log.Panic(key, " not found in .env")
	}

	if !exists {
		log.Print(key, " not found in .env")
		return ""
	}

	return value
}

func GetInt(key string, doPanic bool, defaultValue int) int {
	value := Get(key, doPanic)

	if value == "" {
		return defaultValue
	}

	result, err := strconv.Atoi(value)

	if err != nil {
		if doPanic {
			log.Panic("Error parsing ", key, " to int with value ", value)
		} else {
			log.Print("Error parsing ", key, " to int with value ", value)
		}
	}

	return result
}
