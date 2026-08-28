package github

import "testing"

func TestMatchesColorschemePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"colors/foo.vim", true},
		{"colors/foo.lua", true},
		{"colors/nested/foo.vim", true},
		{"colors/README.md", false},
		{"lua/foo/colors.lua", false},
		{"autoload/colors/foo.vim", false},
		{"colors/foo.vim.bak", false},
		{"colors", false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got := matchesColorschemePath(test.path); got != test.want {
				t.Fatalf("matchesColorschemePath(%q) = %t, want %t", test.path, got, test.want)
			}
		})
	}
}
