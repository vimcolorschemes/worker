package cli

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
)

func TestIsDefaultColorscheme(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"default", true},
		{"habamax", true},
		{"slate", true},
		{"zaibatsu", true},
		// real colorscheme names must NOT be filtered
		{"gruvbox", false},
		{"nord", false},
		{"tokyonight", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDefaultColorscheme(tt.name)
			if got != tt.want {
				t.Fatalf("isDefaultColorscheme(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRepositoryColorschemes(t *testing.T) {
	defaults := map[string]bool{"blue": true, "desert": true}

	t.Run("should drop the colorschemes present before installation", func(t *testing.T) {
		got := repositoryColorschemes([]string{"gruvbox", "blue", "nord", "desert"}, defaults)
		want := []string{"gruvbox", "nord"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("repositoryColorschemes() = %v, want %v", got, want)
		}
	})

	t.Run("should drop the built-in colorschemes", func(t *testing.T) {
		got := repositoryColorschemes([]string{"habamax", "nord", "zaibatsu", ""}, defaults)
		want := []string{"nord"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("repositoryColorschemes() = %v, want %v", got, want)
		}
	})

	t.Run("should sort and deduplicate", func(t *testing.T) {
		got := repositoryColorschemes([]string{"nord", "gruvbox", "nord", "apprentice"}, defaults)
		want := []string{"apprentice", "gruvbox", "nord"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("repositoryColorschemes() = %v, want %v", got, want)
		}
	})

	t.Run("should return an empty list when nothing is left", func(t *testing.T) {
		got := repositoryColorschemes([]string{"blue", "desert"}, defaults)

		if len(got) != 0 {
			t.Fatalf("repositoryColorschemes() = %v, want an empty list", got)
		}
	})
}

func TestChunkColorschemes(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		size  int
		want  [][]string
	}{
		{
			name:  "exact multiple",
			names: []string{"a", "b", "c", "d"},
			size:  2,
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "trailing remainder",
			names: []string{"a", "b", "c"},
			size:  2,
			want:  [][]string{{"a", "b"}, {"c"}},
		},
		{
			name:  "smaller than a batch",
			names: []string{"a"},
			size:  50,
			want:  [][]string{{"a"}},
		},
		{
			name:  "empty",
			names: []string{},
			size:  50,
			want:  [][]string{},
		},
		{
			name:  "one at a time",
			names: []string{"a", "b"},
			size:  1,
			want:  [][]string{{"a"}, {"b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunkColorschemes(tt.names, tt.size)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("chunkColorschemes(%v, %d) = %v, want %v", tt.names, tt.size, got, tt.want)
			}
		})
	}

	t.Run("should never build empty batches", func(t *testing.T) {
		got := chunkColorschemes([]string{"a", "b"}, 0)
		want := [][]string{{"a"}, {"b"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("chunkColorschemes() = %v, want %v", got, want)
		}
	})
}

func TestParseColorData(t *testing.T) {
	t.Run("should decode an extraction result", func(t *testing.T) {
		got, err := parseColorData([]byte(`{"nord":{"dark":[{"name":"NormalFg","hexCode":"#D8DEE9"}]}}`))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		want := map[string]repoHelper.ColorschemeData{
			"nord": {Dark: []repoHelper.ColorschemeGroup{{Name: "NormalFg", HexCode: "#D8DEE9"}}},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("parseColorData() = %v, want %v", got, want)
		}
	})

	t.Run("should read an empty array as no colorscheme", func(t *testing.T) {
		got, err := parseColorData([]byte("[]\n"))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		if len(got) != 0 {
			t.Fatalf("parseColorData() = %v, want an empty map", got)
		}
	})

	t.Run("should read empty content as no colorscheme", func(t *testing.T) {
		got, err := parseColorData([]byte("  "))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		if len(got) != 0 {
			t.Fatalf("parseColorData() = %v, want an empty map", got)
		}
	})

	t.Run("should reject a non-empty array", func(t *testing.T) {
		if _, err := parseColorData([]byte(`["nord"]`)); err == nil {
			t.Fatal("Expected an error, got none")
		}
	})

	t.Run("should reject malformed content", func(t *testing.T) {
		if _, err := parseColorData([]byte("not json")); err == nil {
			t.Fatal("Expected an error, got none")
		}
	})
}

func TestCaptureColorschemeNamesCommand(t *testing.T) {
	command := captureColorschemeNamesCommand("/tmp/colorschemes.json")

	for _, fragment := range []string{
		"pcall(require('extractor').colorschemes",
		"io.stderr:write(tostring(err))",
		"vim.cmd('cquit 1')",
		"vim.cmd('qa!')",
	} {
		if !strings.Contains(command, fragment) {
			t.Errorf("captureColorschemeNamesCommand() missing %q in %q", fragment, command)
		}
	}
}

func TestRemoveIfExists(t *testing.T) {
	path := t.TempDir() + "/colorschemes.json"
	if err := os.WriteFile(path, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := removeIfExists(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("removeIfExists() left %q behind, stat error: %v", path, err)
	}

	if err := removeIfExists(path); err != nil {
		t.Fatalf("removeIfExists() returned an error for a missing file: %v", err)
	}
}

func TestMergeColorData(t *testing.T) {
	t.Run("should merge disjoint batches", func(t *testing.T) {
		target := map[string]repoHelper.ColorschemeData{
			"nord": {Dark: []repoHelper.ColorschemeGroup{{Name: "NormalFg"}}},
		}
		source := map[string]repoHelper.ColorschemeData{
			"gruvbox": {Light: []repoHelper.ColorschemeGroup{{Name: "NormalBg"}}},
		}

		mergeColorData(target, source)

		if len(target) != 2 {
			t.Fatalf("mergeColorData() left %d colorschemes, want 2", len(target))
		}
		if target["gruvbox"].Light == nil {
			t.Error("Missing merged colorscheme data")
		}
	})

	t.Run("should keep the data of an earlier batch", func(t *testing.T) {
		target := map[string]repoHelper.ColorschemeData{
			"nord": {Dark: []repoHelper.ColorschemeGroup{{Name: "NormalFg"}}},
		}
		source := map[string]repoHelper.ColorschemeData{"nord": {}}

		mergeColorData(target, source)

		if len(target["nord"].Dark) != 1 {
			t.Fatalf("mergeColorData() overwrote an earlier batch, got: %v", target["nord"])
		}
	})
}

func TestClassifyGenerateResult(t *testing.T) {
	tests := []struct {
		name              string
		colorschemeCount  int
		shipsColorschemes bool
		checkError        error
		want              generateOutcome
		wantChecked       bool
	}{
		{
			name:             "writes without checking when colorschemes were generated",
			colorschemeCount: 3,
			want:             generateOutcomeWrite,
		},
		{
			name:              "errors on zero colorschemes when the repository ships colors files",
			shipsColorschemes: true,
			want:              generateOutcomeError,
			wantChecked:       true,
		},
		{
			name:        "reports empty on zero colorschemes when the repository ships none",
			want:        generateOutcomeEmpty,
			wantChecked: true,
		},
		{
			name:        "errors when the colors check itself fails",
			checkError:  errors.New("boom"),
			want:        generateOutcomeError,
			wantChecked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := hasColorschemeFiles
			t.Cleanup(func() { hasColorschemeFiles = original })

			checked := false
			hasColorschemeFiles = func(ownerName string, name string) (bool, error) {
				checked = true
				if ownerName != "owner" || name != "repo" {
					t.Fatalf("hasColorschemeFiles(%q, %q), want (\"owner\", \"repo\")", ownerName, name)
				}
				return tt.shipsColorschemes, tt.checkError
			}

			got := classifyGenerateResult("owner", "repo", tt.colorschemeCount)
			if got != tt.want {
				t.Fatalf("classifyGenerateResult() = %v, want %v", got, tt.want)
			}
			if checked != tt.wantChecked {
				t.Fatalf("colors check called = %v, want %v", checked, tt.wantChecked)
			}
		})
	}
}
