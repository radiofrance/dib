package dib_test

import (
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeDag(t *testing.T) *dag.DAG {
	t.Helper()

	rootNode := newNode("test-registry/bullseye", "floor-venus-august-venus", "docker/bullseye")
	firstChildNode := newNode("test-registry/first", "cup-neptune-snake-thirteen", "docker/bullseye/first")
	secondChildNode := newNode("test-registry/second", "blue-bulldog-fourteen-angel", "docker/bullseye/second")
	duplicatedNode := newNode("test-registry/second", "blue-bulldog-fourteen-angel", "docker/bullseye/second")
	subChildNode := newNode("test-registry/third", "lamp-delaware-nineteen-angel", "docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)
	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)
	rootNode.AddChild(duplicatedNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	return DAG
}

func Test_GenerateList_Console(t *testing.T) {
	t.Parallel()

	DAG := setupFakeDag(t)
	opts := dib.FormatOpts{Type: "console"}
	err := dib.GenerateList(DAG, opts)

	require.NoError(t, err)
}

//nolint:lll
func Test_GenerateList_GoTemplateFile(t *testing.T) {
	t.Parallel()

	DAG := setupFakeDag(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	tests := []struct {
		name         string
		outputFormat dib.FormatOpts
		expectError  bool
	}{
		{
			name:         "valid go-template-file",
			outputFormat: dib.FormatOpts{Type: "go-template-file", TemplatePath: path.Join(cwd, "../../test/example_1.yml")},
			expectError:  false,
		},
		{
			name:         "invalid path to go-template-file",
			outputFormat: dib.FormatOpts{Type: "go-template-file", TemplatePath: path.Join(cwd, "lorem")},
			expectError:  true,
		},
		{
			name:         "invalid go-template-file (use of property that doesn't exist)",
			outputFormat: dib.FormatOpts{Type: "go-template-file", TemplatePath: path.Join(cwd, "../../test/invalid_example.yml")},
			expectError:  true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := dib.GenerateList(DAG, test.outputFormat)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_GetImageList(t *testing.T) {
	t.Parallel()

	DAG := setupFakeDag(t)
	actual := dib.GetImagesList(DAG)
	expected := []dag.Image{
		{Name: "test-registry/bullseye", ShortName: "bullseye", Hash: "floor-venus-august-venus"},
		{Name: "test-registry/first", ShortName: "first", Hash: "cup-neptune-snake-thirteen"},
		{Name: "test-registry/second", ShortName: "second", Hash: "blue-bulldog-fourteen-angel"},
		{Name: "test-registry/third", ShortName: "third", Hash: "lamp-delaware-nineteen-angel"},
	}

	assert.Len(t, actual, 4)
	for index := range expected {
		assert.Equal(t, expected[index].Name, actual[index].Name)
		assert.Equal(t, expected[index].ShortName, actual[index].ShortName)
		assert.Equal(t, expected[index].Hash, actual[index].Hash)
	}
}

func Test_ParseOutputOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		given            string
		expected         dib.FormatOpts
		expectedErrorMsg string
	}{
		{
			name:             "Format: console",
			given:            "console",
			expected:         dib.FormatOpts{Type: "console"},
			expectedErrorMsg: "",
		},
		{
			name:             "Format: console (default)",
			given:            "",
			expected:         dib.FormatOpts{Type: "console"},
			expectedErrorMsg: "",
		},
		{
			name:             "Format: go-template-file",
			given:            "go-template-file=/tmp/output.gotemplate",
			expected:         dib.FormatOpts{Type: "go-template-file", TemplatePath: "/tmp/output.gotemplate"},
			expectedErrorMsg: "",
		},
		{
			name:             "Format: go-template-file (invalid)",
			given:            "go-template-file",
			expected:         dib.FormatOpts{},
			expectedErrorMsg: "you need to provide a path to template file when using \"go-template-file\" options",
		},
		{
			name:             "Format: unsupported / invalid",
			given:            "json",
			expected:         dib.FormatOpts{},
			expectedErrorMsg: "\"json\" is not a valid output format",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := dib.ParseOutputOptions(test.given)
			assert.Equal(t, test.expected, actual)
			if test.expectedErrorMsg == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.expectedErrorMsg)
			}
		})
	}
}
