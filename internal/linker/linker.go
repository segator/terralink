package linker

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"terralink/internal/ignore"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// Annotation constants
const (
	stateAnnotationPrefix = "# terralink-state:"
)

// ActiveLinkInfo holds information about a module currently in dev mode.
type ActiveLinkInfo struct {
	ModuleName string
	FilePath   string
}

type LoadedModules []string

type Linker struct {
	matcher *ignore.IgnoreMatcher
}

func NewLinker(matcher *ignore.IgnoreMatcher) *Linker {
	return &Linker{
		matcher: matcher,
	}
}

func (l *Linker) Check(scanPath string) (map[string]LoadedModules, error) {
	loadedModulesFile := make(map[string]LoadedModules)

	err := filepath.WalkDir(scanPath, func(path string, d fs.DirEntry, _ error) error {
		if l.matcher.ShouldIgnore(path) {
			return filepath.SkipDir
		}

		loadedModules, err := check(path)
		if err != nil {
			return err
		}
		loadedModulesFile[path] = loadedModules

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directories: %w", err)
	}
	return loadedModulesFile, nil
}

func (l *Linker) DevLoad(scanPath string) (map[string]int, error) {
	changesPerFile := make(map[string]int)

	err := filepath.WalkDir(scanPath, func(path string, d fs.DirEntry, _ error) error {
		if l.matcher.ShouldIgnore(path) {
			return nil
		}

		changes, err := devLoad(path)
		if err != nil {
			return err
		}
		if changes > 0 {
			changesPerFile[path] = changes
		}
		return nil
	})
	if err != nil {
		return changesPerFile, fmt.Errorf("error walking directories: %w", err)
	}
	return changesPerFile, nil
}

func (l *Linker) DevUnload(scanPath string) (map[string]int, error) {
	changesPerFile := make(map[string]int)

	err := filepath.WalkDir(scanPath, func(path string, d fs.DirEntry, _ error) error {
		if l.matcher.ShouldIgnore(path) {
			return nil
		}

		changes, err := devUnload(path)
		if err != nil {
			return err
		}
		if changes > 0 {
			changesPerFile[path] = changes
		}

		return nil
	})
	if err != nil {
		return changesPerFile, fmt.Errorf("error walking directories: %w", err)
	}
	return changesPerFile, nil
}

func check(path string) (LoadedModules, error) {
	var loadedModules LoadedModules
	hclFile, err := openFile(path)
	if err != nil {
		return loadedModules, err
	}
	_, err = walkBlocks(hclFile.Body().Blocks(), func(moduleBlock *hclwrite.Block) (int, error) {
		moduleBody := moduleBlock.Body()
		inputModuleBlockTokens := moduleBody.BuildTokens(nil)
		searchForToken(inputModuleBlockTokens, hclsyntax.TokenComment, func(token *hclwrite.Token) bool {
			tokenStr := string(token.Bytes)
			_, tokenIsTerralinkStateAnnotation := parseTerralinkStateAnnotation(tokenStr)
			if tokenIsTerralinkStateAnnotation {
				loadedModules = append(loadedModules, moduleBlock.Labels()[0])
			}
			return tokenIsTerralinkStateAnnotation
		})

		return len(loadedModules), nil
	})
	if err != nil {
		return loadedModules, fmt.Errorf("error walking blocks in %s: %w", path, err)
	}

	return loadedModules, nil
}

func devLoad(path string) (int, error) {
	hclFile, err := openFile(path)
	if err != nil {
		return 0, err
	}
	changes, err := walkBlocks(hclFile.Body().Blocks(), func(moduleBlock *hclwrite.Block) (int, error) {
		var changesMade int
		var moduleTokens hclwrite.Tokens
		moduleBody := moduleBlock.Body()

		sourceAttr := moduleBody.GetAttribute("source")
		if sourceAttr == nil {
			return 0, fmt.Errorf("module %s block does not have a source attribute", moduleBlock.Labels()[0])
		}
		originalSource := getAttrValueAsString(sourceAttr)

		var originalVersion string
		if versionAttr := moduleBody.GetAttribute("version"); versionAttr != nil {
			originalVersion = getAttrValueAsString(versionAttr)

		}
		inputModuleBlockTokens := moduleBody.BuildTokens(nil)
		moduleLocalPath := ""
		_, terraLinkCommentFound := searchForToken(inputModuleBlockTokens, hclsyntax.TokenComment, func(token *hclwrite.Token) bool {
			tokenStr := string(token.Bytes)
			localPath, tokenIsTerralinkAnnotation := parseTerralinkAnnotation(tokenStr)
			if tokenIsTerralinkAnnotation {
				moduleLocalPath = localPath
			}
			return tokenIsTerralinkAnnotation
		})

		searchForToken(inputModuleBlockTokens, hclsyntax.TokenComment, func(token *hclwrite.Token) bool {
			tokenStr := string(token.Bytes)
			terraLinkState, tokenIsTerralinkStateAnnotation := parseTerralinkStateAnnotation(tokenStr)
			if tokenIsTerralinkStateAnnotation {
				originalSource = terraLinkState["source"]
				if version, ok := terraLinkState["version"]; ok && version != "" {
					originalVersion = version
				}
			}
			return tokenIsTerralinkStateAnnotation
		})

		if terraLinkCommentFound {
			moduleBody.SetAttributeValue("source", cty.StringVal(moduleLocalPath))
			moduleBody.RemoveAttribute("version")
		}
		inputModuleBlockTokens = moduleBody.BuildTokens(nil)
		for i := 0; i < len(inputModuleBlockTokens); i++ {
			token := inputModuleBlockTokens[i]
			if token.Type != hclsyntax.TokenComment {
				moduleTokens = append(moduleTokens, token)
				continue
			}

			tokenStr := string(token.Bytes)
			_, tokenIsTerralinkStateAnnotation := parseTerralinkStateAnnotation(tokenStr)
			if tokenIsTerralinkStateAnnotation {
				changesMade++
				continue
			}

			moduleTokens = append(moduleTokens, token)
			_, tokenIsTerralinkAnnotation := parseTerralinkAnnotation(tokenStr)
			if !tokenIsTerralinkAnnotation {
				continue
			}

			moduleTokens = append(moduleTokens, hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenComment,
					Bytes: []byte(buildStateAnnotation(originalSource, originalVersion)),
				},
				{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				},
			}...)
			fmt.Printf("load module: %s on %s\n", moduleBlock.Labels()[0], path)
			changesMade++
		}

		if changesMade > 0 {
			moduleBody.Clear()
			moduleBody.AppendUnstructuredTokens(moduleTokens)
		}
		return changesMade, nil
	})

	if err != nil {
		return changes, err
	}

	if changes > 0 {
		return 0, writeHclFile(path, hclFile)
	}
	return 0, nil
}

