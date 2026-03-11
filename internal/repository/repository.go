package repository

import (
	"log"
	"sort"
	"time"

	gogithub "github.com/google/go-github/v68/github"

	"github.com/vimcolorschemes/worker/internal/date"
)

// Repository represents a repository as it's stored in the database
type Repository struct {
	ID                     int64                        `json:"id"`
	Owner                  Owner                        `json:"owner"`
	Name                   string                       `json:"name"`
	GithubURL              string                       `json:"githubURL"`
	StargazersCount        int                          `json:"stargazersCount"`
	StargazersCountHistory []StargazersCountHistoryItem `json:"stargazersCountHistory"`
	WeekStargazersCount    int                          `json:"weekStargazersCount"`
	GithubCreatedAt        time.Time                    `json:"githubCreatedAt"`
	PushedAt               time.Time                    `json:"pushedAt"`
	Colorschemes           []Colorscheme                `json:"colorschemes,omitempty"`
	IsEligible             bool                         `json:"isEligible"`
	UpdatedAt              time.Time                    `json:"updatedAt"`
}

// Owner represents the owner of a repository
type Owner struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatarURL"`
}

// StargazersCountHistoryItem represents a repository's stargazers count at a given date
type StargazersCountHistoryItem struct {
	Date            time.Time `json:"date"`
	StargazersCount int       `json:"stargazersCount"`
}

// Colorscheme represents a colorscheme's metadata
type Colorscheme struct {
	Name        string            `json:"name"`
	Data        ColorschemeData   `json:"data"`
	Backgrounds []BackgroundValue `json:"backgrounds"`
}

// ColorschemeData represents the color values for light and dark backgrounds
type ColorschemeData struct {
	Light []ColorschemeGroup `json:"light,omitempty"`
	Dark  []ColorschemeGroup `json:"dark,omitempty"`
}

// ColorschemeGroup represents a colorscheme group's data
type ColorschemeGroup struct {
	Name    string `json:"name"`
	HexCode string `json:"hexCode"`
}

// BackgroundValue sets up an enum containing possible background values
type BackgroundValue string

const (
	// LightBackground represents light background value
	LightBackground BackgroundValue = "light"

	// DarkBackground represents dark background value
	DarkBackground BackgroundValue = "dark"
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
func (repository Repository) AppendToStargazersCountHistory() []StargazersCountHistoryItem {
	history := repository.StargazersCountHistory
	if history == nil {
		history = []StargazersCountHistoryItem{
			{
				Date:            repository.GithubCreatedAt,
				StargazersCount: repository.StargazersCount,
			},
		}
	} // sort: newest first
	if len(history) > 0 {
		sort.Slice(history, func(i int, j int) bool {
			return history[i].Date.After(history[j].Date)
		})
	}

	today := date.RoundTimeToDate(time.Now().UTC())

	// remove present entries for today
	for index := 0; index < len(history); {
		item := history[index]
		if item.Date.Equal(today) {
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
func (repository Repository) ComputeTrendingStargazersCount(dayCount int) int {
	if len(repository.StargazersCountHistory) == 0 {
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

// IsEligibleAfterUpdate returns true if a repository is considered
// eligible from our standards after an update job
func (repository Repository) IsEligibleAfterUpdate() bool {
	if repository.PushedAt.IsZero() {
		log.Print("Repository last commit date is not valid")
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

	return true
}
