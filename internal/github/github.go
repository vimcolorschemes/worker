package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/repository"

	gogithub "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

var client *gogithub.Client

const searchResultCountHardLimit = 1000

var colorschemeFilePattern = regexp.MustCompile(`^colors/[^/]+\.(vim|lua)$`)

// ErrTreeTruncated is returned when Github truncated the tree response before
// any colorscheme file showed up, so the count cannot be trusted.
var ErrTreeTruncated = errors.New("github tree response truncated")

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

// isEmptyRepository reports whether err is the Github API 409 for a repository
// with no commits, which has no tree to count.
func isEmptyRepository(err error) bool {
	var errResp *gogithub.ErrorResponse
	if !errors.As(err, &errResp) || errResp.Response == nil {
		return false
	}
	if errResp.Response.StatusCode != http.StatusConflict {
		return false
	}
	return strings.Contains(strings.ToLower(errResp.Message), "empty")
}

// CountColorschemeFiles reports how many colors/*.{vim,lua} files a repository
// ships, by matching its recursive HEAD tree.
func CountColorschemeFiles(ownerName string, name string) (int, error) {
	if strings.HasSuffix(os.Args[0], ".test") {
		return 0, errors.New("running in test mode")
	}

	tree, response, err := client.Git.GetTree(context.Background(), ownerName, name, "HEAD", true)

	if _, ok := err.(*gogithub.RateLimitError); ok {
		log.Print("Hit rate limit reached")
		waitForRateLimitReset(response.Rate.Reset)
		return CountColorschemeFiles(ownerName, name)
	} else if abuseErr, ok := err.(*gogithub.AbuseRateLimitError); ok {
		log.Print("Hit secondary rate limit")
		waitForSecondaryRateLimit(abuseErr, response)
		return CountColorschemeFiles(ownerName, name)
	} else if Is404(err) || isEmptyRepository(err) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	count := countColorschemeFiles(tree)
	if count == 0 && tree.GetTruncated() {
		return 0, ErrTreeTruncated
	}

	return count, nil
}

func countColorschemeFiles(tree *gogithub.Tree) int {
	count := 0
	for _, entry := range tree.Entries {
		if colorschemeFilePattern.MatchString(entry.GetPath()) {
			count++
		}
	}
	return count
}

// SearchRepositories returns all repositories from Github API matching some queries
func SearchRepositories(queries []string, repositoryCountLimit int, repositoryCountLimitPerPage int) []*gogithub.Repository {
	if strings.HasSuffix(os.Args[0], ".test") {
		return []*gogithub.Repository{}
	}

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
	if strings.HasSuffix(os.Args[0], ".test") {
		return []*gogithub.Repository{}
	}

	page := 1
	totalCount := -1
	repositories := []*gogithub.Repository{}

	for len(repositories) != totalCount && page*repositoryCountLimitPerPage <= searchResultCountHardLimit {
		log.Print("page: ", page)
		log.Print("repository count: ", len(repositories))

		searchOptions := &gogithub.SearchOptions{Sort: "stars", ListOptions: gogithub.ListOptions{PerPage: repositoryCountLimitPerPage, Page: page}}
		result, response, err := client.Search.Repositories(context.Background(), query, searchOptions)
		if _, ok := err.(*gogithub.RateLimitError); ok {
			log.Print("Hit rate limit reached")
			waitForRateLimitReset(response.Rate.Reset)
			return queryRepositories(query, repositoryCountLimit, repositoryCountLimitPerPage)
		} else if err != nil {
			log.Panic(err)
		}

		if totalCount == -1 {
			totalCount = result.GetTotal()
			totalCount = int(math.Min(float64(totalCount), float64(repositoryCountLimit)))
			log.Printf("total count: %d", totalCount)
		}

		repositories = append(repositories, result.Repositories...)

		page++
	}

	return repositories
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

// secondaryRateLimitFallbackWait floors the wait when Github sends neither
// Retry-After nor a reset still in the future, so the retry does not fire again
// within the second for the whole limit window.
const secondaryRateLimitFallbackWait = time.Minute

// waitForSecondaryRateLimit sleeps out a secondary rate limit, which Github
// answers with Retry-After rather than the rate limit reset timestamp.
func waitForSecondaryRateLimit(err *gogithub.AbuseRateLimitError, response *gogithub.Response) {
	if err.RetryAfter != nil {
		waitForRateLimitReset(gogithub.Timestamp{Time: time.Now().Add(*err.RetryAfter)})
		return
	}

	if response != nil && response.Rate.Reset.After(time.Now()) {
		waitForRateLimitReset(response.Rate.Reset)
		return
	}

	waitForRateLimitReset(gogithub.Timestamp{Time: time.Now().Add(secondaryRateLimitFallbackWait)})
}
