package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/github"
)

func Import() {
	log.Print("run import")

	repositories := github.SearchRepositories()

	log.Print(len(repositories))
}
