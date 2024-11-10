package repository

import (
	"reflect"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v62/github"
	dateUtil "github.com/vimcolorschemes/worker/internal/date"
)

func TestUniquifyRepositories(t *testing.T) {
	t.Run("should keep the only repository", func(t *testing.T) {
		id := int64(12345)
		list := []*gogithub.Repository{{ID: &id}}

		unique := UniquifyRepositories(list)
		expected := []*gogithub.Repository{{ID: &id}}

		if !reflect.DeepEqual(unique, expected) {
			t.Errorf("Incorrect result for UniquifyRepositories, got length: %d, want length: %d", len(unique), 1)
		}
	})

	t.Run("should remove duplicate", func(t *testing.T) {
		id1 := int64(12345)
		id2 := int64(12345)
		list := []*gogithub.Repository{{ID: &id1}, {ID: &id2}}

		unique := UniquifyRepositories(list)
		expected := []*gogithub.Repository{{ID: &id1}}

		if !reflect.DeepEqual(unique, expected) {
			t.Errorf("Incorrect result for UniquifyRepositories, got length: %d, want length: %d", len(unique), 1)
		}
	})

	t.Run("should remove duplicate and keep non duplicates", func(t *testing.T) {
		id1 := int64(12345)
		id2 := int64(12345)
		id3 := int64(12346)
		list := []*gogithub.Repository{{ID: &id1}, {ID: &id2}, {ID: &id3}}

		unique := UniquifyRepositories(list)
		expected := []*gogithub.Repository{{ID: &id1}, {ID: &id3}}

		if !reflect.DeepEqual(unique, expected) {
			t.Errorf("Incorrect result for UniquifyRepositories, got length: %d, want length: %d", len(unique), 2)
		}
	})
}

func TestAppendToStargazersCountHistory(t *testing.T) {
	t.Run("should initialize history when it's empty", func(t *testing.T) {
		repository := Repository{
			GitHubCreatedAt: time.Date(1990, time.November, 1, 0, 0, 0, 0, time.UTC),
			StargazersCount: 100,
		}
		history := repository.AppendToStargazersCountHistory()
		today := dateUtil.Today()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 100},
			{Date: repository.GitHubCreatedAt, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for AppendToStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should add item when there are some already", func(t *testing.T) {
		date := time.Date(1990, time.November, 1, 0, 0, 0, 0, time.UTC)

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date, StargazersCount: 100},
			},
		}

		history := repository.AppendToStargazersCountHistory()

		today := dateUtil.Today()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for AppendToStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should sort items by date (desc)", func(t *testing.T) {
		date1 := time.Date(1990, time.November, 10, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(1990, time.November, 20, 0, 0, 0, 0, time.UTC)
		date3 := time.Date(1990, time.November, 30, 0, 0, 0, 0, time.UTC)

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date1, StargazersCount: 90},
				{Date: date2, StargazersCount: 100},
				{Date: date3, StargazersCount: 110},
			},
		}

		history := repository.AppendToStargazersCountHistory()

		today := dateUtil.Today()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date3, StargazersCount: 110},
			{Date: date2, StargazersCount: 100},
			{Date: date1, StargazersCount: 90},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for AppendToStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should override today's item", func(t *testing.T) {
		date := time.Date(1990, time.November, 30, 0, 0, 0, 0, time.UTC)

		today := dateUtil.Today()

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
			},
		}

		history := repository.AppendToStargazersCountHistory()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for AppendToStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should override all of today's items", func(t *testing.T) {
		date := time.Date(1990, time.November, 30, 0, 0, 0, 0, time.UTC)

		today := dateUtil.Today()

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
			},
		}

		history := repository.AppendToStargazersCountHistory()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for AppendToStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should keep only 31 days worth of history", func(t *testing.T) {
		history := []StargazersCountHistoryItem{}

		for i := 1; i <= 31; i++ {
			date := time.Date(1990, time.November, i, 0, 0, 0, 0, time.UTC)
			item := StargazersCountHistoryItem{Date: date, StargazersCount: 100}
			history = append(history, item)
		}

		repository := Repository{
			StargazersCount:        101,
			StargazersCountHistory: history,
		}

		result := repository.AppendToStargazersCountHistory()

		if len(result) != 31 {
			t.Errorf("Incorrect result for AppendToStargazersCountHistory, got length: %d, want length: %d", len(result), 31)
		}
	})
}

