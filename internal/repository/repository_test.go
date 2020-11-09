package repository

import (
	"testing"

	gogithub "github.com/google/go-github/v32/github"
)

func TestUniquifyRepositoriesSingle(t *testing.T) {
	id := int64(12345)
	list := []*gogithub.Repository{{ID: &id}}

	unique := UniquifyRepositories(list)

	if len(unique) != 1 {
		t.Errorf("Incorrect result for UniquifyRepositories, got length: %d, want length: %d", len(unique), 1)
	}
}

func TestUniquifyRepositoriesDuplicate(t *testing.T) {
	id1 := int64(12345)
	id2 := int64(12345)
	list := []*gogithub.Repository{{ID: &id1}, {ID: &id2}}

	unique := UniquifyRepositories(list)

	if len(unique) != 1 {
		t.Errorf("Incorrect result for UniquifyRepositories, got length: %d, want length: %d", len(unique), 1)
	}
}

func TestUniquifyRepositoriesDuplicateAndMultiple(t *testing.T) {
	id1 := int64(12345)
	id2 := int64(12345)
	id3 := int64(12346)
	list := []*gogithub.Repository{{ID: &id1}, {ID: &id2}, {ID: &id3}}

	unique := UniquifyRepositories(list)

	if len(unique) != 2 {
		t.Errorf("Incorrect result for UniquifyRepositories, got length: %d, want length: %d", len(unique), 2)
	}
}
