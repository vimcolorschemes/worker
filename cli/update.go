package cli

import (
	"fmt"
	"log"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	"github.com/vimcolorschemes/worker/internal/file"
	"github.com/vimcolorschemes/worker/internal/github"
	repoUtil "github.com/vimcolorschemes/worker/internal/repository"
	"github.com/vimcolorschemes/worker/internal/vim"

	"go.mongodb.org/mongo-driver/bson"
)

// Update the imported repositories with all kinds of useful information
func Update() {
	log.Print("Run update")

	startTime := time.Now()

	repositories := database.GetRepositories()

	log.Print(len(repositories), " repositories to update")

	for _, repository := range repositories {
		fmt.Println()

		log.Print("Updating ", repository.Owner.Name, "/", repository.Name)

		updatedRepository := updateRepository(repository)

		updateObject := getUpdateRepositoryObject(updatedRepository)

		database.UpsertRepository(repository.ID, updateObject)
	}

	fmt.Println()

	elapsedTime := time.Since(startTime)
	log.Printf("Elapsed time: %s", elapsedTime)

	fmt.Println()

	log.Print("Creating update report")
	data := bson.M{"repositoryCount": len(repositories)}
	database.CreateReport("update", elapsedTime.Seconds(), data)

	fmt.Println()

	log.Print(":wq")
}

func updateRepository(repository repoUtil.Repository) repoUtil.Repository {
	githubRepository, err := github.GetRepository(repository.Owner.Name, repository.Name)
	if err != nil {
		log.Print("Error fetching ", repository.Owner.Name, "/", repository.Name)
		repository.Valid = false
		return repository
	}

	log.Print("Gathering basic infos")
	repository.StargazersCount = *githubRepository.StargazersCount

	log.Print("Fetching date of last commit")
	repository.LastCommitAt = github.GetLastCommitAt(githubRepository)

	log.Print("Building stargazers count history")
	repository.StargazersCountHistory = repoUtil.AppendToStargazersCountHistory(repository)

	log.Print("Computing week stargazers count")
	repository.WeekStargazersCount = repoUtil.ComputeTrendingStargazersCount(repository, 7)

	log.Print("Getting vim color scheme names")
	fileURLs := github.GetRepositoryFileURLs(githubRepository)
	log.Print(len(fileURLs), " files found")
	vimFileURLs := file.GetFileURLsWithExtensions(fileURLs, []string{"erb", "vim"})
	log.Print(len(vimFileURLs), " vim files found")
	if len(vimFileURLs) > 0 {
		log.Print("Checking for vim color scheme names")
		vimColorSchemes, err := vim.GetVimColorSchemes(vimFileURLs)
		if err != nil {
			log.Print("Did not find any vim color schemes")
		}
		repository.VimColorSchemes = vimColorSchemes
	}

	log.Print("Checking if ", repository.Owner.Name, "/", repository.Name, " is valid")
	repository.Valid = repoUtil.IsRepositoryValid(repository)
	log.Printf("Valid: %v", repository.Valid)

	return repository
}

func getUpdateRepositoryObject(repository repoUtil.Repository) bson.M {
	return bson.M{
		"lastCommitAt":           repository.LastCommitAt,
		"stargazersCount":        repository.StargazersCount,
		"stargazersCountHistory": repository.StargazersCountHistory,
		"weekStargazersCount":    repository.WeekStargazersCount,
		"vimColorSchemes":        repository.VimColorSchemes,
		"valid":                  repository.Valid,
	}
}
