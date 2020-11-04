package cli

import (
	"context"
	"github.com/google/go-github/v32/github"
	"log"
)

func Import() {
	log.Print("Running import")

	client := github.NewClient(nil)

	repositories := searchRepositories(client)

	log.Print(len(repositories))
}

func searchRepositories(client *github.Client) []*github.Repository {
	result, _, err := client.Search.Repositories(context.Background(), "vim color scheme", nil)

	if err != nil {
		panic(err)
	}

	return result.Repositories
}
