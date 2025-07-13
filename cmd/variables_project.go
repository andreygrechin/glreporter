package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var variablesProjectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"projects"},
	Short:   "Fetch project-level CI/CD variables",
	Long: `Fetch project-level CI/CD variables from GitLab. You can:
- Specify a project ID to fetch project variables from a single project
- Specify a group ID to fetch project variables starting from that group recursively
- Specify neither to fetch project variables from all accessible projects`,
	RunE: runVariablesProject,
}

func runVariablesProject(_ *cobra.Command, _ []string) error {
	// Trim slashes from IDs
	projectID = strings.Trim(projectID, "/")
	groupID = strings.Trim(groupID, "/")

	// Check for token
	tokenValue := getToken()
	if tokenValue == "" {
		return ErrGitLabTokenRequired
	}

	// Create client
	client, err := glclient.NewClient(tokenValue, debug)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Create formatter
	formatter, err := output.NewFormatter(output.Format(format))
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerDelay*time.Millisecond)
	s.Suffix = " Fetching project variables..."
	s.Start()

	var variables []*glclient.ProjectVariableWithProject

	if projectID != "" {
		// Single project
		variables, err = client.GetProjectVariables(projectID)
		if err != nil {
			s.Stop()

			return fmt.Errorf("failed to fetch variables: %w", err)
		}
	} else {
		// Group recursively or all accessible
		variables, err = client.GetProjectVariablesRecursively(groupID)
		if err != nil {
			s.Stop()

			return fmt.Errorf("failed to fetch variables: %w", err)
		}
	}

	s.Stop()

	// Format variables
	if err := formatter.FormatProjectVariables(variables, includeValues); err != nil {
		return fmt.Errorf("failed to format variables: %w", err)
	}

	return nil
}
