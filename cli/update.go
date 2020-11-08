package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/internal/database"
)

func Update() {
	log.Print("Run update")

	repositories := database.GetRepositories()

	for _, repository := range repositories {
		log.Print(repository.Owner.Name, "/", repository.Name)
	}

	log.Print(len(repositories), " repositories to update")

	log.Print(":wq")
}
