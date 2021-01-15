package cli

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/dotenv"
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
func Import(_force bool) {
	log.Print("Run import")

	log.Printf("Repository limit: %d", repositoryCountLimit)

	startTime := time.Now()

	repositories := github.SearchRepositories(queries, repositoryCountLimit, repositoryCountLimitPerPage)

	log.Print("Upserting ", len(repositories), " repositories")
	for _, repository := range repositories {
		log.Print("Upserting ", *repository.Name)
		repositoryUpdateObject := getImportRepositoryObject(repository)
		database.UpsertRepository(*repository.ID, repositoryUpdateObject)
	}

	fmt.Println()

	elapsedTime := time.Since(startTime)
	log.Printf("Elapsed time: %s", elapsedTime)

	fmt.Println()

	log.Print("Creating import report")
	data := bson.M{"repositoryCount": len(repositories)}
	database.CreateReport("import", elapsedTime.Seconds(), data)

	fmt.Println()

	log.Print(":wq")
}

func getImportRepositoryObject(repository *gogithub.Repository) bson.M {
	return bson.M{
		"_id":             repository.GetID(),
		"owner.name":      repository.GetOwner().GetLogin(),
		"owner.avatarURL": repository.GetOwner().GetAvatarURL(),
		"name":            repository.GetName(),
		"description":     repository.GetDescription(),
		"githubURL":       repository.GetHTMLURL(),
		"githubCreatedAt": repository.GetCreatedAt().Time,
		"homepageURL":     repository.GetHomepage(),
	}
}
