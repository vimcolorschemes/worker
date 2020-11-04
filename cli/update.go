package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/database"
)

func Update() {
	log.Print("Run update")

	repositories := database.GetRepositories()

	log.Println(repositories)
}
