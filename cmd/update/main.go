package main

import (
	"log"

	"github.com/vimcolorschemes/worker/internal/pkg/database"
)

func main() {
	repositories := database.GetRepositories()

	log.Println(repositories)
}
