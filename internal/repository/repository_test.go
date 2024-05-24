package repository

import (
	"reflect"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v32/github"
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
			{Date: repository.GitHubCreatedAt, StargazersCount: 0},
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
		repository.VimColorSchemes = []VimColorScheme{{Name: "test", FileURL: "http://vim.org"}}

		isValid := repository.IsValidAfterUpdate()
		if !isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, true)
		}
	})

	t.Run("should return invalid for a repository with no last commit at", func(t *testing.T) {
		var repository Repository
		repository.StargazersCount = 1
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: dateUtil.Today(), StargazersCount: 1}}
		repository.VimColorSchemes = []VimColorScheme{{Name: "test", FileURL: "http://vim.org"}}

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
		repository.VimColorSchemes = []VimColorScheme{{Name: "test", FileURL: "http://vim.org"}}

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
		repository.VimColorSchemes = []VimColorScheme{{Name: "test", FileURL: "http://vim.org"}}

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
		repository.VimColorSchemes = []VimColorScheme{{Name: "test", FileURL: "http://vim.org"}}

		isValid := repository.IsValidAfterUpdate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, false)
		}
	})

	t.Run("should return invalid for a repository with an empty vim color scheme name list", func(t *testing.T) {
		var repository Repository
		repository.LastCommitAt = time.Now()
		repository.StargazersCount = 1
		repository.StargazersCountHistory = []StargazersCountHistoryItem{{Date: dateUtil.Today(), StargazersCount: 1}}
		repository.VimColorSchemes = []VimColorScheme{}

		isValid := repository.IsValidAfterUpdate()
		if isValid {
			t.Errorf("Incorrect result for IsRepositoryValidAfterUpdate, got: %v, want: %v", isValid, false)
		}
	})
}

func TestComputeRepositoryValidityAfterGenerate(t *testing.T) {
	t.Run("should return valid for a repository that checks all boxes", func(t *testing.T) {
		var repository Repository
		repository.VimColorSchemes = []VimColorScheme{{Name: "test", FileURL: "http://vim.org", Valid: true}}
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
		repository.VimColorSchemes = []VimColorScheme{{Name: "test", FileURL: "http://vim.org", Valid: false}}

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

func TestSynchronizeVimColorSchemes(t *testing.T) {
	t.Run("should erase the original list given an empty list", func(t *testing.T) {
		originalList := []VimColorScheme{{Name: "test"}}
		newList := []VimColorScheme{}

		repository := Repository{VimColorSchemes: originalList}

		repository.SyncVimColorSchemes(newList)

		if len(repository.VimColorSchemes) != 0 {
			t.Errorf(
				"Incorrect result for SyncVimColorSchemes, got length: %d, want length: %d",
				len(repository.VimColorSchemes),
				0,
			)
		}
	})

	t.Run("should replace the whole list given a list with different vim color scheme names", func(t *testing.T) {
		originalList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "test.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
		}

		newList := []VimColorScheme{
			{
				Name: "test_2",
			},
		}

		repository := Repository{VimColorSchemes: originalList}

		repository.SyncVimColorSchemes(newList)

		if !reflect.DeepEqual(repository.VimColorSchemes, newList) {
			t.Errorf(
				"Incorrect result for SyncVimColorSchemes, got: %v, want length: %v",
				repository.VimColorSchemes,
				newList,
			)
		}
	})

	t.Run("should update the corresponding list items", func(t *testing.T) {
		originalList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "old.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
		}

		newList := []VimColorScheme{
			{
				Name:    "test",
				FileURL: "new.url",
			},
		}

		expectedList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "new.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
		}

		repository := Repository{VimColorSchemes: originalList}

		repository.SyncVimColorSchemes(newList)

		if !reflect.DeepEqual(repository.VimColorSchemes, expectedList) {
			t.Errorf(
				"Incorrect result for SyncVimColorSchemes, got: %v, want length: %v",
				repository.VimColorSchemes,
				expectedList,
			)
		}
	})

	t.Run("should update the corresponding list items and insert new ones", func(t *testing.T) {
		originalList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "old.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
		}

		newList := []VimColorScheme{
			{
				Name:    "test",
				FileURL: "new.url",
			},
			{
				Name:    "new",
				FileURL: "new.url",
			},
		}

		expectedList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "new.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
			{
				Name:    "new",
				FileURL: "new.url",
			},
		}

		repository := Repository{VimColorSchemes: originalList}

		repository.SyncVimColorSchemes(newList)

		if !reflect.DeepEqual(repository.VimColorSchemes, expectedList) {
			t.Errorf(
				"Incorrect result for SyncVimColorSchemes, got: %v, want length: %v",
				repository.VimColorSchemes,
				expectedList,
			)
		}
	})

	t.Run("should update the corresponding list items and erase old ones", func(t *testing.T) {
		originalList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "old.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
			{
				Name:    "old",
				Valid:   true,
				FileURL: "old.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
		}

		newList := []VimColorScheme{
			{
				Name:    "test",
				FileURL: "new.url",
			},
		}

		expectedList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "new.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
			},
		}

		repository := Repository{VimColorSchemes: originalList}

		repository.SyncVimColorSchemes(newList)

		if !reflect.DeepEqual(repository.VimColorSchemes, expectedList) {
			t.Errorf(
				"Incorrect result for SyncVimColorSchemes, got: %v, want length: %v",
				repository.VimColorSchemes,
				expectedList,
			)
		}
	})

	t.Run("should update the corresponding list items, insert new ones and erase old ones", func(t *testing.T) {
		time := time.Now()
		originalList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "old.test.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
				IsLua:        false,
				LastCommitAt: time,
			},
			{
				Name:    "test.old",
				Valid:   true,
				FileURL: "old.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
				IsLua:        false,
				LastCommitAt: time,
			},
		}

		newList := []VimColorScheme{
			{
				Name:         "test",
				FileURL:      "new.test.url",
				IsLua:        false,
				LastCommitAt: time,
			},
			{
				Name:         "test.new",
				FileURL:      "new.url",
				IsLua:        false,
				LastCommitAt: time,
			},
		}

		expectedList := []VimColorScheme{
			{
				Name:    "test",
				Valid:   true,
				FileURL: "new.test.url",
				Data: VimColorSchemeData{
					Dark: []VimColorSchemeGroup{
						{HexCode: "#000000", Name: "NormalBg"},
					},
					Light: nil,
				},
				IsLua:        false,
				LastCommitAt: time,
			},
			{
				Name:         "test.new",
				FileURL:      "new.url",
				IsLua:        false,
				LastCommitAt: time,
			},
		}

		repository := Repository{VimColorSchemes: originalList}

		repository.SyncVimColorSchemes(newList)

		if !reflect.DeepEqual(repository.VimColorSchemes, expectedList) {
			t.Errorf(
				"Incorrect result for SyncVimColorSchemes, got: %v, want: %v",
				repository.VimColorSchemes,
				expectedList,
			)
		}
	})
}

