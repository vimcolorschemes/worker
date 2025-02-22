package cli

import (
	"fmt"
	"log"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/github"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
)

// Update the imported repositories with all kinds of useful information
func Update(_force bool, _debug bool, repoKey string) bson.M {
	var repositories []repoHelper.Repository
	if repoKey != "" {
		repository, err := database.GetRepository(repoKey)
		if err != nil {
			log.Panic(err)
		}
		repositories = []repoHelper.Repository{repository}
	} else {
		repositories = database.GetRepositories()
	}

	log.Print(len(repositories), " repositories to update")

	for index, repository := range repositories {
		fmt.Println()

		log.Print("Updating ", index, " of ", len(repositories), ": ", repository.Owner.Name, "/", repository.Name)

		updatedRepository := updateRepository(repository)

		updateObject := getUpdateRepositoryObject(updatedRepository)

		database.UpsertRepository(repository.ID, updateObject)
	}

	return bson.M{"repositoryCount": len(repositories)}
}

func updateRepository(repository repoHelper.Repository) repoHelper.Repository {
	githubRepository, err := github.GetRepository(repository.Owner.Name, repository.Name)
	if err != nil {
		log.Print("Error fetching ", repository.Owner.Name, "/", repository.Name)
		repository.UpdateValid = false
		return repository
	}

	repository.GithubUpdatedAt = githubRepository.UpdatedAt.Time

	log.Print("Gathering basic infos")
	repository.StargazersCount = *githubRepository.StargazersCount

	log.Print("Building stargazers count history")
	repository.StargazersCountHistory = repository.AppendToStargazersCountHistory()

	log.Print("Computing week stargazers count")
	repository.WeekStargazersCount = repository.ComputeTrendingStargazersCount(7)

	log.Print("Checking if ", repository.Owner.Name, "/", repository.Name, " is valid")
	repository.UpdateValid = repository.IsValidAfterUpdate()
	log.Printf("Update valid: %v", repository.UpdateValid)

	return repository
}

func getUpdateRepositoryObject(repository repoHelper.Repository) bson.M {
	return bson.M{
		"githubUpdatedAt":        repository.GithubUpdatedAt,
		"stargazersCount":        repository.StargazersCount,
		"stargazersCountHistory": repository.StargazersCountHistory,
		"weekStargazersCount":    repository.WeekStargazersCount,
		"updateValid":            repository.UpdateValid,
		"updatedAt":              time.Now(),
	}
}
