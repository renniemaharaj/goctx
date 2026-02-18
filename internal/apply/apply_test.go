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

func TestApplyHunksToString_Sequential(t *testing.T) {
	// Test that the output of one hunk can be the search target for the next
	original := "start"
	hunks := []patch.Hunk{
		{Search: "start", Replace: "middle"},
		{Search: "middle", Replace: "end"},
	}

	expected := "end"
	result, err := ApplyHunksToString(original, hunks)
	if err != nil {
		t.Fatalf("Sequential apply failed: %v", err)
	}
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestApplyHunksToString_Overlapping(t *testing.T) {
	// Test that multiple hunks targeting different parts of the same file work
	original := "line A\nline B\nline C"
	hunks := []patch.Hunk{
		{Search: "line A", Replace: "Alpha"},
		{Search: "line C", Replace: "Gamma"},
	}

	expected := "Alpha\nline B\nGamma"
	result, err := ApplyHunksToString(original, hunks)
	if err != nil {
		t.Fatalf("Overlapping apply failed: %v", err)
	}
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
