package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/repository"

	gogithub "github.com/google/go-github/v32/github"

	"golang.org/x/oauth2"
)

var client *gogithub.Client

func init() {
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: dotenv.Get("GITHUB_TOKEN", true)},
	)

	tc := oauth2.NewClient(ctx, ts)

	client = gogithub.NewClient(tc)
}

func GetRepository(ownerName string, name string) (*gogithub.Repository, error) {
	repository, _, err := client.Repositories.Get(context.Background(), ownerName, name)

	if err != nil {
		return nil, err
	}

	return repository, nil
}

func GetLastCommitAt(repository *gogithub.Repository) (time.Time, error) {
	ownerName := *repository.Owner.Login
	name := *repository.Name
	defaultBranch := *repository.DefaultBranch
	options := &gogithub.CommitsListOptions{SHA: defaultBranch}

	commits, _, err := client.Repositories.ListCommits(context.Background(), ownerName, name, options)

	if err != nil {
		return time.Now(), err
	}

	if len(commits) == 0 {
		return time.Now(), errors.New("no commits")
	}

	return commits[0].Commit.Author.GetDate(), nil
}

func SearchRepositories(queries []string, repositoryCountLimit int, repositoryCountLimitPerPage int) []*gogithub.Repository {
	log.Print("Search repositories")

	var repositories []*gogithub.Repository

	for _, query := range queries {
		query = fmt.Sprintf("%s %s", query, "NOT dotfiles stars:>0")

		log.Print("query: ", query)

		newRepositories := queryRepositories(query, repositoryCountLimit, repositoryCountLimitPerPage)
		log.Print("result count: ", len(newRepositories))

		repositories = append(repositories, newRepositories...)

		if len(repositories) >= repositoryCountLimit {
			break
		}
	}

	return repository.UniquifyRepositories(repositories)
}

func queryRepositories(query string, repositoryCountLimit int, repositoryCountLimitPerPage int) []*gogithub.Repository {
	page := 1
	totalCount := -1
	repositories := []*gogithub.Repository{}

	for len(repositories) != totalCount {
		log.Print("page: ", page)

		searchOptions := &gogithub.SearchOptions{Sort: "stars", ListOptions: gogithub.ListOptions{PerPage: repositoryCountLimitPerPage, Page: page}}
		result, _, err := client.Search.Repositories(context.Background(), query, searchOptions)

		if err != nil {
			panic(err)
		}

		if totalCount == -1 {
			totalCount = result.GetTotal()
			totalCount = int(math.Min(float64(totalCount), float64(repositoryCountLimit)))
			log.Print("total count: ", totalCount)
		}

		repositories = append(repositories, result.Repositories...)

		page++
	}

	return repositories
}
