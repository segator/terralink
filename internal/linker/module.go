package linker

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

// Module represents a single "module" block within a Terraform file.
// It provides methods to inspect and manipulate the module's state.
type Module struct {
	name  string
	block *hclwrite.Block
}

// NewModule creates a new Module instance from a name and an HCL block.
func NewModule(name string, block *hclwrite.Block) *Module {
	return &Module{name: name, block: block}
}

// Name returns the name of the module.
func (m *Module) Name() string {
	return m.name
}

// IsLoaded checks if the module is currently in a "loaded" (dev) state by
// looking for a state annotation.
func (m *Module) IsLoaded() bool {
	_, found := findStateAnnotation(m.block)
	return found
}

// Load activates the development mode for this module by replacing the source
// with a local path and injecting a state annotation to remember the original source.
// It performs low-level token manipulation to ensure comments are placed correctly.
// It returns true if a change was made.
func (m *Module) Load() (bool, error) {
	if m.IsLoaded() {
		return false, nil
	}

	devPath, devAnnotationFound := findDevAnnotation(m.block)
	if !devAnnotationFound {
		return false, nil
	}

	originalSource := getAttrValueAsString(m.block.Body().GetAttribute("source"))
	originalVersion := getAttrValueAsString(m.block.Body().GetAttribute("version"))

	if originalSource == "" {
		return false, fmt.Errorf("module has no source attribute")
	}

	stateAnnotationStr := buildStateAnnotation(originalSource, originalVersion)

	body := m.block.Body()
	inputTokens := body.BuildTokens(nil)
	outputTokens := hclwrite.Tokens{}
	sourceReplaced := false
	braceLevel := 0
	parenLevel := 0
	for i := 0; i < len(inputTokens); i++ {
		token := inputTokens[i]

		if token.Type == hclsyntax.TokenOBrace {
			braceLevel++
		}

		if token.Type == hclsyntax.TokenCBrace {
			braceLevel--
		}
		if token.Type == hclsyntax.TokenOParen {
			parenLevel++
		}
		if token.Type == hclsyntax.TokenCParen {
			parenLevel--
		}

		// Find the `source` attribute and replace its line with the new local path.
		if !sourceReplaced && token.Type == hclsyntax.TokenIdent && string(token.Bytes) == "source" && braceLevel == 0 && parenLevel == 0 {
			outputTokens = append(outputTokens, buildAttributeTokens("source", devPath)...)
			sourceReplaced = true

			// Skip the original source attribute tokens until the next newline.
			tokensToSkip, err := tokensUntil(inputTokens[i:], hclsyntax.TokenNewline)
			if err != nil {
				return false, fmt.Errorf("failed to find end of source attribute: %w", err)
			}
			i += tokensToSkip
			continue
		}

		// Find and remove the `version` attribute line.
		if token.Type == hclsyntax.TokenIdent && string(token.Bytes) == "version" && braceLevel == 0 && parenLevel == 0 {
			tokensToSkip, err := tokensUntil(inputTokens[i:], hclsyntax.TokenNewline)
			if err != nil {
				return false, fmt.Errorf("failed to find end of version attribute: %w", err)
			}
			i += tokensToSkip
			continue
		}

		outputTokens = append(outputTokens, token)

		// Find the dev annotation and inject the state annotation right after it.
		if token.Type == hclsyntax.TokenComment {
			if _, isDev := parseDevAnnotation(string(token.Bytes)); isDev {
				outputTokens = append(outputTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenComment,
					Bytes: []byte(stateAnnotationStr),
				}, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
		}
	}

	body.Clear()
	body.AppendUnstructuredTokens(outputTokens)

	logrus.Infof("loading module '%s' with local path '%s'\n", m.name, devPath)
	return true, nil
}

// Unload deactivates dev mode by restoring the original source and version
// from the state annotation and removing the annotation itself.
// It uses token manipulation to preserve user comments and formatting.
// It returns true if a change was made.
func (m *Module) Unload() (bool, error) {
	state, stateAnnotationFound := findStateAnnotation(m.block)
	if !stateAnnotationFound {
		return false, nil
	}

	body := m.block.Body()
	inputTokens := body.BuildTokens(nil)
	outputTokens := hclwrite.Tokens{}
	sourceReplaced := false

	for i := 0; i < len(inputTokens); i++ {
		token := inputTokens[i]

		// Skip existing terralink-state comments.
		if token.Type == hclsyntax.TokenComment {
			if _, isState := parseStateAnnotation(string(token.Bytes)); isState {
				// Also skip the following newline if it exists.
				if i+1 < len(inputTokens) && inputTokens[i+1].Type == hclsyntax.TokenNewline {
					i++
				}
				continue
			}
		}

		// Find and replace the source attribute.
		if !sourceReplaced && token.Type == hclsyntax.TokenIdent && string(token.Bytes) == "source" {
			outputTokens = append(outputTokens, buildAttributeTokens("source", state.Source)...)
			if state.Version != "" {
				outputTokens = append(outputTokens, buildAttributeTokens("version", state.Version)...)
			}
			sourceReplaced = true

			tokensToSkip, err := tokensUntil(inputTokens[i:], hclsyntax.TokenNewline)
			if err != nil {
				return false, fmt.Errorf("failed to find end of source attribute: %w", err)
			}
			i += tokensToSkip
			continue
		}

		outputTokens = append(outputTokens, token)
	}

	body.Clear()
	body.AppendUnstructuredTokens(outputTokens)
	logrus.Infof("unloading module '%s' to original source '%s'\n", m.name, state.Source)
	return true, nil
}

// --- Token Helpers ---

// tokensUntil is a helper to count tokens until a specific type is found.
func tokensUntil(tokens hclwrite.Tokens, tokenType hclsyntax.TokenType) (int, error) {
	for i, token := range tokens {
		if token.Type == tokenType {
			return i, nil
		}
	}
	return 0, fmt.Errorf("token %s not found in tokens", tokenType)
}

// buildAttributeTokens creates the HCL tokens for a complete attribute line.
func buildAttributeTokens(name, value string) hclwrite.Tokens {
	// Using NewExpressionLiteral ensures proper quoting and escaping.
	expr := hclwrite.NewExpressionLiteral(cty.StringVal(value))
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(name)},
		{Type: hclsyntax.TokenEqual, Bytes: []byte("=")},
	}
	tokens = append(tokens, expr.BuildTokens(nil)...)
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	return tokens
}
