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

var repositoryCountLimit int
var repositoryCountLimitPerPage int

const searchResultCountHardLimit = 1000

var queryPageCountLimit int = searchResultCountHardLimit

var queries = [10]string{
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
	initGitHubClient()

	repositoryCountLimit = dotenv.GetInt("GITHUB_REPOSITORY_COUNT_LIMIT", false, 100)
	log.Print("Fetching a soft limit of ", repositoryCountLimit, " repositories")
	repositoryCountLimitPerPage = int(math.Min(float64(repositoryCountLimit), 100))
	queryPageCountLimit = getPageCount(searchResultCountHardLimit, repositoryCountLimitPerPage)
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

		if len(repositories) >= repositoryCountLimit {
			break
		}
	}

	return uniquifyRepositories(repositories)
}

func initGitHubClient() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: dotenv.Get("GITHUB_TOKEN", true)},
	)
	tc := oauth2.NewClient(ctx, ts)
	client = gogithub.NewClient(tc)
}

func queryRepositories(query string) []*gogithub.Repository {
	log.Print("page: ", 1)

	result, _, err := client.Search.Repositories(context.Background(), query, buildSearchOptions(1))

	if err != nil {
		panic(err)
	}

	repositories := result.Repositories
	log.Print("result count: ", len(repositories))

	totalCount := result.GetTotal()
	totalCount = int(math.Min(float64(totalCount), float64(repositoryCountLimit)))
	log.Print("total count: ", totalCount)

	pageCount := getPageCount(totalCount, repositoryCountLimitPerPage)
	log.Print("page count: ", pageCount)

	if pageCount == 1 || len(repositories) >= repositoryCountLimit {
		return repositories
	}

	return append(repositories, queryPaginatedRepositories(pageCount, query)...)
}

func queryPaginatedRepositories(pageCount int, query string) []*gogithub.Repository {
	var repositories []*gogithub.Repository

	for page := 2; page <= pageCount; page++ {
		log.Print("page: ", page)
		result, _, err := client.Search.Repositories(context.Background(), query, buildSearchOptions(page))

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

func getPageCount(totalCount int, itemPerPageCount int) int {
	pageCount := int(math.Ceil(float64(totalCount / itemPerPageCount)))

	if pageCount > queryPageCountLimit {
		pageCount = queryPageCountLimit
	}

	return pageCount
}

func buildSearchOptions(page int) *gogithub.SearchOptions {
	return &gogithub.SearchOptions{Sort: "stars", ListOptions: gogithub.ListOptions{PerPage: repositoryCountLimitPerPage, Page: page}}
}
