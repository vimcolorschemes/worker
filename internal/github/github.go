package github

import (
	"context"
	"fmt"
	"log"
	"math"

	gogithub "github.com/google/go-github/v32/github"

	"github.com/vimcolorschemes/worker/internal/dotenv"

	"golang.org/x/oauth2"
)

var client *gogithub.Client

// TODO use const
var searchOptions = &gogithub.SearchOptions{Sort: "stars", ListOptions: gogithub.ListOptions{PerPage: 100, Page: 1}}

// TODO use const
var queries = []string{
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

	var repositories []*gogithub.Repository

	for _, query := range queries {
		query = fmt.Sprintf("%s %s", query, "NOT dotfiles stars:>0")

		log.Print("query: ", query)

		newRepositories := queryRepositories(query)
		log.Print("result count: ", len(newRepositories))

		repositories = append(repositories, newRepositories...)
	}

	return uniquifyRepositories(repositories)
}

func queryRepositories(query string) []*gogithub.Repository {
	log.Print("page: ", 1)
	result, _, err := client.Search.Repositories(context.Background(), query, &gogithub.SearchOptions{Sort: "stars", ListOptions: gogithub.ListOptions{PerPage: 100, Page: 1}})

	if err != nil {
		panic(err)
	}

	repositories := result.Repositories
	log.Print("result count: ", len(repositories))

	totalCount := result.GetTotal()
	log.Print("total count: ", totalCount)

	pageCount := int(math.Ceil(float64(totalCount) / 100))

	log.Print("page count: ", pageCount)

	if pageCount > 10 {
		pageCount = 10
		log.Print("page count limited to: ", pageCount)
	}

	if pageCount == 1 {
		return repositories
	}

	for page := 2; page <= pageCount; page++ {
		log.Print("page: ", page)
		result, _, err = client.Search.Repositories(context.Background(), query, &gogithub.SearchOptions{Sort: "stars", ListOptions: gogithub.ListOptions{PerPage: 100, Page: page}})

		if err != nil {
			panic(err)
		}

		newRepositories := result.Repositories

		log.Print("result count: ", len(newRepositories))

		repositories = append(repositories, newRepositories...)
	}

	return repositories
}

func uniquifyRepositories(repositories []*gogithub.Repository) []*gogithub.Repository {
	keys := make(map[int64]bool)
	unique := []*gogithub.Repository{}

	for _, repository := range repositories {
		if _, value := keys[*repository.ID]; !value {
			keys[*repository.ID] = true
			unique = append(unique, repository)
		}
	}

	return unique
}