func searchForToken(tokens hclwrite.Tokens, tokenType hclsyntax.TokenType, f func(token *hclwrite.Token) bool) (*hclwrite.Token, bool) {
	for _, token := range tokens {
		if token.Type == tokenType {
			if f != nil {
				if f(token) {
					return token, true
				}
			}
		}
	}
	return nil, false
}
func devUnload(path string) (int, error) {
	hclFile, err := openFile(path)
	if err != nil {
		return 0, err
	}
	changes, err := walkBlocks(hclFile.Body().Blocks(), func(moduleBlock *hclwrite.Block) (int, error) {
		var changesMade int
		var moduleTokens hclwrite.Tokens
		moduleBody := moduleBlock.Body()
		inputModuleBlockTokens := moduleBody.BuildTokens(nil)
		originalSource := ""
		originalVersion := ""
		searchForToken(inputModuleBlockTokens, hclsyntax.TokenComment, func(token *hclwrite.Token) bool {
			tokenStr := string(token.Bytes)
			terraLinkState, tokenIsTerralinkStateAnnotation := parseTerralinkStateAnnotation(tokenStr)
			if tokenIsTerralinkStateAnnotation {
				originalSource = terraLinkState["source"]
				if version, ok := terraLinkState["version"]; ok && version != "" {
					originalVersion = version
				}
			}
			return tokenIsTerralinkStateAnnotation
		})
		if originalSource == "" {
			return 0, nil
		}
		moduleBody.SetAttributeValue("source", cty.StringVal(originalSource))

		inputModuleBlockTokens = moduleBody.BuildTokens(nil)
		for i := 0; i < len(inputModuleBlockTokens); i++ {
			token := inputModuleBlockTokens[i]
			if token.Type == hclsyntax.TokenIdent {
				if string(token.Bytes) == "source" {
					c, err := tokensUntil(inputModuleBlockTokens[i:], hclsyntax.TokenNewline)
					if err != nil {
						return 0, fmt.Errorf("failed to find end of source attribute in %s: %w", path, err)
					}
					i = i + c
					moduleTokens = append(moduleTokens, addNewAttributeTokens("source", originalSource)...)
					if originalVersion != "" {
						moduleTokens = append(moduleTokens, addNewAttributeTokens("version", originalVersion)...)
					}
					fmt.Printf("unload module: %s on %s\n", moduleBlock.Labels()[0], path)
					continue
				}
			}
			if token.Type != hclsyntax.TokenComment {
				moduleTokens = append(moduleTokens, token)
				continue
			}

			tokenStr := string(token.Bytes)

			_, tokenIsTerralinkStateAnnotation := parseTerralinkStateAnnotation(tokenStr)
			if !tokenIsTerralinkStateAnnotation {
				moduleTokens = append(moduleTokens, token)
				continue
			}

			changesMade++
		}
		if changesMade > 0 {
			moduleBody.Clear()
			moduleBody.AppendUnstructuredTokens(moduleTokens)
		}
		return changesMade, nil
	})

	if err != nil {
		return changes, err
	}

	if changes > 0 {
		return 0, writeHclFile(path, hclFile)
	}
	return 0, nil
}

