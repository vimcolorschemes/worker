package cli

import (
	"log"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/github"

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
		// 3. stargazersCountHistory
		// 4. weekStargazersCount
		// 5. vimColorSchemeNames
	}

	log.Print(":wq")
}

func updateRepository(repository database.Repository) database.Repository {
	githubRepository, err := github.GetRepository(repository.Owner.Name, repository.Name)

	if err != nil {
		log.Print("Error fetching ", repository.Owner.Name, "/", repository.Name)
		repository.Valid = false
		return repository
	}

	lastCommitAt, err := github.GetLastCommitAt(githubRepository)

	if err != nil {
		log.Print("Error getting last commit of ", repository.Owner.Name, "/", repository.Name)
		repository.Valid = false
		return repository
	}

	repository.LastCommitAt = lastCommitAt

	repository.Valid = true

	return repository
}

func getUpdateRepositoryObject(repository database.Repository) bson.M {
	return bson.M{
		"lastCommitAt": repository.LastCommitAt,
		"valid":        repository.Valid,
	}
}
