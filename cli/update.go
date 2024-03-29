package cli

import (
	"fmt"
	"log"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/file"
	"github.com/vimcolorschemes/worker/internal/github"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
	"github.com/vimcolorschemes/worker/internal/vim"

	"go.mongodb.org/mongo-driver/bson"
)

// Update the imported repositories with all kinds of useful information
func Update(force bool, _debug bool, repoKey string) bson.M {
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

	for _, repository := range repositories {
		fmt.Println()

		log.Print("Updating ", repository.Owner.Name, "/", repository.Name)

		updatedRepository := updateRepository(repository, force)

		updateObject := getUpdateRepositoryObject(updatedRepository)

		database.UpsertRepository(repository.ID, updateObject)
	}

	return bson.M{"repositoryCount": len(repositories)}
}

func updateRepository(repository repoHelper.Repository, force bool) repoHelper.Repository {
	githubRepository, err := github.GetRepository(repository.Owner.Name, repository.Name)
	if err != nil {
		log.Print("Error fetching ", repository.Owner.Name, "/", repository.Name)
		repository.UpdateValid = false
		return repository
	}

	license := githubRepository.License
	if license != nil {
		repository.License = *license.SPDXID
	} else {
		repository.License = ""
	}

	log.Print("Gathering basic infos")
	repository.StargazersCount = *githubRepository.StargazersCount

	log.Print("Fetching date of last commit")
	repository.LastCommitAt = github.GetLastCommitAt(githubRepository)

	log.Print("Building stargazers count history")
	repository.StargazersCountHistory = repository.AppendToStargazersCountHistory()

	log.Print("Computing week stargazers count")
	repository.WeekStargazersCount = repository.ComputeTrendingStargazersCount(7)

	if !force && repository.UpdatedAt.After(repository.LastCommitAt) {
		log.Print("Repository is not due for a full update")
		return repository
	}

	log.Print("Getting vim color scheme names")
	files := github.GetRepositoryFiles(githubRepository)
	log.Print(len(files), " files found")
	vimFiles := file.GetFilesWithExtensions(files, []string{"erb", "vim", "lua"})
	log.Print(len(vimFiles), " vim/lua files found")
	if len(vimFiles) > 0 {
		log.Print("Checking for vim color scheme names")
		vimColorSchemes, err := vim.GetVimColorSchemes(githubRepository, vimFiles)
		if err != nil {
			log.Print("Did not find any vim color schemes")
		}
		repository.SyncVimColorSchemes(vimColorSchemes)
	}

	repository.AssignRepositoryType()

	log.Print("Checking if ", repository.Owner.Name, "/", repository.Name, " is valid")
	repository.UpdateValid = repository.IsValidAfterUpdate()
	log.Printf("Update valid: %v", repository.UpdateValid)

	return repository
}

func getUpdateRepositoryObject(repository repoHelper.Repository) bson.M {
	return bson.M{
		"license":                repository.License,
		"lastCommitAt":           repository.LastCommitAt,
		"stargazersCount":        repository.StargazersCount,
		"stargazersCountHistory": repository.StargazersCountHistory,
		"weekStargazersCount":    repository.WeekStargazersCount,
		"vimColorSchemes":        repository.VimColorSchemes,
		"isLua":                  repository.IsLua,
		"isVim":                  repository.IsVim,
		"updateValid":            repository.UpdateValid,
		"updatedAt":              time.Now(),
	}
}
