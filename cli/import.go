package cli

import (
	"log"
	"math"
	"strings"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/github"

	gogithub "github.com/google/go-github/v68/github"
)

var repositoryCountLimit int
var repositoryCountLimitPerPage int

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
	repositoryCountLimitValue, err := dotenv.GetInt("GITHUB_REPOSITORY_COUNT_LIMIT")
	if err != nil {
		repositoryCountLimitValue = 100
	}
	repositoryCountLimit = repositoryCountLimitValue

	repositoryCountLimitPerPage = int(math.Min(float64(repositoryCountLimit), 100))
}

// Import potential colorscheme repositories from Github
func Import(_force bool, _debug bool, repoKey string) map[string]interface{} {
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
		data := getImportData(repository)
		database.UpsertRepositoryFromImport(data)
	}

	return map[string]interface{}{"repositoryCount": len(repositories)}
}

func getImportData(repository *gogithub.Repository) database.ImportData {
	return database.ImportData{
		ID:              repository.GetID(),
		OwnerName:       repository.GetOwner().GetLogin(),
		OwnerAvatarURL:  repository.GetOwner().GetAvatarURL(),
		Name:            repository.GetName(),
		Description:     repository.GetDescription(),
		GithubURL:       repository.GetHTMLURL(),
		GithubCreatedAt: repository.GetCreatedAt().Time,
		PushedAt:        repository.GetPushedAt().Time,
	}
}
