package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/repository"

	gogithub "github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

var client *gogithub.Client

const fileQueryLimit = 50

const searchResultCountHardLimit = 1000

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

// GetRepository gets a repository from the GitHub API using a repository's owner and name
func GetRepository(ownerName string, name string) (*gogithub.Repository, error) {
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

// GetLastCommitAt gets the date of the last commit done in a repository's default branch
func GetLastCommitAt(repository *gogithub.Repository) time.Time {
	ownerName := *repository.Owner.Login
	name := *repository.Name
	defaultBranch := *repository.DefaultBranch
	options := &gogithub.CommitsListOptions{SHA: defaultBranch}

	commits, response, err := client.Repositories.ListCommits(context.Background(), ownerName, name, options)
	if _, ok := err.(*gogithub.RateLimitError); ok {
		log.Print("Hit rate limit reached")
		waitForRateLimitReset(response.Rate.Reset)
		return GetLastCommitAt(repository)
	} else if err != nil || len(commits) == 0 {
		log.Printf("Error getting last commit of %s/%s: %s", ownerName, name, err)
		return time.Time{}
	}

	return commits[0].Commit.Author.GetDate()
}

// GetRepositoryFileURLs returns all file URLs in a repository
func GetRepositoryFileURLs(repository *gogithub.Repository) []string {
	ownerName := *repository.Owner.Login
	name := *repository.Name
	basePath := ""

	fileURLs, err := getRepositoryFileUrlsAtPath(ownerName, name, basePath)
	if err != nil {
		log.Print(err)
		return []string{}
	}

	return fileURLs
}

func getRepositoryFileUrlsAtPath(ownerName string, name string, path string) ([]string, error) {
	options := &gogithub.RepositoryContentGetOptions{}
	_, contents, response, err := client.Repositories.GetContents(context.Background(), ownerName, name, path, options)
	if _, ok := err.(*gogithub.RateLimitError); ok {
		log.Print("Hit rate limit reached")
		waitForRateLimitReset(response.Rate.Reset)
		return getRepositoryFileUrlsAtPath(ownerName, name, path)
	} else if err != nil {
		log.Print(err)
		return []string{}, err
	}

	fileURLs := []string{}

	for _, content := range contents {
		if len(fileURLs) > fileQueryLimit {
			// limit reached
			return []string{}, errors.New("File limit reached")
		}

		switch content.GetType() {
		case "file":
			fileURLs = append(fileURLs, content.GetDownloadURL())
			break
		case "dir":
			newFileURLs, err := getRepositoryFileUrlsAtPath(ownerName, name, content.GetPath())
			if err != nil {
				return []string{}, err
			}
			fileURLs = append(fileURLs, newFileURLs...)
			break
		default:
			break
		}
	}

	return fileURLs, nil
}

// SearchRepositories returns all repositories from GitHub API matching some queries
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
	log.Printf("Sleep until rate limit reset at %s", resetTime)

	for {
		timeLeft := resetTime.Time.Sub(time.Now())
		log.Printf("Time left until reset: %s", timeLeft)

		time.Sleep(time.Second)

		if resetTime.Time.Before(time.Now()) {
			log.Print("Rate limit over, continuing...")
			break
		}
	}
}
