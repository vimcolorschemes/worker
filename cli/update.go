package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/internal/database"
)

func Update() {
	log.Print("Run update")

	repositories := database.GetRepositories()

	log.Print(len(repositories), " repositories to update")

	log.Print(":wq")
}
