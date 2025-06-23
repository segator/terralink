package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoreMatcher(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a test .terralinkignore file
	ignoreContent := `
node_modules
.git
dist
`
	if err := os.WriteFile(filepath.Join(tempDir, ".terralinkignore"), []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write test .terralinkignore file: %v", err)
	}

	// Create the matcher
	matcher, err := NewMatcher(tempDir)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Test cases
	tests := []struct {
		path     string
		expected bool
	}{
		{filepath.Join(tempDir, ".terraform", "file.tf"), true},           // Default pattern
		{filepath.Join(tempDir, "subdir", ".terraform", "file.tf"), true}, // Default pattern in subdir
		{filepath.Join(tempDir, "node_modules"), true},                    // Pattern from file
		{filepath.Join(tempDir, "app", "node_modules"), true},             // Pattern from file in subdir
		{filepath.Join(tempDir, ".git"), true},                            // Pattern from file
		{filepath.Join(tempDir, "dist", "file.tf"), true},                 // Pattern from file
		{filepath.Join(tempDir, "app", "file.tf"), false},                 // *.tf files not ignored
		{filepath.Join(tempDir, "app", "mo2", "file4.hcl"), false},        // *.hcl files not ignored
		{filepath.Join(tempDir, "src", "file.tf"), false},                 // Not in ignore list
		{filepath.Join(tempDir, "app", "src", "file.hcl"), false},         // Not in ignore list
	}

	for _, tc := range tests {
		if got := matcher.ShouldIgnore(tc.path); got != tc.expected {
			t.Errorf("ShouldIgnore(%q) = %v; want %v", tc.path, got, tc.expected)
		}
	}
}

func TestEmptyIgnoreFile(t *testing.T) {
	tempDir := t.TempDir()

	// No .terralinkignore file created, should only use defaults
	matcher, err := NewMatcher(tempDir)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Should still ignore .terraform (default)
	if !matcher.ShouldIgnore(filepath.Join(tempDir, ".terraform")) {
		t.Error("Expected .terraform to be ignored by default")
	}

	// Should not ignore other directories
	if matcher.ShouldIgnore(filepath.Join(tempDir, "src/file.tf")) {
		t.Error("Did not expect src to be ignored")
	}
}
