package linker

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a test module and its parent file from HCL string content
func createTestModule(t *testing.T, content string) (*Module, *hclwrite.File) {
	t.Helper()
	hclFile, diags := hclwrite.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "HCL parsing failed")

	block := hclFile.Body().Blocks()[0]
	require.NotNil(t, block, "No module block found in test HCL")
	return NewModule(block.Labels()[0], block), hclFile
}

func TestModule_Load(t *testing.T) {
	testCases := []struct {
		name         string
		initialHCL   string
		expectedHCL  string
		expectChange bool
		expectErr    bool
	}{
		{
			name: "Load a module with version",
			initialHCL: `
module "test" {
  # terralink: path=../local
  source  = "remote/source"
  version = "1.0.0"
}`,
			expectedHCL: `
module "test" {
  # terralink: path=../local
  # terralink-state: source="remote/source" version="1.0.0"
  source = "../local"
}`,
			expectChange: true,
		},
		{
			name: "Load a module without version",
			initialHCL: `
module "test" {
  # terralink: path=../local
  source  = "remote/source"
}`,
			expectedHCL: `
module "test" {
  # terralink: path=../local
  # terralink-state: source="remote/source"
  source = "../local"
}`,
			expectChange: true,
		},
		{
			name: "Idempotency: Do not load an already loaded module",
			initialHCL: `
module "test" {
  # terralink: path=../local
  # terralink-state: source="remote/source" version="1.0.0"
  source = "../local"
}`,
			expectedHCL: `
module "test" {
  # terralink: path=../local
  # terralink-state: source="remote/source" version="1.0.0"
  source = "../local"
}`,
			expectChange: false,
		},
		{
			name: "Do not load module without annotation",
			initialHCL: `
module "test" {
  source  = "remote/source"
  version = "1.0.0"
}`,
			expectedHCL: `
module "test" {
  source  = "remote/source"
  version = "1.0.0"
}`,
			expectChange: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			module, hclFile := createTestModule(t, tc.initialHCL)
			changed, err := module.Load()

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectChange, changed)
			assert.Equal(t, formatHcl([]byte(tc.expectedHCL)), formatHcl(hclFile.Bytes()))
		})
	}
}

func TestModule_Unload(t *testing.T) {
	testCases := []struct {
		name         string
		initialHCL   string
		expectedHCL  string
		expectChange bool
		expectErr    bool
	}{
		{
			name: "Unload a loaded module with version",
			initialHCL: `
module "test" {
  # terralink: path=../local
  # terralink-state: source="remote/source" version="1.0.0"
  source = "../local"
}`,
			expectedHCL: `
module "test" {
  # terralink: path=../local
  source  = "remote/source"
  version = "1.0.0"
}`,
			expectChange: true,
		},
		{
			name: "Unload a loaded module without version",
			initialHCL: `
module "test" {
  # terralink: path=../local
  # terralink-state: source="remote/source"
  source = "../local"
}`,
			expectedHCL: `
module "test" {
  # terralink: path=../local
  source = "remote/source"
}`,
			expectChange: true,
		},
		{
			name: "Idempotency: Do not unload a module not in dev mode",
			initialHCL: `
module "test" {
  # terralink: path=../local
  source  = "remote/source"
  version = "1.0.0"
}`,
			expectedHCL: `
module "test" {
  # terralink: path=../local
  source  = "remote/source"
  version = "1.0.0"
}`,
			expectChange: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			module, hclFile := createTestModule(t, tc.initialHCL)
			changed, err := module.Unload()

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectChange, changed)
			assert.Equal(t, formatHcl([]byte(tc.expectedHCL)), formatHcl(hclFile.Bytes()))
		})
	}
}
