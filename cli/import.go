package cli

import (
	"fmt"
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

var countColorschemeFiles = github.CountColorschemeFiles
var getRepositoryIDs = database.GetRepositoryIDs

// Keep the job report small enough to travel in an SNS notification.
const repositoryDroppedNameSampleSize = 5

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
	checkedCount := 0
	droppedNames := []string{}

	if repoKey != "" {
		matches := strings.Split(repoKey, "/")
		if len(matches) < 2 {
			log.Panic("repo key not valid")
		}
		repository, err := github.GetRepository(matches[0], matches[1])
		if err != nil {
			log.Panic(err)
		}
		// An operator naming a repository overrides the colorscheme gate.
		repositories = []*gogithub.Repository{repository}
	} else {
		repositories = github.SearchRepositories(queries, repositoryCountLimit, repositoryCountLimitPerPage)
		repositories, checkedCount, droppedNames = filterColorschemeRepositories(repositories)
	}

	log.Print("Preparing import data for ", len(repositories), " repositories")
	data := make([]database.ImportData, 0, len(repositories))
	for _, repository := range repositories {
		log.Print("Preparing ", *repository.Name)
		data = append(data, getImportData(repository))
	}
	database.UpsertRepositoriesFromImport(data)

	return map[string]interface{}{
		"repositoryCount":        len(repositories),
		"repositoryCheckedCount": checkedCount,
		"repositoryDroppedCount": len(droppedNames),
		"repositoryDroppedNames": sampleDroppedNames(droppedNames),
	}
}

// filterColorschemeRepositories drops candidates that do not ship a colorscheme.
// Github search only matches name, description and topics, so it happily returns
// dotfiles and tutorials.
func filterColorschemeRepositories(repositories []*gogithub.Repository) ([]*gogithub.Repository, int, []string) {
	existingIDs, err := getRepositoryIDs()
	if err != nil {
		log.Panic(err)
	}

	kept := make([]*gogithub.Repository, 0, len(repositories))
	checkedCount := 0
	droppedNames := []string{}

	for _, repository := range repositories {
		if existingIDs[repository.GetID()] {
			kept = append(kept, repository)
			continue
		}

		ownerName := repository.GetOwner().GetLogin()
		name := repository.GetName()

		checkedCount++
		count, err := countColorschemeFiles(ownerName, name)
		if err != nil {
			log.Printf("Error counting colorscheme files for %s/%s, keeping it: %s", ownerName, name, err)
			kept = append(kept, repository)
			continue
		}

		if count == 0 {
			log.Printf("Dropping %s/%s, no colors/*.{vim,lua} file", ownerName, name)
			droppedNames = append(droppedNames, fmt.Sprintf("%s/%s", ownerName, name))
			continue
		}

		log.Printf("Keeping new repository %s/%s with %d colorscheme files", ownerName, name, count)
		kept = append(kept, repository)
	}

	log.Printf("Checked %d new candidates, dropped %d", checkedCount, len(droppedNames))

	return kept, checkedCount, droppedNames
}

func sampleDroppedNames(names []string) []string {
	if len(names) <= repositoryDroppedNameSampleSize {
		return names
	}
	return names[:repositoryDroppedNameSampleSize]
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
