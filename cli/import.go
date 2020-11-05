package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/internal/github"
)

func Import() {
	log.Print("run import")

	repositories := github.SearchRepositories()

	log.Print(len(repositories))
}
