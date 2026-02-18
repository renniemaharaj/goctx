package patch

import (
	"strings"
	"testing"
)

func TestParseNative_SingleFile(t *testing.T) {
	input := `
"internal/ui/ui.go":
<<<<<< SEARCH
old_code()
======
new_code()
>>>>>> REPLACE
`
	output, ok := ParseNative(input)
	if !ok {
		t.Fatal("Failed to parse native dialect")
	}

	if len(output.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(output.Files))
	}

	content, exists := output.Files["internal/ui/ui.go"]
	if !exists {
		t.Fatal("File path not found in output")
	}

	if !strings.Contains(content, "SEARCH") || !strings.Contains(content, "REPLACE") {
		t.Errorf("Content does not appear to contain markers: %s", content)
	}
}

func TestParseNative_MultipleFiles(t *testing.T) {
	input := `
"file1.go":
<<<<<< SEARCH
A
======
B
>>>>>> REPLACE

"file2.go":
<<<<<< SEARCH
C
======
D
>>>>>> REPLACE
`
	output, ok := ParseNative(input)
	if !ok || len(output.Files) != 2 {
		t.Fatalf("Failed to parse multiple files, got %d", len(output.Files))
	}
}