func tokensUntil(tokens hclwrite.Tokens, tokenType hclsyntax.TokenType) (int, error) {
	for i, token := range tokens {
		if token.Type == tokenType {
			return i, nil
		}
	}
	return 0, fmt.Errorf("token %s not found in tokens", tokenType)

}

func openFile(path string) (*hclwrite.File, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	hclFile, diags := hclwrite.ParseConfig(content, path, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL in %s: %s", path, diags.Error())
	}

	return hclFile, nil

}

type internalWalkFunc func(block *hclwrite.Block) (int, error)

func walkBlocks(blocks []*hclwrite.Block, fnc internalWalkFunc) (int, error) {
	totalChanges := 0
	for _, moduleBlock := range blocks {
		if moduleBlock.Type() != "module" {
			continue
		}
		moduleChanges, err := fnc(moduleBlock)
		totalChanges += moduleChanges
		if err != nil {
			return totalChanges, err
		}
	}
	return totalChanges, nil
}

func writeHclFile(path string, hclFile *hclwrite.File) error {
	return os.WriteFile(path, hclFile.Bytes(), 0644)
}

// --- Annotation Helper Functions ---

func parseTerralinkAnnotation(comment string) (string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(comment))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		attributes := parseTerralinkAttrs(line)
		if len(attributes) > 0 && attributes["path"] != "" {
			return attributes["path"], true
		}

	}
	return "", false
}

func parseTerralinkStateAnnotation(comment string) (map[string]string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(comment))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, stateAnnotationPrefix) {
			state := make(map[string]string)
			// Use strings.Fields to handle multiple spaces between key-value pairs
			data := strings.TrimPrefix(line, stateAnnotationPrefix)
			scanner := bufio.NewScanner(strings.NewReader(data))
			scanner.Split(bufio.ScanWords)
			for scanner.Scan() {
				pair := scanner.Text()
				if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
					state[kv[0]] = strings.Trim(kv[1], ` "`)
				}
			}
			return state, true
		}
	}
	return nil, false
}

func buildStateAnnotation(source, version string) string {
	versionPart := ""
	if version != "" {
		versionPart = fmt.Sprintf(` version="%s"`, version)
	}
	return fmt.Sprintf(`%s source="%s"%s`, stateAnnotationPrefix, source, versionPart)
}

func getAttrValueAsString(attr *hclwrite.Attribute) string {
	rawStr := string(attr.Expr().BuildTokens(nil).Bytes())
	return strings.Trim(rawStr, ` "`)
}

var lineRe = regexp.MustCompile(`^#\s*terralink:\s*(.*)$`)
var pairRe = regexp.MustCompile(`(\w+)\s*=\s*([^\s]+)`)

func parseTerralinkAttrs(line string) map[string]string {
	result := map[string]string{}
	m := lineRe.FindStringSubmatch(strings.TrimSpace(line))
	if len(m) != 2 {
		return result
	}
	attrs := m[1]
	for _, pair := range pairRe.FindAllStringSubmatch(attrs, -1) {
		if len(pair) == 3 {
			result[pair[1]] = pair[2]
		}
	}
	return result
}

func addNewAttributeTokens(attrName string, attrValue string) hclwrite.Tokens {
	tokens := hclwrite.TokensForIdentifier(attrName)
	tokens = append(tokens, hclwrite.Tokens{
		{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte("="),
		},
	}...)
	tokens = append(tokens, hclwrite.NewExpressionLiteral(cty.StringVal(attrValue)).BuildTokens(nil)...)
	tokens = append(tokens, hclwrite.Tokens{
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		},
	}...)
	return tokens
}
