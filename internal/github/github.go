package github

import (
	"context"
	"fmt"
	"log"

	gogithub "github.com/google/go-github/v32/github"

	"github.com/vimcolorschemes/worker/internal/dotenv"

	"golang.org/x/oauth2"
)

var client *gogithub.Client

func init() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: dotenv.Get("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client = gogithub.NewClient(tc)
}

func SearchRepositories() []*gogithub.Repository {
	log.Print("Search repositories")

	queries := []string{
		"vim theme",
		"vim color scheme",
		"vim colorscheme",
		"vim colour scheme",
		"vim colourscheme",
		"neovim theme",
		"neovim color scheme",
		"neovim colorscheme",
		"neovim colour scheme",
		"neovim colourscheme",
	}

	var repositories []*gogithub.Repository

	for _, query := range queries {
		log.Print("query: ", query)

		result, _, err := client.Search.Repositories(context.Background(), query, nil)
		if err != nil {
			panic(err)
		}

		newRepositories := result.Repositories
		log.Print("result count: ", len(newRepositories))

		repositories = append(repositories, newRepositories...)
	}

	return uniquifyRepositories(repositories)
}

func uniquifyRepositories(repositories []*gogithub.Repository) []*gogithub.Repository {
	keys := make(map[string]bool)
	unique := []*gogithub.Repository{}

	for _, repository := range repositories {
		repositoryKey := fmt.Sprintf("%d/%d", repository.Owner.Name, repository.Name)
		if _, value := keys[repositoryKey]; !value {
			keys[repositoryKey] = true
			unique = append(unique, repository)
		}
	}

	return unique
}