func TestAssignRepositoryType(t *testing.T) {
	t.Run("should set type vim when a vim color scheme exists", func(t *testing.T) {
		vimColorSchemes := []VimColorScheme{
			{
				Name:  "vim",
				IsLua: false,
			},
		}

		repository := Repository{VimColorSchemes: vimColorSchemes}
		repository.AssignRepositoryType()

		if !repository.IsVim {
			t.Error("Incorrect result for AssignRepositoryType, got: IsVim=false, want: true")
		}

		if repository.IsLua {
			t.Error("Incorrect result for AssignRepositoryType, got: IsLua=true, want: false")
		}
	})

	t.Run("should set type lua when a lua color scheme exists", func(t *testing.T) {
		vimColorSchemes := []VimColorScheme{
			{
				Name:  "lua",
				IsLua: true,
			},
		}

		repository := Repository{VimColorSchemes: vimColorSchemes}
		repository.AssignRepositoryType()

		if repository.IsVim {
			t.Error("Incorrect result for AssignRepositoryType, got: IsVim=true, want: false")
		}

		if !repository.IsLua {
			t.Error("Incorrect result for AssignRepositoryType, got: IsLua=false, want: true")
		}
	})

	t.Run("should set both types when vim lua color schemes exist", func(t *testing.T) {
		vimColorSchemes := []VimColorScheme{
			{
				Name:  "vim",
				IsLua: false,
			},
			{
				Name:  "lua",
				IsLua: true,
			},
		}

		repository := Repository{VimColorSchemes: vimColorSchemes}
		repository.AssignRepositoryType()

		if !repository.IsVim {
			t.Error("Incorrect result for AssignRepositoryType, got: IsVim=false, want: true")
		}

		if !repository.IsLua {
			t.Error("Incorrect result for AssignRepositoryType, got: IsLua=false, want: true")
		}
	})
}

func TestUniquifyVimColorSchemes(t *testing.T) {
	t.Run("should return the same list if it has no duplicate name", func(t *testing.T) {
		vimColorSchemes := []VimColorScheme{{Name: "vim"}, {Name: "lua"}}
		uniqueList := uniquifyVimColorSchemes(vimColorSchemes)
		if !reflect.DeepEqual(vimColorSchemes, uniqueList) {
			t.Errorf("Incorrect result for getUniqueVimColorSchemeList, got %d color schemes, expected %d", len(uniqueList), len(vimColorSchemes))
		}
	})

	t.Run("should remove duplicate items keeping most recent LastCommitAt", func(t *testing.T) {
		vimColorSchemes := []VimColorScheme{
			{Name: "vim", LastCommitAt: time.Date(2022, time.Month(2), 21, 1, 0, 0, 0, time.UTC)},
			{Name: "lua", LastCommitAt: time.Date(2023, time.Month(2), 21, 1, 0, 0, 0, time.UTC)},
			{Name: "vim", LastCommitAt: time.Date(2021, time.Month(2), 21, 1, 0, 0, 0, time.UTC)},
		}

		result := uniquifyVimColorSchemes(vimColorSchemes)
		expectedResult := []VimColorScheme{
			{Name: "vim", LastCommitAt: time.Date(2022, time.Month(2), 21, 1, 0, 0, 0, time.UTC)},
			{Name: "lua", LastCommitAt: time.Date(2023, time.Month(2), 21, 1, 0, 0, 0, time.UTC)},
		}

		if !reflect.DeepEqual(result, expectedResult) {
			t.Errorf("Incorrect result for getUniqueVimColorSchemeList, got %d color schemes, expected %d", len(result), len(expectedResult))
		}
	})
}
