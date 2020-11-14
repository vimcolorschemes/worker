package repository

import (
	"log"
	"sort"
	"time"

	gogithub "github.com/google/go-github/v32/github"

	"github.com/vimcolorschemes/worker/internal/date"
)

type Repository struct {
	ID                     int64                        `bson:"_id,omitempty"`
	Name                   string                       `bson:"name"`
	Owner                  RepositoryOwner              `bson:"owner"`
	Valid                  bool                         `bson:"valid"`
	LastCommitAt           time.Time                    `bson:"lastCommitAt"`
	StargazersCount        int                          `bson:"stargazersCount"`
	StargazersCountHistory []StargazersCountHistoryItem `bson:"stargazersCountHistory"`
	WeekStargazersCount    int                          `bson:"weekStargazersCount"`
	VimColorSchemes        []ColorScheme                `bson:"vimColorSchemes"`
}

type RepositoryOwner struct {
	Name      string `bson:"name"`
	AvatarURL string `bson:"avatarURL"`
}

type StargazersCountHistoryItem struct {
	Date            time.Time `bson:"date"`
	StargazersCount int       `bson:"stargazersCount"`
}

type ColorScheme struct {
	FileURL string `bson:"fileURL"`
	Name    string `bson:"name"`
}

func UniquifyRepositories(repositories []*gogithub.Repository) []*gogithub.Repository {
	keys := make(map[int64]bool)
	unique := []*gogithub.Repository{}

	for _, repository := range repositories {
		if _, value := keys[*repository.ID]; !value {
			keys[*repository.ID] = true
			unique = append(unique, repository)
		}
	}

	return unique
}

func GetStargazersCountHistory(repository Repository) []StargazersCountHistoryItem {
	history := repository.StargazersCountHistory
	if history == nil {
		history = []StargazersCountHistoryItem{}
	}

	// sort: newest first
	if len(history) > 0 {
		sort.Slice(history, func(i int, j int) bool {
			return history[i].Date.After(history[j].Date)
		})
	}

	today := date.RoundTimeToDate(time.Now().UTC())

	// remove present entries for today
	for index := 0; index < len(history); {
		item := history[index]
		if item.Date == today {
			history = append(history[:index], history[index+1:]...)
		} else {
			index++
		}
	}

	todaysHistoryItem := StargazersCountHistoryItem{
		Date:            today,
		StargazersCount: repository.StargazersCount,
	}

	// prepend new history item
	history = append([]StargazersCountHistoryItem{todaysHistoryItem}, history...)

	if len(history) > 31 {
		// remove last item
		lastIndex := len(history) - 1
		history = append(history[:lastIndex], history[lastIndex+1:]...)
	}

	return history
}

func ComputeTrendingStargazersCount(repository Repository, dayCount int) int {
	if repository.StargazersCountHistory == nil || len(repository.StargazersCountHistory) == 0 {
		return 0
	}

	firstIndex := len(repository.StargazersCountHistory) - 1
	if dayCount-1 < firstIndex {
		firstIndex = dayCount - 1
	}

	firstDayCount := repository.StargazersCountHistory[firstIndex].StargazersCount
	lastDayCount := repository.StargazersCountHistory[0].StargazersCount

	return lastDayCount - firstDayCount
}

func IsRepositoryValid(repository Repository) bool {
	var defaultTime time.Time
	if repository.LastCommitAt == defaultTime {
		log.Print("Repository last commit at is not valid")
		return false
	}

	if repository.StargazersCount < 1 {
		log.Print("Repository does not have enough stars")
		return false
	}

	if len(repository.StargazersCountHistory) < 1 {
		log.Print("Repository stargazers count history is empty")
		return false
	}

	if !date.IsSameDay(repository.StargazersCountHistory[0].Date, date.Today()) {
		log.Print("Repository stargazers count history last entry is not today")
		return false
	}

	if len(repository.VimColorSchemes) < 1 {
		log.Print("Repository does not have vim color schemes")
		return false
	}

	return true
}
