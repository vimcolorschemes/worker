package github

import (
	"context"
	"log"

	gogithub "github.com/google/go-github/v32/github"

	"github.com/vimcolorschemes/worker/dotenv"

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

	var uniqueRepositories []*gogithub.Repository
	m := map[*int64]bool{}

	for _, repository := range repositories {
		if !m[repository.ID] {
			m[repository.ID] = true
			log.Print(repository.ID)
			log.Print(repository.Owner.Name)
			log.Print(repository.Name)
			uniqueRepositories = append(uniqueRepositories, repository)
		}
	}

	return uniqueRepositories
}
