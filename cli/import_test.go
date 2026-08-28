package cli

import (
	"errors"
	"testing"

	gogithub "github.com/google/go-github/v68/github"
	"github.com/vimcolorschemes/worker/internal/github"
)

func githubRepository(id int64, ownerName string, name string) *gogithub.Repository {
	return &gogithub.Repository{
		ID:    gogithub.Ptr(id),
		Name:  gogithub.Ptr(name),
		Owner: &gogithub.User{Login: gogithub.Ptr(ownerName)},
	}
}

func repositoryNames(repositories []*gogithub.Repository) []string {
	names := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		names = append(names, repository.GetName())
	}
	return names
}

func TestFilterColorschemeRepositories(t *testing.T) {
	originalCountColorschemeFiles := countColorschemeFiles
	originalGetRepositoryIDs := getRepositoryIDs
	t.Cleanup(func() {
		countColorschemeFiles = originalCountColorschemeFiles
		getRepositoryIDs = originalGetRepositoryIDs
	})

	t.Run("keeps a known repository without calling Github", func(t *testing.T) {
		getRepositoryIDs = func() (map[int64]bool, error) { return map[int64]bool{1: true}, nil }
		callCount := 0
		countColorschemeFiles = func(ownerName string, name string) (int, error) {
			callCount++
			return 0, nil
		}

		kept, checkedCount, droppedNames := filterColorschemeRepositories([]*gogithub.Repository{
			githubRepository(1, "owner", "known"),
		})

		if callCount != 0 {
			t.Fatalf("callCount = %d, want 0", callCount)
		}
		if checkedCount != 0 {
			t.Fatalf("checkedCount = %d, want 0", checkedCount)
		}
		if len(kept) != 1 {
			t.Fatalf("kept = %v, want [known]", repositoryNames(kept))
		}
		if len(droppedNames) != 0 {
			t.Fatalf("droppedNames = %v, want []", droppedNames)
		}
	})

	t.Run("drops a new repository with no colorscheme file", func(t *testing.T) {
		getRepositoryIDs = func() (map[int64]bool, error) { return map[int64]bool{}, nil }
		countColorschemeFiles = func(ownerName string, name string) (int, error) { return 0, nil }

		kept, checkedCount, droppedNames := filterColorschemeRepositories([]*gogithub.Repository{
			githubRepository(1, "owner", "dotfiles"),
		})

		if checkedCount != 1 {
			t.Fatalf("checkedCount = %d, want 1", checkedCount)
		}
		if len(kept) != 0 {
			t.Fatalf("kept = %v, want []", repositoryNames(kept))
		}
		if len(droppedNames) != 1 || droppedNames[0] != "owner/dotfiles" {
			t.Fatalf("droppedNames = %v, want [owner/dotfiles]", droppedNames)
		}
	})

	t.Run("keeps a new repository shipping colorscheme files", func(t *testing.T) {
		getRepositoryIDs = func() (map[int64]bool, error) { return map[int64]bool{}, nil }
		countColorschemeFiles = func(ownerName string, name string) (int, error) { return 5, nil }

		kept, checkedCount, droppedNames := filterColorschemeRepositories([]*gogithub.Repository{
			githubRepository(1, "folke", "tokyonight.nvim"),
		})

		if checkedCount != 1 {
			t.Fatalf("checkedCount = %d, want 1", checkedCount)
		}
		if len(kept) != 1 {
			t.Fatalf("kept = %v, want [tokyonight.nvim]", repositoryNames(kept))
		}
		if len(droppedNames) != 0 {
			t.Fatalf("droppedNames = %v, want []", droppedNames)
		}
	})

	t.Run("keeps a new repository when the check fails", func(t *testing.T) {
		getRepositoryIDs = func() (map[int64]bool, error) { return map[int64]bool{}, nil }
		countColorschemeFiles = func(ownerName string, name string) (int, error) {
			return 0, errors.New("boom")
		}

		kept, _, droppedNames := filterColorschemeRepositories([]*gogithub.Repository{
			githubRepository(1, "owner", "flaky"),
		})

		if len(kept) != 1 {
			t.Fatalf("kept = %v, want [flaky]", repositoryNames(kept))
		}
		if len(droppedNames) != 0 {
			t.Fatalf("droppedNames = %v, want []", droppedNames)
		}
	})

	t.Run("keeps a new repository when the tree was truncated", func(t *testing.T) {
		getRepositoryIDs = func() (map[int64]bool, error) { return map[int64]bool{}, nil }
		countColorschemeFiles = func(ownerName string, name string) (int, error) {
			return 0, github.ErrTreeTruncated
		}

		kept, _, droppedNames := filterColorschemeRepositories([]*gogithub.Repository{
			githubRepository(1, "owner", "huge"),
		})

		if len(kept) != 1 {
			t.Fatalf("kept = %v, want [huge]", repositoryNames(kept))
		}
		if len(droppedNames) != 0 {
			t.Fatalf("droppedNames = %v, want []", droppedNames)
		}
	})

	t.Run("only checks the candidates that are new", func(t *testing.T) {
		getRepositoryIDs = func() (map[int64]bool, error) { return map[int64]bool{1: true, 3: true}, nil }
		checked := []string{}
		countColorschemeFiles = func(ownerName string, name string) (int, error) {
			checked = append(checked, name)
			if name == "new-colorscheme" {
				return 1, nil
			}
			return 0, nil
		}

		kept, checkedCount, droppedNames := filterColorschemeRepositories([]*gogithub.Repository{
			githubRepository(1, "owner", "known-one"),
			githubRepository(2, "owner", "new-colorscheme"),
			githubRepository(3, "owner", "known-two"),
			githubRepository(4, "owner", "new-dotfiles"),
		})

		if len(checked) != 2 || checked[0] != "new-colorscheme" || checked[1] != "new-dotfiles" {
			t.Fatalf("checked = %v, want [new-colorscheme new-dotfiles]", checked)
		}
		if checkedCount != 2 {
			t.Fatalf("checkedCount = %d, want 2", checkedCount)
		}
		if got := repositoryNames(kept); len(got) != 3 {
			t.Fatalf("kept = %v, want 3 repositories", got)
		}
		if len(droppedNames) != 1 || droppedNames[0] != "owner/new-dotfiles" {
			t.Fatalf("droppedNames = %v, want [owner/new-dotfiles]", droppedNames)
		}
	})
}

func TestSampleDroppedNames(t *testing.T) {
	t.Run("returns every name when under the sample size", func(t *testing.T) {
		names := []string{"a/one", "a/two"}
		if got := sampleDroppedNames(names); len(got) != 2 {
			t.Fatalf("sampleDroppedNames() = %v, want %v", got, names)
		}
	})

	t.Run("caps the names at the sample size", func(t *testing.T) {
		names := []string{"a/one", "a/two", "a/three", "a/four", "a/five", "a/six"}
		if got := sampleDroppedNames(names); len(got) != repositoryDroppedNameSampleSize {
			t.Fatalf("len(sampleDroppedNames()) = %d, want %d", len(got), repositoryDroppedNameSampleSize)
		}
	})
}
