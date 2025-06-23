package linker

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
)

func TestAnnotation_ParseDevAnnotation(t *testing.T) {
	testCases := []struct {
		name         string
		comment      string
		expectedPath string
		expectedBool bool
	}{
		{"Valid Annotation", "# terralink: path=../local/module", "../local/module", true},
		{"Extra Whitespace", "   #   terralink:    path=./module   ", "./module", true},
		{"No Path Key", "# terralink: source=../local/module", "", false},
		{"Invalid Format", "# terralink: ../local/module", "", false},
		{"Not a terralink comment", "# some other comment", "", false},
		{"Empty Comment", "", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, isDev := parseDevAnnotation(tc.comment)
			assert.Equal(t, tc.expectedPath, path)
			assert.Equal(t, tc.expectedBool, isDev)
		})
	}
}

func TestAnnotation_ParseStateAnnotation(t *testing.T) {
	testCases := []struct {
		name          string
		comment       string
		expectedState StateAnnotation
		expectedBool  bool
		expectError   bool
	}{
		{
			name:    "Valid with version",
			comment: `# terralink-state: source="remote/source" version="1.2.3"`,
			expectedState: StateAnnotation{
				Source:  "remote/source",
				Version: "1.2.3",
			},
			expectedBool: true,
		},
		{
			name:    "Valid without version",
			comment: `# terralink-state: source="remote/source"`,
			expectedState: StateAnnotation{
				Source: "remote/source",
			},
			expectedBool: true,
		},
		{
			name:         "Not a state comment",
			comment:      "# Some other comment",
			expectedBool: false,
		},
		{
			name:         "Invalid format",
			comment:      `# terralink-state: source="remote/source" version=1.2.3`, // unquoted version
			expectedBool: true,                                                      // It will find the source
			expectedState: StateAnnotation{
				Source: "remote/source",
			},
		},
		{
			name:         "Empty comment",
			comment:      "",
			expectedBool: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state, isState := parseStateAnnotation(tc.comment)
			assert.Equal(t, tc.expectedBool, isState)
			if isState {
				assert.Equal(t, tc.expectedState, state)
			}
		})
	}
}

func TestAnnotation_BuildStateAnnotation(t *testing.T) {
	t.Run("With source and version", func(t *testing.T) {
		expected := `# terralink-state: source="remote/source" version="1.0.0"`
		actual := buildStateAnnotation("remote/source", "1.0.0")
		assert.Equal(t, expected, actual)
	})

	t.Run("With source only", func(t *testing.T) {
		expected := `# terralink-state: source="remote/source"`
		actual := buildStateAnnotation("remote/source", "")
		assert.Equal(t, expected, actual)
	})
}

func TestAnnotation_GetAttrValueAsString(t *testing.T) {
	t.Run("Valid attribute", func(t *testing.T) {
		hclFile, diags := hclwrite.ParseConfig([]byte(`attr = "value"`), "", hcl.InitialPos)
		assert.False(t, diags.HasErrors())
		attr := hclFile.Body().GetAttribute("attr")
		assert.NotNil(t, attr)
		assert.Equal(t, "value", getAttrValueAsString(attr))
	})

	t.Run("Nil attribute", func(t *testing.T) {
		assert.Equal(t, "", getAttrValueAsString(nil))
	})
}
