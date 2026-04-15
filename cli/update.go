package cli

import (
	"fmt"
	"log"
	"time"

	gogithub "github.com/google/go-github/v68/github"
	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/github"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
)

var getGithubRepository = github.GetRepository
var isGithub404 = github.Is404

// Update the imported repositories with all kinds of useful information
func Update(_force bool, _debug bool, repoKey string) map[string]interface{} {
	var repositories []repoHelper.Repository
	if repoKey != "" {
		repository, err := database.GetRepository(repoKey)
		if err != nil {
			log.Panic(err)
		}
		repositories = []repoHelper.Repository{repository}
	} else {
		var err error
		repositories, err = database.GetRepositories()
		if err != nil {
			log.Panic(err)
		}
	}

	log.Print(len(repositories), " repositories to update")
	repositoryErrorCount := 0
	repositoryDeletedCount := 0
	repositoryDisabledCount := 0

	for index, repository := range repositories {
		fmt.Println()

		log.Print("Updating ", index, " of ", len(repositories), ": ", repository.Owner.Name, "/", repository.Name)

		updatedRepository, hadError, deleted := updateRepository(repository)
		if hadError {
			repositoryErrorCount++
		}
		if deleted {
			repositoryDeletedCount++
			continue
		}
		if updatedRepository.IsDisabled && !repository.IsDisabled {
			repositoryDisabledCount++
		}

		data := getUpdateData(updatedRepository)

		database.UpdateRepositoryFromUpdate(repository.ID, data)
	}

	return map[string]interface{}{
		"repositoryCount":         len(repositories),
		"repositoryErrorCount":    repositoryErrorCount,
		"repositoryDeletedCount":  repositoryDeletedCount,
		"repositoryDisabledCount": repositoryDisabledCount,
	}
}

func updateRepository(repository repoHelper.Repository) (repoHelper.Repository, bool, bool) {
	githubRepository, err := getGithubRepository(repository.Owner.Name, repository.Name)
	if err != nil {
		log.Printf("Error fetching %s/%s: %v", repository.Owner.Name, repository.Name, err)
		if isGithub404(err) {
			// Repo was deleted, renamed, or made private — prune it so we
			// stop trying every day. Cascade clears colorschemes and events.
			if delErr := database.DeleteRepository(repository.ID); delErr != nil {
				log.Printf("Error deleting repository %s/%s: %v", repository.Owner.Name, repository.Name, delErr)
				repository.IsEligible = false
				return repository, true, false
			}
			log.Printf("Deleted repository %s/%s (404 from Github)", repository.Owner.Name, repository.Name)
			return repository, true, true
		}
		repository.IsEligible = false
		return repository, true, false
	}

	if githubRepository.PushedAt == nil {
		log.Print("No commits on ", repository.Owner.Name, "/", repository.Name)
		repository.IsEligible = false
		repository.IsDisabled = true
		log.Print("Automatically disabled ", repository.Owner.Name, "/", repository.Name, " because it has no commits")
		return repository, false, false
	}

	repository.PushedAt = githubRepository.PushedAt.Time

	log.Print("Gathering basic infos")
	repository.StargazersCount = *githubRepository.StargazersCount

	log.Print("Building stargazers count history")
	repository.StargazersCountHistory = repository.AppendToStargazersCountHistory()

	log.Print("Computing week stargazers count")
	repository.WeekStargazersCount = repository.ComputeTrendingStargazersCount(7)

	log.Print("Checking if ", repository.Owner.Name, "/", repository.Name, " is eligible")
	repository.IsEligible = repository.IsEligibleAfterUpdate()
	// A successful update means the repo is active — clear any prior disable flag
	repository.IsDisabled = false
	log.Printf("Eligible after update: %v", repository.IsEligible)

	return repository, false, false
}

func getUpdateData(repository repoHelper.Repository) database.UpdateData {
	return database.UpdateData{
		PushedAt:               repository.PushedAt,
		StargazersCount:        repository.StargazersCount,
		StargazersCountHistory: repository.StargazersCountHistory,
		WeekStargazersCount:    repository.WeekStargazersCount,
		IsEligible:             repository.IsEligible,
		IsDisabled:             repository.IsDisabled,
		UpdatedAt:              time.Now(),
	}
}

var _ func(string, string) (*gogithub.Repository, error) = getGithubRepository
