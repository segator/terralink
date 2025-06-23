package linker

import (
	"os"
	"path/filepath"
	"terralink/internal/ignore"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
)

// --- Test Cases Definition ---

var testCases = []struct {
	name              string
	initialHCL        string
	expectedDevLoad   string
	expectedDevUnload string
}{
	{
		name: "Simple module with version",
		initialHCL: `
module "my_module" {
  # terralink: path=../modules/my-module
  source  = "app.terraform.io/my-org/my-module/aws"
  version = "1.0.0"


  some_var = "value"
}
`,
		expectedDevLoad: `
module "my_module" {
  # terralink: path=../modules/my-module
  # terralink-state: source="app.terraform.io/my-org/my-module/aws" version="1.0.0"
  source = "../modules/my-module"


  some_var = "value"
}
`,
		expectedDevUnload: `
module "my_module" {
  # terralink: path=../modules/my-module
  source  = "app.terraform.io/my-org/my-module/aws"
  version = "1.0.0"


  some_var = "value"
}
`,
	},
	{
		name: "Module without version",
		initialHCL: `
module "no_version" {
  # terralink: path=./local_vpc
  source = "git::https://example.com/vpc.git"

}
`,
		expectedDevLoad: `
module "no_version" {
  # terralink: path=./local_vpc
  # terralink-state: source="git::https://example.com/vpc.git"
  source = "./local_vpc"

}
`,
		expectedDevUnload: `
module "no_version" {
  # terralink: path=./local_vpc
  source = "git::https://example.com/vpc.git"

}
`,
	},
	{
		name: "File with multiple modules",
		initialHCL: `
module "unmanaged_module" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"
}

module "managed_module" {
  # comment random
  # terralink: path=../local/managed
  source  = "my-registry/managed/aws"
  version = "1.2.3"
}
`,
		expectedDevLoad: `
module "unmanaged_module" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"
}

module "managed_module" {
  # comment random
  # terralink: path=../local/managed
  # terralink-state: source="my-registry/managed/aws" version="1.2.3"
  source = "../local/managed"
}
`,
		expectedDevUnload: `
module "unmanaged_module" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"
}

module "managed_module" {
  # comment random
  # terralink: path=../local/managed
  source  = "my-registry/managed/aws"
  version = "1.2.3"
}
`,
	},
	{
		name: "Idempotency test for dev mode",
		initialHCL: `
module "already_dev" {
  # terralink: path=../dev/module
  # terralink-state: source="remote/source" version="1.0.0"
  source = "../dev/module"
}
`,
		// Expected output is the same as input, no changes should be made.
		expectedDevLoad: `
module "already_dev" {
  # terralink: path=../dev/module
  # terralink-state: source="remote/source" version="1.0.0"
  source = "../dev/module"
}
`,
		// We don't test prod -> prod idempotency as the 'check' command is the gatekeeper there.
		expectedDevUnload: `
module "already_dev" {
  # terralink: path=../dev/module
  source  = "remote/source"
  version = "1.0.0"
}
`,
	},
}

// --- Test Functions ---

func TestDevLoad(t *testing.T) {
	matcher, err := ignore.NewMatcher(".")
	require.NoError(t, err)
	linker := NewLinker(matcher)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup: create a temporary file
			dir := t.TempDir()
			filePath := filepath.Join(dir, "test.tf")
			require.NoError(t, os.WriteFile(filePath, []byte(tc.initialHCL), 0644))

			// Execute
			_, err := linker.DevLoad(filePath)
			require.NoError(t, err)

			// Verify
			resultBytes, err := os.ReadFile(filePath)
			require.NoError(t, err)

			compareHcl(t, []byte(tc.expectedDevLoad), resultBytes)
		})
	}
}

func TestDevUnload(t *testing.T) {
	matcher, err := ignore.NewMatcher(".")
	require.NoError(t, err)
	linker := NewLinker(matcher)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup: start with the file in dev mode
			dir := t.TempDir()
			filePath := filepath.Join(dir, "test.tf")
			require.NoError(t, os.WriteFile(filePath, []byte(tc.expectedDevLoad), 0644))

			// Execute
			_, err := linker.DevUnload(filePath)
			require.NoError(t, err)

			// Verify
			resultBytes, err := os.ReadFile(filePath)
			require.NoError(t, err)

			compareHcl(t, []byte(tc.expectedDevUnload), resultBytes)
		})
	}
}

func formatHcl(hclCode []byte) string {
	return string(hclwrite.Format(hclCode))
}

func compareHcl(t *testing.T, expectedHCL, actualHCL []byte) {
	expectedFormatted := formatHcl(expectedHCL)
	actualFormatted := formatHcl(actualHCL)
	if actualFormatted != expectedFormatted {
		t.Errorf("Mismatch in prod mode (round trip failed).\n--- EXPECTED ---\n%s\n--- ACTUAL ---\n%s", expectedFormatted, actualFormatted)
	}
}

func TestCheckCommand(t *testing.T) {
	matcher, err := ignore.NewMatcher(".")
	require.NoError(t, err)
	linker := NewLinker(matcher)

	t.Run("Check finds active dev modules", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "dev.tf")
		content := testCases[0].expectedDevLoad // Use a known dev-mode file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write dev file: %v", err)
		}

		loadedModulesPerFile, err := linker.Check(filePath)
		require.NoError(t, err)

		loadedModules, exists := loadedModulesPerFile[filePath]
		if !exists {
			t.Fatalf("Expected to find file %s in loaded modules, but it was not found", filePath)
		}
		if len(loadedModules) != 1 {
			t.Errorf("Expected to find 1 loaded module, but found %d", len(loadedModules))
		}
		if loadedModules[0] != "my_module" {
			t.Errorf("Expected module name 'my_module', got '%s'", loadedModules[0])
		}
	})

	t.Run("Check finds no loaded modules", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "prod.tf")
		content := testCases[0].initialHCL // Use a known prod-mode file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write prod file: %v", err)
		}

		loadedModulesPerFile, err := linker.Check(filePath)
		if err != nil {
			t.Fatalf("Check command failed: %v", err)
		}

		loadedModules, exists := loadedModulesPerFile[filePath]
		if !exists {
			t.Fatalf("Expected to find file %s in loaded modules, but it was not found", filePath)
		}
		if len(loadedModules) != 0 {
			t.Errorf("Expected to find 0 loaded Modules, but found %d", len(loadedModules))
		}
	})
}
