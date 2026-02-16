package patch

import (
	"strings"
	"testing"
)

func TestParseHunksBasic(t *testing.T) {
	content := `<<<<<< SEARCH
old line 1
old line 2
======
new line 1
new line 2
>>>>>> REPLACE`
	hunks := ParseHunks(content)
	if len(hunks) != 1 {
		t.Errorf("expected 1 hunk, got %d", len(hunks))
	}
	if hunks[0].Search != "old line 1\nold line 2" {
		t.Errorf("unexpected Search content: %s", hunks[0].Search)
	}
	if hunks[0].Replace != "new line 1\nnew line 2" {
		t.Errorf("unexpected Replace content: %s", hunks[0].Replace)
	}
}

func TestApplyHunkBasic(t *testing.T) {
	original := "line 0\nold line 1\nold line 2\nline 3"
	hunk := Hunk{
		Search:  "old line 1\nold line 2",
		Replace: "new line 1\nnew line 2",
	}
	result, ok := ApplyHunk(original, hunk)
	if !ok {
		t.Fatalf("expected hunk to apply")
	}
	expected := "line 0\nnew line 1\nnew line 2\nline 3"
	if result != expected {
		t.Errorf("unexpected result:\n%s", result)
	}
}

func TestApplyHunkNoMatch(t *testing.T) {
	original := "line a\nline b"
	hunk := Hunk{
		Search:  "not present",
		Replace: "new stuff",
	}
	result, ok := ApplyHunk(original, hunk)
	if ok {
		t.Errorf("expected hunk not to apply")
	}
	if result != original {
		t.Errorf("file content should be unchanged")
	}
}

func TestParseHunksMultiple(t *testing.T) {
	content := strings.Join([]string{
		"<<<<<< SEARCH\nold1\n======\nnew1\n>>>>>> REPLACE",
		"<<<<<< SEARCH\nold2\n======\nnew2\n>>>>>> REPLACE",
	}, "")
	hunks := ParseHunks(content)
	if len(hunks) != 2 {
		t.Errorf("expected 2 hunks, got %d", len(hunks))
	}
	if hunks[1].Replace != "new2" {
		t.Errorf("unexpected second hunk Replace: %s", hunks[1].Replace)
	}
}
