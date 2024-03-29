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
	Owner                  Owner                        `bson:"owner"`
	Name                   string                       `bson:"name"`
	GitHubURL              string                       `bson:"githubURL"`
	HomepageURL            string                       `bson:"homepageURL"`
	StargazersCount        int                          `bson:"stargazersCount"`
	StargazersCountHistory []StargazersCountHistoryItem `bson:"stargazersCountHistory"`
	WeekStargazersCount    int                          `bson:"weekStargazersCount"`
	License                string                       `bson:"license"`
	Size                   int                          `bson:"size"`
	GitHubCreatedAt        time.Time                    `bson:"githubCreatedAt"`
	LastCommitAt           time.Time                    `bson:"lastCommitAt"`
	VimColorSchemes        []VimColorScheme             `bson:"vimColorSchemes,omitempty"`
	IsVim                  bool                         `bson:"isVim"`
	IsLua                  bool                         `bson:"isLua"`
	UpdateValid            bool                         `bson:"updateValid"`
	UpdatedAt              time.Time                    `bson:"updatedAt"`
	GenerateValid          bool                         `bson:"generateValid"`
	GeneratedAt            time.Time                    `bson:"generatedAt"`
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
	FileURL      string             `bson:"fileURL"`
	Name         string             `bson:"name"`
	Data         VimColorSchemeData `bson:"data"`
	Valid        bool               `bson:"valid"`
	Backgrounds  []string           `bson:"backgrounds"`
	IsLua        bool               `bson:"isLua"`
	LastCommitAt time.Time          `bson:"lastCommitAt"`
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
func (repository Repository) AppendToStargazersCountHistory() []StargazersCountHistoryItem {
	history := repository.StargazersCountHistory
	if history == nil {
		history = []StargazersCountHistoryItem{}
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
func (repository Repository) ComputeTrendingStargazersCount(dayCount int) int {
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

// IsValidAfterUpdate returns true if a repository is considered
// valid from our standards after an update job
func (repository Repository) IsValidAfterUpdate() bool {
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

// IsValidAfterGenerate returns true if a repository is considered
// valid from our standards after a generate job
func (repository Repository) IsValidAfterGenerate() bool {
	if !repository.UpdateValid {
		return false
	}

	var hasValidVimColorScheme = false
	for _, vimColorScheme := range repository.VimColorSchemes {
		if vimColorScheme.Valid {
			hasValidVimColorScheme = true
		}
	}

	if !hasValidVimColorScheme {
		log.Print("Repository does not have a valid vim color scheme")
		return false
	}

	return true
}

// SyncVimColorSchemes replaces the vim color scheme list of a repository with
// a new one while keeping as much data from the old list as possible
func (repository *Repository) SyncVimColorSchemes(vimColorSchemes []VimColorScheme) {
	var newList []VimColorScheme

	for _, vimColorScheme := range uniquifyVimColorSchemes(vimColorSchemes) {
		match, exists := repository.getMatchingVimColorScheme(vimColorScheme.Name)
		if !exists {
			newList = append(newList, vimColorScheme)
		} else {
			match.FileURL = vimColorScheme.FileURL
			match.IsLua = vimColorScheme.IsLua
			match.LastCommitAt = vimColorScheme.LastCommitAt
			newList = append(newList, match)
		}
	}

	repository.VimColorSchemes = newList
}

// AssignRepositoryType defines is a repository is vim, lua, or both
func (repository *Repository) AssignRepositoryType() {
	for _, vimColorScheme := range repository.VimColorSchemes {
		if repository.IsLua && repository.IsVim {
			break
		}

		if vimColorScheme.IsLua {
			repository.IsLua = true
		} else {
			repository.IsVim = true
		}
	}
}

func uniquifyVimColorSchemes(vimColorSchemes []VimColorScheme) []VimColorScheme {
	groups := make(map[string][]VimColorScheme)

	for _, vimColorScheme := range vimColorSchemes {
		groups[vimColorScheme.Name] = append(groups[vimColorScheme.Name], vimColorScheme)
	}

	var uniqueList []VimColorScheme

	for _, group := range groups {
		chosen := group[0]

		for _, vimColorScheme := range group {
			if vimColorScheme.LastCommitAt.After(chosen.LastCommitAt) {
				chosen = vimColorScheme
			}
		}

		uniqueList = append(uniqueList, chosen)
	}

	return uniqueList
}

func (repository Repository) getMatchingVimColorScheme(name string) (VimColorScheme, bool) {
	for _, vimColorScheme := range repository.VimColorSchemes {
		if vimColorScheme.Name == name {
			return vimColorScheme, true
		}
	}
	return VimColorScheme{}, false
}