func TestComputeTrendingStargazersCount(t *testing.T) {
	t.Run("should sum the difference for all items when count is equal to days count", func(t *testing.T) {
		today := dateUtil.Today()

		history := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 120},
			{Date: today, StargazersCount: 100},
		}

		repository := Repository{
			StargazersCountHistory: history,
		}

		result := repository.ComputeTrendingStargazersCount(2)

		if result != 20 {
			t.Errorf("Incorrect result for ComputeTrendingStargazersCount, got: %d, want: %d", result, 20)
		}
	})

	t.Run("should sum the difference for all items when count is less than days count", func(t *testing.T) {
		today := dateUtil.Today()

		history := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 120},
			{Date: today, StargazersCount: 100},
		}

		repository := Repository{
			StargazersCountHistory: history,
		}

		result := repository.ComputeTrendingStargazersCount(3)

		if result != 20 {
			t.Errorf("Incorrect result for ComputeTrendingStargazersCount, got: %d, want: %d", result, 20)
		}
	})

	t.Run("should sum the difference for first items when count is more than days count", func(t *testing.T) {
		today := dateUtil.Today()

		history := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 150},
			{Date: today, StargazersCount: 140},
			{Date: today, StargazersCount: 130},
			{Date: today, StargazersCount: 120},
			{Date: today, StargazersCount: 110},
			{Date: today, StargazersCount: 100},
		}

		repository := Repository{
			StargazersCountHistory: history,
		}

		result := repository.ComputeTrendingStargazersCount(2)

		if result != 10 {
			t.Errorf("Incorrect result for ComputeTrendingStargazersCount, got: %d, want: %d", result, 10)
		}
	})

	t.Run("should return 0 when there is no items", func(t *testing.T) {
		history := []StargazersCountHistoryItem{}

		repository := Repository{
			StargazersCountHistory: history,
		}

		result := repository.ComputeTrendingStargazersCount(2)

		if result != 0 {
			t.Errorf("Incorrect result for ComputeTrendingStargazersCount, got: %d, want: %d", result, 0)
		}
	})

	t.Run("should handle negative counts", func(t *testing.T) {
		today := dateUtil.Today()

		history := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 100},
			{Date: today, StargazersCount: 120},
		}

		repository := Repository{
			StargazersCountHistory: history,
		}

		result := repository.ComputeTrendingStargazersCount(2)

		if result != -20 {
			t.Errorf("Incorrect result for ComputeTrendingStargazersCount, got: %d, want: %d", result, -20)
		}
	})

	t.Run("should handle 'back-to-zero' counts", func(t *testing.T) {
		today := dateUtil.Today()

		history := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 120},
			{Date: today, StargazersCount: 100},
			{Date: today, StargazersCount: 120},
		}

		repository := Repository{
			StargazersCountHistory: history,
		}

		result := repository.ComputeTrendingStargazersCount(3)

		if result != 0 {
			t.Errorf("Incorrect result for ComputeTrendingStargazersCount, got: %d, want: %d", result, 0)
		}
	})
}

func TestComputeRepositoryValidityAfterUpdate(t *testing.T) {
	t.Run("should return valid for a repository that checks all boxes", func(t *testing.T) {
		var repository Repository
		repository.LastCommitAt = time.Now()
		repository.StargazersCount = 1
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: dateUtil.Today(), StargazersCount: 1}}

		isValid := repository.IsValidAfterUpdate()
		if !isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, true)
		}
	})

	t.Run("should return invalid for a repository with no last commit at", func(t *testing.T) {
		var repository Repository
		repository.StargazersCount = 1
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: dateUtil.Today(), StargazersCount: 1}}
		repository.VimColorSchemes = []VimColorScheme{{Name: "test"}}

		isValid := repository.IsValidAfterUpdate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, false)
		}
	})

	t.Run("should return invalid for a repository with a non strictly positive stargazers count", func(t *testing.T) {
		var repository Repository
		repository.LastCommitAt = time.Now()
		repository.StargazersCount = 0
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: dateUtil.Today(), StargazersCount: 1}}
		repository.VimColorSchemes = []VimColorScheme{{Name: "test"}}

		isValid := repository.IsValidAfterUpdate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, false)
		}
	})

	t.Run("should return invalid for a repository with an empty stargazers count history", func(t *testing.T) {
		var repository Repository
		repository.LastCommitAt = time.Now()
		repository.StargazersCount = 1
		repository.StargazersCountHistory = []StargazersCountHistoryItem{}
		repository.VimColorSchemes = []VimColorScheme{{Name: "test"}}

		isValid := repository.IsValidAfterUpdate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, false)
		}
	})

	t.Run("should return invalid for a repository with an outdated stargazers count history", func(t *testing.T) {
		var repository Repository
		repository.LastCommitAt = time.Now()
		repository.StargazersCount = 1
		date := time.Date(1990, time.November, 01, 0, 0, 0, 0, time.UTC)
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: date, StargazersCount: 1}}
		repository.VimColorSchemes = []VimColorScheme{{Name: "test"}}

		isValid := repository.IsValidAfterUpdate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, false)
		}
	})
}

func TestComputeRepositoryValidityAfterGenerate(t *testing.T) {
	t.Run("should return valid for a repository that checks all boxes", func(t *testing.T) {
		var repository Repository
		repository.VimColorSchemes = []VimColorScheme{{Name: "test"}}
		repository.UpdateValid = true

		isValid := repository.IsValidAfterGenerate()
		if !isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterGenerate, got: %v, want: %v", isValid, true)
		}
	})

	t.Run("should return invalid for a repository with no valid vim color schemes", func(t *testing.T) {
		var repository Repository
		repository.StargazersCount = 1
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: dateUtil.Today(), StargazersCount: 1}}
		repository.VimColorSchemes = []VimColorScheme{{Name: "test"}}

		isValid := repository.IsValidAfterGenerate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterGenerate, got: %v, want: %v", isValid, false)
		}
	})

	t.Run("should return invalid for a repository with no vim color schemes", func(t *testing.T) {
		var repository Repository
		repository.StargazersCount = 1
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: dateUtil.Today(), StargazersCount: 1}}
		repository.VimColorSchemes = []VimColorScheme{}

		isValid := repository.IsValidAfterGenerate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterGenerate, got: %v, want: %v", isValid, false)
		}
	})
}
