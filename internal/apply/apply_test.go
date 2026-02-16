package apply

import (
	"strings"
	"testing"

	"goctx/internal/patch"
)

func TestApplyHunksToString_Success(t *testing.T) {
	original := "line1\nline2\nline3\n"
	hunks := []patch.Hunk{
		{
			Search:  "line2",
			Replace: "line2_modified",
		},
	}

	expected := "line1\nline2_modified\nline3\n"

	result, err := ApplyHunksToString(original, hunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestApplyHunksToString_MatchFail(t *testing.T) {
	original := "line1\nline2\nline3\n"
	hunks := []patch.Hunk{
		{
			Search:  "line4", // does not exist
			Replace: "line4_modified",
		},
	}

	_, err := ApplyHunksToString(original, hunks)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
}

func TestApplyHunksToString_MultipleHunks(t *testing.T) {
	original := "a\nb\nc\nd\n"
	hunks := []patch.Hunk{
		{Search: "b", Replace: "B"},
		{Search: "d", Replace: "D"},
	}

	expected := "a\nB\nc\nD\n"
	result, err := ApplyHunksToString(original, hunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestSafePath(t *testing.T) {
	root := "/tmp/project"

	tests := []struct {
		path     string
		expected bool
	}{
		{"/tmp/project/file.txt", true},
		{"/tmp/project/sub/file.txt", true},
		{"/tmp/other/file.txt", false},
		{"../project/file.txt", false},
	}

	for _, tt := range tests {
		got := safePath(root, tt.path)
		if got != tt.expected {
			t.Errorf("safePath(%q, %q) = %v; want %v", root, tt.path, got, tt.expected)
		}
	}
}

func TestApplyHunksToString_EmptyHunks(t *testing.T) {
	original := "foo\nbar\n"
	result, err := ApplyHunksToString(original, []patch.Hunk{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != original {
		t.Errorf("expected unchanged string, got %q", result)
	}
}

// optional: test string replacement of multi-line blocks
func TestApplyHunksToString_MultiLine(t *testing.T) {
	original := "alpha\nbeta\ngamma\ndelta\n"
	hunks := []patch.Hunk{
		{
			Search:  "beta\ngamma",
			Replace: "BETA\nGAMMA",
		},
	}

	expected := "alpha\nBETA\nGAMMA\ndelta\n"

	result, err := ApplyHunksToString(original, hunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Compare(result, expected) != 0 {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}
