package linker

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a test module and its parent file from HCL string content
func createTestTerragruntTerraform(t *testing.T, content string) (LoadableBlock, *hclwrite.File) {
	t.Helper()
	hclFile, diags := hclwrite.ParseConfig([]byte(content), "terragrunt.hcl", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "HCL parsing failed")

	block := hclFile.Body().Blocks()[0]
	require.NotNil(t, block, "No module block found in test HCL")
	return NewTerragruntTerraform(block), hclFile
}

func TestTerragruntTerraform_Load(t *testing.T) {
	testCases := []struct {
		name         string
		initialHCL   string
		expectedHCL  string
		expectChange bool
		expectErr    bool
	}{
		{
			name: "Load terraform",
			initialHCL: `
terraform {
  # terralink: path=../local
  source  = "tfr://domain.com/remote/source?version=1.0.0"  
}`,
			expectedHCL: `
terraform {
  # terralink: path=../local
  # terralink-state: source="tfr://domain.com/remote/source?version=1.0.0"
  source = "../local"
}`,
			expectChange: true,
		},
		{
			name: "Load terraform with hcl as source",
			initialHCL: `
terraform {
  # terralink: path=../local
  source  = include.root.locals.module_path
}`,
			expectedHCL: `
terraform {
  # terralink: path=../local
  # terralink-state: source="hcl:include.root.locals.module_path"
  source = "../local"
}`,
			expectChange: true,
		},
		{
			name: "Idempotency: Do not load an already loaded module",
			initialHCL: `
terraform {
  # terralink: path=../local
  # terralink-state: source="remote/source?version=1.0.0"
  source = "../local"
}`,
			expectedHCL: `
terraform {
  # terralink: path=../local
  # terralink-state: source="remote/source?version=1.0.0"
  source = "../local"
}`,
			expectChange: false,
		},
		{
			name: "Do not load terragrunt without annotation",
			initialHCL: `
terraform {
  source  = "remote/source?version=1.0.0"
}`,
			expectedHCL: `
terraform {
  source  = "remote/source?version=1.0.0"
}`,
			expectChange: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			terragruntTerraformBlock, hclFile := createTestTerragruntTerraform(t, tc.initialHCL)
			changed, err := terragruntTerraformBlock.Load()

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

func TestTerragruntTerraform_Unload(t *testing.T) {
	testCases := []struct {
		name         string
		initialHCL   string
		expectedHCL  string
		expectChange bool
		expectErr    bool
	}{
		{
			name: "Unload a loaded terragrunt terraform",
			initialHCL: `
terraform {
  # terralink: path=../local
  # terralink-state: source="remote/source?version=1.0.0"
  source = "../local"
}`,
			expectedHCL: `
terraform {
  # terralink: path=../local
  source  = "remote/source?version=1.0.0"
}`,
			expectChange: true,
		},
		{
			name: "Unload a loaded terragrunt terraform with hcl",
			initialHCL: `
terraform {
  # terralink: path=../local
  # terralink-state: source="hcl:include.root.locals.module_path"
  source = "../local"
}`,
			expectedHCL: `
terraform {
  # terralink: path=../local
  source  = include.root.locals.module_path
}`,
			expectChange: true,
		},
		{
			name: "Unload a loaded terragrunt without version",
			initialHCL: `
terraform {
  # terralink: path=../local
  # terralink-state: source="remote/source"
  source = "../local"
}`,
			expectedHCL: `
terraform {
  # terralink: path=../local
  source = "remote/source"
}`,
			expectChange: true,
		},
		{
			name: "Idempotency: Do not unload a terragrunt not in dev mode",
			initialHCL: `
terraform {
  # terralink: path=../local
  source  = "remote/source?version=1.0.0"
}`,
			expectedHCL: `
terraform {
  # terralink: path=../local
  source  = "remote/source?version=1.0.0"
}`,
			expectChange: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			terragruntTerraformBlock, hclFile := createTestTerragruntTerraform(t, tc.initialHCL)
			changed, err := terragruntTerraformBlock.Unload()

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
