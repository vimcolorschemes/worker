package dotenv

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	log.Print("Loading .env file")
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func Get(key string) string {
	value, exists := os.LookupEnv(key)

	if !exists {
		log.Panic(key, " not found in .env")
	}

	return value
}
