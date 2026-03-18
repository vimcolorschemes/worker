package cli

import (
	"fmt"
	"log"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/github"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
)

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

	for index, repository := range repositories {
		fmt.Println()

		log.Print("Updating ", index, " of ", len(repositories), ": ", repository.Owner.Name, "/", repository.Name)

		updatedRepository := updateRepository(repository)

		data := getUpdateData(updatedRepository)

		database.UpdateRepositoryFromUpdate(repository.ID, data)
	}

	return map[string]interface{}{"repositoryCount": len(repositories)}
}

func updateRepository(repository repoHelper.Repository) repoHelper.Repository {
	githubRepository, err := github.GetRepository(repository.Owner.Name, repository.Name)
	if err != nil {
		log.Print("Error fetching ", repository.Owner.Name, "/", repository.Name)
		repository.IsEligible = false
		return repository
	}

	if githubRepository.PushedAt == nil {
		log.Print("No commits on ", repository.Owner.Name, "/", repository.Name)
		repository.IsEligible = false
		return repository
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
	log.Printf("Eligible after update: %v", repository.IsEligible)

	return repository
}

func getUpdateData(repository repoHelper.Repository) database.UpdateData {
	return database.UpdateData{
		PushedAt:               repository.PushedAt,
		StargazersCount:        repository.StargazersCount,
		StargazersCountHistory: repository.StargazersCountHistory,
		WeekStargazersCount:    repository.WeekStargazersCount,
		IsEligible:             repository.IsEligible,
		UpdatedAt:              time.Now(),
	}
}
