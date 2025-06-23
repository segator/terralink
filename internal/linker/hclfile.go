package linker

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// HCLFile represents a single Terraform (.tf) file. It encapsulates the file path,
// the parsed HCL content, and the modules defined within it.
type HCLFile struct {
	path    string
	hclFile *hclwrite.File
	modules []*Module
}

// NewHCLFile reads and parses a Terraform file from the given path.
// It returns a new HCLFile instance or an error if the file cannot be
// read or parsed.
func NewHCLFile(path string) (*HCLFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	hclFile, diags := hclwrite.ParseConfig(content, path, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL in %s: %w", path, diags)
	}

	return &HCLFile{path: path, hclFile: hclFile}, nil
}

// Modules returns a slice of all "module" blocks found in the HCL file.
// It parses the blocks on the first call and caches the result.
func (f *HCLFile) Modules() []*Module {
	if f.modules != nil {
		return f.modules
	}

	f.modules = []*Module{}
	for _, block := range f.hclFile.Body().Blocks() {
		if block.Type() == "module" {
			// We expect module blocks to have exactly one label (the module name).
			if len(block.Labels()) == 1 {
				moduleName := block.Labels()[0]
				f.modules = append(f.modules, NewModule(moduleName, block))
			}
		}
	}
	return f.modules
}

// Write saves the current in-memory representation of the HCL file
// back to disk, overwriting the original file.
func (f *HCLFile) Write() error {
	// Format the file before writing
	f.hclFile.Body().BuildTokens(nil)
	err := os.WriteFile(f.path, f.hclFile.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", f.path, err)
	}
	return nil
}
