package cli

import (
	"reflect"
	"testing"

	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
)

func TestCanonicalRepositoryName(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"token", "token"},
		{"Token", "token"},
		{"tokyonight.nvim", "tokyonight"},
		{"gruvbox.nvim", "gruvbox"},
		{"Gruvbox.Nvim", "gruvbox"},
		{"foo-nvim", "foo"},
		{"foo_nvim", "foo"},
		{"foo.vim", "foo"},
		{"foo-vim", "foo"},
		{"nvim-tokyonight", "tokyonight"},
		{"vim-gruvbox", "gruvbox"},
		{"foo-theme", "foo"},
		{"foo-colorscheme", "foo"},
		{"foo-colorschemes", "foo"},
		{"foo-colourscheme", "foo"},
		{"foo-nvim-theme", "foo"},
		{"nvim", "nvim"},
		{"neovim", "neovim"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			if got := canonicalRepositoryName(tt.repo); got != tt.want {
				t.Fatalf("canonicalRepositoryName(%q) = %q, want %q", tt.repo, got, tt.want)
			}
		})
	}
}

func TestOrderedColorschemeNames(t *testing.T) {
	t.Run("exact match goes first", func(t *testing.T) {
		names := []string{"token-ultradark", "token", "token-light"}
		orderedColorschemeNames(names, "token")
		want := []string{"token", "token-light", "token-ultradark"}

		if !reflect.DeepEqual(names, want) {
			t.Fatalf("ordered = %v, want %v", names, want)
		}
	})

	t.Run("strips nvim suffix for the match", func(t *testing.T) {
		names := []string{"tokyonight-storm", "tokyonight", "tokyonight-day"}
		orderedColorschemeNames(names, "tokyonight.nvim")
		want := []string{"tokyonight", "tokyonight-day", "tokyonight-storm"}

		if !reflect.DeepEqual(names, want) {
			t.Fatalf("ordered = %v, want %v", names, want)
		}
	})

	t.Run("match is case-insensitive", func(t *testing.T) {
		names := []string{"gruvbox-baby", "Gruvbox", "gruvbox-material"}
		orderedColorschemeNames(names, "Gruvbox.nvim")
		if names[0] != "Gruvbox" {
			t.Fatalf("ordered[0] = %q, want %q", names[0], "Gruvbox")
		}
	})

	t.Run("no match falls back to alphabetical", func(t *testing.T) {
		names := []string{"nord", "gruvbox", "apprentice"}
		orderedColorschemeNames(names, "nvim")
		want := []string{"apprentice", "gruvbox", "nord"}

		if !reflect.DeepEqual(names, want) {
			t.Fatalf("ordered = %v, want %v", names, want)
		}
	})

	t.Run("does not promote a suffixed scheme", func(t *testing.T) {
		names := []string{"foo-theme", "bar", "aaa"}
		orderedColorschemeNames(names, "foo")
		want := []string{"aaa", "bar", "foo-theme"}

		if !reflect.DeepEqual(names, want) {
			t.Fatalf("ordered = %v, want %v", names, want)
		}
	})
}

func TestBuildRepositoryColorschemes(t *testing.T) {
	data := map[string]repoHelper.ColorschemeData{
		"token-ultradark": {Dark: []repoHelper.ColorschemeGroup{{Name: "Normal"}}},
		"token":           {Dark: []repoHelper.ColorschemeGroup{{Name: "Normal"}}},
		"token-light":     {Light: []repoHelper.ColorschemeGroup{{Name: "Normal"}}},
	}

	t.Run("main scheme first with backgrounds", func(t *testing.T) {
		got := buildRepositoryColorschemes(data, "token")

		wantOrder := []string{"token", "token-light", "token-ultradark"}
		gotOrder := []string{got[0].Name, got[1].Name, got[2].Name}
		if !reflect.DeepEqual(gotOrder, wantOrder) {
			t.Fatalf("order = %v, want %v", gotOrder, wantOrder)
		}

		if len(got[0].Backgrounds) != 1 || got[0].Backgrounds[0] != repoHelper.DarkBackground {
			t.Fatalf("token backgrounds = %v, want [dark]", got[0].Backgrounds)
		}
		if len(got[1].Backgrounds) != 1 || got[1].Backgrounds[0] != repoHelper.LightBackground {
			t.Fatalf("token-light backgrounds = %v, want [light]", got[1].Backgrounds)
		}
	})

	t.Run("deterministic across runs", func(t *testing.T) {
		first := buildRepositoryColorschemes(data, "token")
		for range 20 {
			got := buildRepositoryColorschemes(data, "token")
			for i := range first {
				if got[i].Name != first[i].Name {
					t.Fatalf("nondeterministic order: %v vs %v", got, first)
				}
			}
		}
	})
}
