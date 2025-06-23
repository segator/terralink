package ignore

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Default patterns to ignore
var defaultPatterns = []string{
	".terraform",
}

// IgnoreMatcher handles path matching against ignore patterns
type IgnoreMatcher struct {
	patterns []string
}

// NewMatcher creates a new IgnoreMatcher with default patterns and
// loads additional patterns from .terralinkignore if it exists
func NewMatcher(rootDir string) (*IgnoreMatcher, error) {
	matcher := &IgnoreMatcher{
		patterns: append([]string{}, defaultPatterns...),
	}

	// Try to load .terralinkignore file
	ignoreFile := filepath.Join(rootDir, ".terralinkignore")
	if _, err := os.Stat(ignoreFile); err == nil {
		if err := matcher.loadIgnoreFile(ignoreFile); err != nil {
			return nil, err
		}
	}

	return matcher, nil
}

// loadIgnoreFile loads patterns from a .terralinkignore file
func (m *IgnoreMatcher) loadIgnoreFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Panic(err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		pattern := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if pattern != "" && !strings.HasPrefix(pattern, "#") {
			m.patterns = append(m.patterns, pattern)
		}
	}

	return scanner.Err()
}

// ShouldIgnore checks if a path should be ignored based on the patterns
func (m *IgnoreMatcher) ShouldIgnore(path string) bool {
	// Normalize path for consistent matching
	cleanPath := filepath.ToSlash(path)
	base := filepath.Base(cleanPath)

	for _, pattern := range m.patterns {

		if base == pattern {
			return true
		}

		if strings.Contains(cleanPath, "/"+pattern+"/") {
			return true
		}

		if strings.HasSuffix(cleanPath, "/"+pattern) {
			return true
		}
		if !strings.HasSuffix(base, ".tf") && !strings.HasSuffix(base, ".hcl") {
			return true
		}
	}
	return false
}
