package github

import (
	"testing"

	gogithub "github.com/google/go-github/v68/github"
)

func tree(truncated bool, paths ...string) *gogithub.Tree {
	entries := make([]*gogithub.TreeEntry, 0, len(paths))
	for _, path := range paths {
		entries = append(entries, &gogithub.TreeEntry{Path: gogithub.Ptr(path)})
	}
	return &gogithub.Tree{Entries: entries, Truncated: gogithub.Ptr(truncated)}
}

func TestCountColorschemeFiles(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  int
	}{
		{
			name:  "counts vim and lua colorscheme files",
			paths: []string{"colors/one.vim", "colors/two.lua"},
			want:  2,
		},
		{
			name:  "counts case-differed files, which load on case-insensitive systems",
			paths: []string{"Colors/One.vim", "colors/two.VIM", "COLORS/three.lua"},
			want:  3,
		},
		{
			name:  "counts the after/colors override path",
			paths: []string{"after/colors/late.vim"},
			want:  1,
		},
		{
			name:  "ignores nested and non-root after/colors files",
			paths: []string{"after/colors/nested/one.vim", "lua/after/colors/one.lua"},
			want:  0,
		},
		{
			name:  "ignores nested colorscheme files, which vim never loads",
			paths: []string{"colors/light/one.vim", "colors/dark/two.lua"},
			want:  0,
		},
		{
			name:  "ignores other extensions under colors",
			paths: []string{"colors/README.md", "colors/palette.json", "colors/one.txt"},
			want:  0,
		},
		{
			name:  "ignores the colors directory entry itself",
			paths: []string{"colors"},
			want:  0,
		},
		{
			name:  "ignores colors directories that are not at the root",
			paths: []string{"lua/colors/one.lua", "notcolors/one.vim", "extra/colors/two.vim"},
			want:  0,
		},
		{
			name:  "ignores plugin and autoload files",
			paths: []string{"plugin/one.vim", "autoload/two.lua", "lua/theme/init.lua"},
			want:  0,
		},
		{
			name:  "counts only the colorscheme files in a mixed tree",
			paths: []string{"README.md", "colors/one.vim", "lua/theme/init.lua", "colors/two.vim"},
			want:  2,
		},
		{
			name:  "returns zero for an empty tree",
			paths: nil,
			want:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := countColorschemeFiles(tree(false, test.paths...))
			if got != test.want {
				t.Fatalf("countColorschemeFiles() = %d, want %d", got, test.want)
			}
		})
	}
}
