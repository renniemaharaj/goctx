package builder

import "testing"

func TestMatchesIgnore(t *testing.T) {
	patterns := []string{".git", "node_modules/", "*.exe", "dist/bin"}

	tests := []struct {
		path     string
		expected bool
	}{
		{".git/config", true},
		{"node_modules/package.json", true},
		{"src/main.go", false},
		{"main.exe", true},
		{"dist/bin/app", true},
		{"internal/builder/ignore.go", false},
	}

	for _, tt := range tests {
		got := MatchesIgnore(tt.path, patterns)
		if got != tt.expected {
			t.Errorf("MatchesIgnore(%q) = %v; want %v", tt.path, got, tt.expected)
		}
	}
}