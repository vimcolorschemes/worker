package cli

import (
	"context"
	"log"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/github"
	"github.com/vimcolorschemes/worker/internal/store"
)

var repositoryStargazersCountStore *store.RepositoryStargarzersCountStore

func init() {
	repositoryStargazersCountStore = store.NewRepositoryStargazersCountStore(database.Connect())
}

// Update the imported repositories with all kinds of useful information
func Update(_force bool, _debug bool, repoKey string) int {
	var repositories []store.Repository
	if repoKey != "" {
		repository, err := repositoryStore.GetByKey(context.TODO(), repoKey)
		if err != nil {
			log.Panic(err)
		}
		repositories = []store.Repository{*repository}
	} else {
		repositories = repositoryStore.GetAll()
	}

	log.Print(len(repositories), " repositories to update")

	for index, repository := range repositories {
		log.Print("Updating ", index, " of ", len(repositories), ": ", repository.Owner, "/", repository.Name)

		githubRepository, err := github.GetRepository(repository.Owner, repository.Name)
		if err != nil {
			log.Print("Error fetching ", repository.Owner, "/", repository.Name)
			continue
		}

		err = repositoryStore.Upsert(context.TODO(), store.Repository{
			ID:          *githubRepository.ID,
			Name:        *githubRepository.Name,
			Owner:       *githubRepository.Owner.Login,
			Description: *githubRepository.Description,
			CreatedAt:   *githubRepository.CreatedAt.GetTime(),
			UpdatedAt:   *githubRepository.PushedAt.GetTime(),
		})
		if err != nil {
			log.Println("Error upserting repository:", err)
		}

		err = repositoryStargazersCountStore.Insert(context.TODO(), store.RepositoryStargazersCount{
			RepositoryID:    *githubRepository.ID,
			StargazersCount: *githubRepository.StargazersCount,
		})
		if err != nil {
			log.Println("Error inserting stargazers count:", err)
		}
	}

	return len(repositories)
}
