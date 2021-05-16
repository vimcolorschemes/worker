package repository

import (
	"log"
	"sort"
	"time"

	gogithub "github.com/google/go-github/v32/github"

	"github.com/vimcolorschemes/worker/internal/date"
)

// Repository represents a repository as it's stored in the database
type Repository struct {
	ID                     int64                        `bson:"_id,omitempty"`
	Name                   string                       `bson:"name"`
	Owner                  Owner                        `bson:"owner"`
	GitHubURL              string                       `bson:"githubURL"`
	HomepageURL            string                       `bson:"homepageURL"`
	Valid                  bool                         `bson:"valid"`
	UpdatedAt              time.Time                    `bson:"updatedAt"`
	GeneratedAt            time.Time                    `bson:"generatedAt"`
	LastCommitAt           time.Time                    `bson:"lastCommitAt"`
	GitHubCreatedAt        time.Time                    `bson:"githubCreatedAt"`
	StargazersCount        int                          `bson:"stargazersCount"`
	StargazersCountHistory []StargazersCountHistoryItem `bson:"stargazersCountHistory"`
	WeekStargazersCount    int                          `bson:"weekStargazersCount"`
	VimColorSchemes        []VimColorScheme             `bson:"vimColorSchemes,omitempty"`
}

// Owner represents the owner of a repository
type Owner struct {
	Name      string `bson:"name"`
	AvatarURL string `bson:"avatarURL"`
}

// StargazersCountHistoryItem represents a repository's stargazers count at a given date
type StargazersCountHistoryItem struct {
	Date            time.Time `bson:"date"`
	StargazersCount int       `bson:"stargazersCount"`
}

// VimColorScheme represents a vim color scheme's meta data
type VimColorScheme struct {
	FileURL string             `bson:"fileURL"`
	Name    string             `bson:"name"`
	Data    VimColorSchemeData `bson:"data"`
	Valid   bool               `bson:"valid"`
}

// VimColorSchemeData represents the color values for light and dark backgrounds
type VimColorSchemeData struct {
	Light []VimColorSchemeGroup `bson:"light,omitempty"`
	Dark  []VimColorSchemeGroup `bson:"dark,omitempty"`
}

// VimColorSchemeGroup represents a vim color scheme group's data
type VimColorSchemeGroup struct {
	Name    string `bson:"name"`
	HexCode string `bson:"hexCode"`
}

// VimBackgroundValue sets up an enum containing the possible values for background in vim
type VimBackgroundValue string

const (
	// LightBackground represents light background value for vim
	LightBackground VimBackgroundValue = "light"

	// DarkBackground represents light background value for vim
	DarkBackground VimBackgroundValue = "dark"
)

// UniquifyRepositories makes sure no repository is listed twice in a list
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

// AppendToStargazersCountHistory appends today's item to the stargazers count history
func AppendToStargazersCountHistory(repository Repository) []StargazersCountHistoryItem {
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

// ComputeTrendingStargazersCount adds up the stargazers counts from the last days
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

// IsRepositoryValid returns true if a repository is considered valid from our standards
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
