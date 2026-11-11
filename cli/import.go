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
	"topic:vim-colorscheme",
	"topic:neovim-colorscheme",
	"topic:vim-theme",
	"topic:neovim-theme",
	"topic:nvim-theme",
	"topic:vim-colorschemes",
	"topic:neovim-colorschemes",
	"topic:nvim-colorscheme",
	"topic:nvim-colorschemes",
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
	log.Printf("Repository limit per query: %d", repositoryCountLimit)

	var repositories []*gogithub.Repository
	checkedCount := 0
	addedNames := []string{}
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
		repositories, checkedCount, addedNames, droppedNames = filterColorschemeRepositories(repositories)
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
		"repositoryAddedCount":   len(addedNames),
		"repositoryAddedNames":   addedNames,
		"repositoryDroppedCount": len(droppedNames),
		"repositoryDroppedNames": droppedNames,
	}
}

// filterColorschemeRepositories drops candidates that do not ship a colorscheme.
// Github search only matches name, description and topics, so it happily returns
// dotfiles and tutorials.
func filterColorschemeRepositories(repositories []*gogithub.Repository) ([]*gogithub.Repository, int, []string, []string) {
	existingIDs, err := getRepositoryIDs()
	if err != nil {
		log.Panic(err)
	}

	kept := make([]*gogithub.Repository, 0, len(repositories))
	checkedCount := 0
	addedNames := []string{}
	droppedNames := []string{}

	for _, repository := range repositories {
		if existingIDs[repository.GetID()] {
			kept = append(kept, repository)
			continue
		}

		ownerName := repository.GetOwner().GetLogin()
		name := repository.GetName()

		checkedCount++
		fullName := fmt.Sprintf("%s/%s", ownerName, name)
		count, err := countColorschemeFiles(ownerName, name)
		if err != nil {
			log.Printf("Error counting colorscheme files for %s, keeping it: %s", fullName, err)
			kept = append(kept, repository)
			addedNames = append(addedNames, fullName)
			continue
		}

		if count == 0 {
			log.Printf("Dropping %s, no root colors/ or after/colors/ *.{vim,lua} file", fullName)
			droppedNames = append(droppedNames, fullName)
			continue
		}

		log.Printf("Keeping new repository %s with %d colorscheme files", fullName, count)
		kept = append(kept, repository)
		addedNames = append(addedNames, fullName)
	}

	log.Printf("Checked %d new candidates, added %d, dropped %d", checkedCount, len(addedNames), len(droppedNames))

	return kept, checkedCount, addedNames, droppedNames
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
