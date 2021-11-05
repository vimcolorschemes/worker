package cli

import (
	"log"
	"math"
	"strings"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/emoji"
	"github.com/vimcolorschemes/worker/internal/github"

	"go.mongodb.org/mongo-driver/bson"

	gogithub "github.com/google/go-github/v32/github"
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

// Import potential vim color scheme repositories from GitHub
func Import(_force bool, repoKey string) bson.M {
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
		repositoryUpdateObject := getImportRepositoryObject(repository)
		database.UpsertRepository(*repository.ID, repositoryUpdateObject)
	}

	return bson.M{"repositoryCount": len(repositories)}
}

func getImportRepositoryObject(repository *gogithub.Repository) bson.M {
	description := emoji.ConvertColonEmojis(repository.GetDescription())
	return bson.M{
		"_id":             repository.GetID(),
		"owner.name":      repository.GetOwner().GetLogin(),
		"owner.avatarURL": repository.GetOwner().GetAvatarURL(),
		"name":            repository.GetName(),
		"description":     description,
		"githubURL":       repository.GetHTMLURL(),
		"githubCreatedAt": repository.GetCreatedAt().Time,
		"homepageURL":     repository.GetHomepage(),
		"size":            repository.GetSize(),
	}
}
