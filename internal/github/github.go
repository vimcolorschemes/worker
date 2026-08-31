package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/repository"

	gogithub "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

var client *gogithub.Client

const searchResultCountHardLimit = 1000

const rateLimitRetryLimit = 5

func init() {
	if strings.HasSuffix(os.Args[0], ".test") {
		// Running in test mode
		return
	}

	ctx := context.Background()

	var ts oauth2.TokenSource
	gitHubToken, exists := dotenv.Get("GITHUB_TOKEN")
	if exists {
		ts = oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: gitHubToken},
		)
	}
	tc := oauth2.NewClient(ctx, ts)

	client = gogithub.NewClient(tc)
}

// Is404 reports whether err is a Github API "not found" response. This is the
// signal we use to prune repositories that no longer exist (deleted, renamed,
// or made private) so the update job stops wasting work on them.
func Is404(err error) bool {
	var errResp *gogithub.ErrorResponse
	if !errors.As(err, &errResp) || errResp.Response == nil {
		return false
	}
	return errResp.Response.StatusCode == http.StatusNotFound
}

// GetRepository gets a repository from the Github API using a repository's owner and name
func GetRepository(ownerName string, name string) (*gogithub.Repository, error) {
	if strings.HasSuffix(os.Args[0], ".test") {
		return nil, errors.New("running in test mode")
	}

	repository, response, err := client.Repositories.Get(context.Background(), ownerName, name)

	if _, ok := err.(*gogithub.RateLimitError); ok {
		log.Print("Hit rate limit reached")
		waitForRateLimitReset(response.Rate.Reset)
		return GetRepository(ownerName, name)
	} else if err != nil {
		return nil, err
	}

	return repository, nil
}

// SearchRepositories returns all repositories from Github API matching some queries.
// repositoryCountLimit applies to each query, not to the combined result.
func SearchRepositories(queries []string, repositoryCountLimit int, repositoryCountLimitPerPage int) []*gogithub.Repository {
	if strings.HasSuffix(os.Args[0], ".test") {
		return []*gogithub.Repository{}
	}

	log.Print("Search repositories")

	var repositories []*gogithub.Repository
	failedQueryCount := 0

	for _, query := range queries {
		query = fmt.Sprintf("%s %s", query, "NOT dotfiles stars:>0")

		log.Print("query: ", query)

		newRepositories, err := queryRepositories(query, repositoryCountLimit, repositoryCountLimitPerPage)
		if err != nil {
			log.Print("query failed: ", err)
			failedQueryCount++
		}
		log.Print("result count: ", len(newRepositories))

		repositories = append(repositories, newRepositories...)
	}

	if failedQueryCount > 0 && failedQueryCount == len(queries) {
		log.Panicf("all %d search queries failed", failedQueryCount)
	}

	return repository.UniquifyRepositories(repositories)
}

func queryRepositories(query string, repositoryCountLimit int, repositoryCountLimitPerPage int) ([]*gogithub.Repository, error) {
	if strings.HasSuffix(os.Args[0], ".test") {
		return []*gogithub.Repository{}, nil
	}

	page := 1
	rateLimitRetryCount := 0
	totalCount := repositoryCountLimit
	repositories := []*gogithub.Repository{}

	for len(repositories) < totalCount && page*repositoryCountLimitPerPage <= searchResultCountHardLimit {
		log.Print("page: ", page)
		log.Print("repository count: ", len(repositories))

		searchOptions := &gogithub.SearchOptions{Sort: "stars", ListOptions: gogithub.ListOptions{PerPage: repositoryCountLimitPerPage, Page: page}}
		result, response, err := client.Search.Repositories(context.Background(), query, searchOptions)
		if _, ok := err.(*gogithub.RateLimitError); ok {
			// An empty reset timestamp makes the wait return immediately.
			if rateLimitRetryCount >= rateLimitRetryLimit {
				return repositories, fmt.Errorf("rate limit still not clear after %d retries", rateLimitRetryLimit)
			}
			rateLimitRetryCount++

			log.Print("Hit rate limit reached")
			waitForRateLimitReset(response.Rate.Reset)
			continue
		} else if err != nil {
			return repositories, err
		}

		rateLimitRetryCount = 0

		if page == 1 {
			totalCount = int(math.Min(float64(result.GetTotal()), float64(repositoryCountLimit)))
			log.Printf("total count: %d", totalCount)
		}

		if len(result.Repositories) == 0 {
			break
		}

		repositories = append(repositories, result.Repositories...)

		page++
	}

	return repositories, nil
}

func waitForRateLimitReset(resetTime gogithub.Timestamp) {
	if strings.HasSuffix(os.Args[0], ".test") {
		return
	}

	log.Printf("Sleep until rate limit reset at %s", resetTime)

	for {
		timeLeft := time.Until(resetTime.Time)
		log.Printf("Time left until reset: %s", timeLeft)

		time.Sleep(time.Second)

		if resetTime.Before(time.Now()) {
			log.Print("Rate limit over, continuing...")
			break
		}
	}
}
