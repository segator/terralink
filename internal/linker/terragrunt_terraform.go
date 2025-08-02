package linker

import (
	"fmt"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sirupsen/logrus"
)

type TerragruntTerraform struct {
	block *hclwrite.Block
}

func (t TerragruntTerraform) Name() string {
	return "terragrunt"
}

func (t TerragruntTerraform) IsLoaded() bool {
	_, found := findStateAnnotation(t.block)
	return found
}

func (t TerragruntTerraform) Load() (bool, error) {
	if t.IsLoaded() {
		return false, nil
	}

	devPath, devAnnotationFound := findDevAnnotation(t.block)
	if !devAnnotationFound {
		return false, nil
	}

	originalSource := getAttrValueAsString(t.block.Body().GetAttribute("source"))

	if originalSource == "" {
		return false, fmt.Errorf("terraform block has no source attribute")
	}

	stateAnnotationStr := buildStateAnnotation(originalSource, "")

	body := t.block.Body()
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
			tokens, err := buildAttributeTokens("source", devPath, false)
			if err != nil {
				return false, fmt.Errorf("failed to build source attribute tokens: %w", err)
			}
			outputTokens = append(outputTokens, tokens...)
			sourceReplaced = true

			// Skip the original source attribute tokens until the next newline.
			tokensToSkip, err := tokensUntil(inputTokens[i:], hclsyntax.TokenNewline)
			if err != nil {
				return false, fmt.Errorf("failed to find end of source attribute: %w", err)
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

	logrus.Infof("loading terragrunt terraform block with local path '%s'\n", devPath)
	return true, nil
}

func (t TerragruntTerraform) Unload() (bool, error) {
	state, stateAnnotationFound := findStateAnnotation(t.block)
	if !stateAnnotationFound {
		return false, nil
	}

	body := t.block.Body()
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
			tokens, err := buildAttributeTokens("source", state.Source, state.SourceIsHCL)
			outputTokens = append(outputTokens, tokens...)
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
	logrus.Infof("unloading terragrunt terraform to original source '%s'\n", state.Source)
	return true, nil
}

func NewTerragruntTerraform(block *hclwrite.Block) *TerragruntTerraform {
	return &TerragruntTerraform{block: block}
}
