package cli

import (
	"context"
	"log"
	"math"
	"strings"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/github"
	"github.com/vimcolorschemes/worker/internal/store"

	gogithub "github.com/google/go-github/v68/github"
)

var repositoryCountLimit int
var repositoryCountLimitPerPage int
var repositoryStore *store.RepositoryStore

var queries = []string{
	"vim theme",
	"vim color scheme",
	"vim colorscheme",
	"vim colour scheme",
	"vim colourscheme",
	"neovim theme",
	"neovim color scheme",
	"neovim colorscheme",
	"neovim colour scheme",
	"neovim colourscheme",
}

func init() {
	repositoryStore = store.NewRepositoryStore(database.Connect())

	repositoryCountLimitValue, err := dotenv.GetInt("GITHUB_REPOSITORY_COUNT_LIMIT")
	if err != nil {
		repositoryCountLimitValue = 100
	}
	repositoryCountLimit = repositoryCountLimitValue

	repositoryCountLimitPerPage = int(math.Min(float64(repositoryCountLimit), 100))
}

// Import potential vim color scheme repositories from Github
func Import(_force bool, _debug bool, repoKey string) int {
	log.Printf("Repository limit: %d", repositoryCountLimit)

	var repositories []*gogithub.Repository
	if repoKey != "" {
		matches := strings.Split(repoKey, "/")
		if len(matches) < 2 {
			log.Panic("repo key not valid")
		}
		repository, err := github.GetRepository(matches[0], matches[1])
		if err != nil {
			log.Panic(err)
		}
		repositories = []*gogithub.Repository{repository}
	} else {
		repositories = github.SearchRepositories(queries, repositoryCountLimit, repositoryCountLimitPerPage)
	}

	log.Print("Upserting ", len(repositories), " repositories")
	for _, repository := range repositories {
		log.Print("Upserting ", *repository.Name)
		err := repositoryStore.Upsert(context.TODO(), store.Repository{
			ID:          *repository.ID,
			Name:        *repository.Name,
			Owner:       *repository.Owner.Login,
			Description: *repository.Description,
			CreatedAt:   *repository.CreatedAt.GetTime(),
			UpdatedAt:   *repository.PushedAt.GetTime(),
		})
		if err != nil {
			log.Println("Error upserting repository:", err)
		}
	}

	return len(repositories)
}
