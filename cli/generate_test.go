package cli

import "testing"

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
