package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/github"
	repoUtil "github.com/vimcolorschemes/worker/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
)

func Update() {
	log.Print("Run update")

	repositories := database.GetRepositories()

	log.Print(len(repositories), " repositories to update")

	for _, repository := range repositories {
		log.Print("Updating ", repository.Owner.Name, "/", repository.Name)

		updatedRepository := updateRepository(repository)

		updateObject := getUpdateRepositoryObject(updatedRepository)

		database.UpsertRepository(repository.ID, updateObject)
	}

	log.Print(":wq")
}

func updateRepository(repository repoUtil.Repository) repoUtil.Repository {
	githubRepository, err := github.GetRepository(repository.Owner.Name, repository.Name)
	if err != nil {
		log.Print("Error fetching ", repository.Owner.Name, "/", repository.Name)
		repository.Valid = false
		return repository
	}

	repository.LastCommitAt = github.GetLastCommitAt(githubRepository)

	repository.StargazersCount = *githubRepository.StargazersCount

	repository.StargazersCountHistory = repoUtil.GetStargazersCountHistory(repository)

	repository.Valid = true

	return repository
}

func getUpdateRepositoryObject(repository repoUtil.Repository) bson.M {
	return bson.M{
		"lastCommitAt":           repository.LastCommitAt,
		"stargazersCount":        repository.StargazersCount,
		"stargazersCountHistory": repository.StargazersCountHistory,
		"valid":                  repository.Valid,
	}
}
