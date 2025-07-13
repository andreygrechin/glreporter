package output_test

import (
	"os"
	"testing"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name    string
		format  output.Format
		wantErr bool
	}{
		{"Table format", output.FormatTable, false},
		{"JSON format", output.FormatJSON, false},
		{"CSV format", output.FormatCSV, false},
		{"Invalid format", output.Format("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := output.NewFormatter(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFormatter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatProjectVariables(t *testing.T) {
	testVariables := []*glclient.ProjectVariableWithProject{
		{
			ProjectVariable: &gitlab.ProjectVariable{
				Key:              "DB_PASSWORD",
				Value:            "secret123",
				VariableType:     "env_var",
				Protected:        true,
				Masked:           true,
				EnvironmentScope: "production",
			},
			ProjectName:      "api-service",
			ProjectPath:      "backend/api-service",
			ProjectNamespace: "backend",
			ProjectWebURL:    "https://gitlab.com/backend/api-service",
		},
		{
			ProjectVariable: &gitlab.ProjectVariable{
				Key:              "CONFIG_FILE",
				Value:            "config content here",
				VariableType:     "file",
				Protected:        false,
				Masked:           false,
				EnvironmentScope: "*",
			},
			ProjectName:      "web-app",
			ProjectPath:      "frontend/web-app",
			ProjectNamespace: "frontend",
			ProjectWebURL:    "https://gitlab.com/frontend/web-app",
		},
	}

	t.Run("formats variables as table", func(t *testing.T) {
		formatter, err := output.NewFormatter(output.FormatTable)
		require.NoError(t, err)

		// Capture stdout
		old := captureStdout(t)
		defer restoreStdout(old)

		err = formatter.FormatProjectVariables(testVariables, true)
		assert.NoError(t, err)
	})

	t.Run("formats variables as JSON", func(t *testing.T) {
		formatter, err := output.NewFormatter(output.FormatJSON)
		require.NoError(t, err)

		// Capture stdout
		old := captureStdout(t)
		defer restoreStdout(old)

		err = formatter.FormatProjectVariables(testVariables, true)
		assert.NoError(t, err)
	})

	t.Run("formats variables as CSV", func(t *testing.T) {
		formatter, err := output.NewFormatter(output.FormatCSV)
		require.NoError(t, err)

		// Capture stdout
		old := captureStdout(t)
		defer restoreStdout(old)

		err = formatter.FormatProjectVariables(testVariables, true)
		assert.NoError(t, err)
	})

	t.Run("handles empty variables list", func(t *testing.T) {
		formatter, err := output.NewFormatter(output.FormatTable)
		require.NoError(t, err)

		err = formatter.FormatProjectVariables([]*glclient.ProjectVariableWithProject{}, true)
		assert.NoError(t, err)
	})
}

// Helper functions for capturing stdout during tests.
func captureStdout(t *testing.T) *os.File {
	t.Helper()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	return old
}

func restoreStdout(old *os.File) {
	os.Stdout = old
}

func TestCSVFormatterHandlesNilEmbeddedStructs(t *testing.T) {
	// Test case where embedded pointer struct might be nil
	testVariables := []*glclient.ProjectVariableWithProject{
		{
			// ProjectVariable is nil - this tests the nil embedded struct handling
			ProjectVariable:  nil,
			ProjectName:      "test-project",
			ProjectPath:      "test/project",
			ProjectNamespace: "test",
			ProjectWebURL:    "https://gitlab.com/test/project",
		},
	}

	formatter, err := output.NewFormatter(output.FormatCSV)
	require.NoError(t, err)

	// Capture stdout
	old := captureStdout(t)
	defer restoreStdout(old)

	// This should not panic even with nil embedded struct
	err = formatter.FormatProjectVariables(testVariables, true)
	assert.NoError(t, err)
}
