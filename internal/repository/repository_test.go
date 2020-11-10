package repository

import (
	"fmt"
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

func TestGetStargazersCountHistory(t *testing.T) {
	t.Run("should initialize history when it's empty", func(t *testing.T) {
		repository := Repository{
			StargazersCount: 100,
		}
		history := GetStargazersCountHistory(repository)
		today := dateUtil.Today()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for GetStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should add item when there are some already", func(t *testing.T) {
		dateForm := "2020-01-01"
		date1, _ := time.Parse(dateForm, "1970-01-01")

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date1, StargazersCount: 100},
			},
		}

		history := GetStargazersCountHistory(repository)

		today := dateUtil.Today()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date1, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for GetStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should sort items by date (desc)", func(t *testing.T) {
		dateLayout := "2020-01-01"
		date1, _ := time.Parse(dateLayout, "1970-01-01")
		date2, _ := time.Parse(dateLayout, "1970-01-02")
		date3, _ := time.Parse(dateLayout, "1970-01-03")

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date1, StargazersCount: 100},
				{Date: date2, StargazersCount: 100},
				{Date: date3, StargazersCount: 100},
			},
		}

		history := GetStargazersCountHistory(repository)

		today := dateUtil.Today()

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date3, StargazersCount: 100},
			{Date: date2, StargazersCount: 100},
			{Date: date1, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for GetStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should override today's item", func(t *testing.T) {
		dateLayout := "2020-01-01"
		date, _ := time.Parse(dateLayout, "1970-01-03")

		today := dateUtil.Today()

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
			},
		}

		history := GetStargazersCountHistory(repository)

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for GetStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should override all of today's items", func(t *testing.T) {
		dateLayout := "2020-01-01"
		date, _ := time.Parse(dateLayout, "1970-01-03")

		today := dateUtil.Today()

		repository := Repository{
			StargazersCount: 101,
			StargazersCountHistory: []StargazersCountHistoryItem{
				{Date: date, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
				{Date: today, StargazersCount: 100},
			},
		}

		history := GetStargazersCountHistory(repository)

		expected := []StargazersCountHistoryItem{
			{Date: today, StargazersCount: 101},
			{Date: date, StargazersCount: 100},
		}

		if !reflect.DeepEqual(history, expected) {
			t.Error("Incorrect result for GetStargazersCountHistory, did not get expected history")
		}
	})

	t.Run("should keep only 31 days worth of history", func(t *testing.T) {
		dateLayout := "2020-01-01"
		history := []StargazersCountHistoryItem{}

		for i := 1; i <= 31; i++ {
			leadingZero := ""
			if i < 10 {
				leadingZero = "0"
			}
			date, _ := time.Parse(dateLayout, fmt.Sprintf("1970-01-%s%d", leadingZero, i))
			item := StargazersCountHistoryItem{Date: date, StargazersCount: 100}
			history = append(history, item)
		}

		repository := Repository{
			StargazersCount:        101,
			StargazersCountHistory: history,
		}

		result := GetStargazersCountHistory(repository)

		if len(result) != 31 {
			t.Errorf("Incorrect result for GetStargazersCountHistory, got length: %d, want length: %d", len(result), 31)
		}
	})
}
