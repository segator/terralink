package linker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Constants and regex for annotation parsing.
const (
	stateAnnotationPrefix = "# terralink-state:"
)

var (
	// Regex to parse a terralink dev comment, e.g., "# terralink: path=../local"
	devAnnotationLineRegex = regexp.MustCompile(`^#\s*terralink:\s*(.*)$`)
	devAnnotationPairRegex = regexp.MustCompile(`(\w+)\s*=\s*([^\s]+)`)

	// Regex to parse key-value pairs from state annotations.
	stateAttrRegex = regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"`)
)

// StateAnnotation holds the original source and version of a module.
type StateAnnotation struct {
	Source  string
	Version string
}

// findDevAnnotation searches through a block's comments to find a dev annotation.
func findDevAnnotation(block *hclwrite.Block) (path string, found bool) {
	for _, token := range block.Body().BuildTokens(nil) {
		if token.Type == hclsyntax.TokenComment {
			if path, isDev := parseDevAnnotation(string(token.Bytes)); isDev {
				return path, true
			}
		}
	}
	return "", false
}

// findStateAnnotation searches for a state annotation comment in a block and parses it.
func findStateAnnotation(block *hclwrite.Block) (StateAnnotation, bool) {
	for _, token := range block.Body().BuildTokens(nil) {
		if token.Type == hclsyntax.TokenComment {
			if state, isState := parseStateAnnotation(string(token.Bytes)); isState {
				return state, true
			}
		}
	}
	return StateAnnotation{}, false
}

// parseDevAnnotation checks if a comment is a dev annotation and extracts the path.
func parseDevAnnotation(comment string) (path string, isDev bool) {
	lineMatch := devAnnotationLineRegex.FindStringSubmatch(strings.TrimSpace(comment))
	if len(lineMatch) != 2 {
		return "", false
	}
	attrsStr := lineMatch[1]
	pairMatches := devAnnotationPairRegex.FindAllStringSubmatch(attrsStr, -1)
	for _, pair := range pairMatches {
		if len(pair) == 3 && pair[1] == "path" {
			return pair[2], true
		}
	}
	return "", false
}

// parseStateAnnotation extracts the source and version from a state line.
func parseStateAnnotation(line string) (StateAnnotation, bool) {
	state := StateAnnotation{}
	if !strings.HasPrefix(strings.TrimSpace(line), stateAnnotationPrefix) {
		return state, false
	}

	data := strings.TrimSpace(strings.TrimPrefix(line, stateAnnotationPrefix))
	matches := stateAttrRegex.FindAllStringSubmatch(data, -1)
	if matches == nil {
		return state, false
	}

	found := false
	for _, match := range matches {
		key, value := match[1], match[2]
		switch key {
		case "source":
			state.Source = value
			found = true
		case "version":
			state.Version = value
		}
	}
	return state, found
}

// buildStateAnnotation constructs the string for a state annotation comment.
func buildStateAnnotation(source, version string) string {
	versionPart := ""
	if version != "" {
		versionPart = fmt.Sprintf(` version="%s"`, version)
	}
	return fmt.Sprintf(`%s source="%s"%s`, stateAnnotationPrefix, source, versionPart)
}

// getAttrValueAsString safely extracts the string value from an HCL attribute.
func getAttrValueAsString(attr *hclwrite.Attribute) string {
	if attr == nil {
		return ""
	}
	// Iterate through the expression tokens to find the literal string.
	// This is more robust than building a raw string and trimming it.
	for _, token := range attr.Expr().BuildTokens(nil) {
		if token.Type == hclsyntax.TokenQuotedLit {
			// The Bytes of a quoted literal include the quotes, so we must trim them.
			return strings.Trim(string(token.Bytes), `"`)
		}
	}
	return "" // Return empty if no string literal is found
}
