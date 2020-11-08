package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/github"
)

func Import() {
	log.Print("run import")

	repositories := github.SearchRepositories()

	log.Print("Upserting ", len(repositories), " repositories")

	database.UpsertRepositories(repositories)

	log.Print(":wq")
}
