package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/database"
)

func Update() {
	log.Print("Running update")

	repositories := database.GetRepositories()

	log.Println(repositories)
}
