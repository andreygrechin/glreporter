package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var VariablesCmd = &cobra.Command{
	Use:   "variables",
	Short: "Fetches and displays CI/CD variables",
	Long: `Fetches and displays CI/CD variables from GitLab projects. You can:
- Specify a group ID to fetch variables from all projects in that group recursively
- Specify a project ID to fetch variables from a single project
- Specify neither to fetch variables from all accessible groups`,
	RunE: runVariables,
}

var variablesProjectID string

func init() {
	VariablesCmd.PersistentFlags().IntVar(&groupID, "group-id", 0,
		"The ID of the GitLab group to fetch variables from recursively "+
			"(optional, fetches from all accessible groups if neither group-id nor project-id is provided)")
	VariablesCmd.PersistentFlags().StringVar(&variablesProjectID, "project-id", "",
		"The ID of the GitLab project to fetch variables from")

	RootCmd.AddCommand(VariablesCmd)
}

func runVariables(_ *cobra.Command, _ []string) error {
	// Validate parameters
	if err := validateVariablesParameters(); err != nil {
		return err
	}

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
	s.Suffix = " Fetching variables..."
	s.Start()

	variables, err := fetchVariables(client)

	s.Stop()

	if err != nil {
		return err
	}

	// Format variables
	return formatVariables(variables, formatter)
}

func validateVariablesParameters() error {
	if groupID != 0 && variablesProjectID != "" {
		return ErrBothGroupIDAndProjectIDProvided
	}

	return nil
}

func fetchVariables(client *glclient.Client) ([]*glclient.ProjectVariableWithProject, error) {
	var (
		variables []*glclient.ProjectVariableWithProject
		err       error
	)

	if variablesProjectID != "" {
		// Single project
		pid, err := strconv.Atoi(variablesProjectID)
		if err != nil {
			return nil, fmt.Errorf("invalid project ID: %w", err)
		}

		variables, err = client.GetProjectVariables(pid)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch variables: %w", err)
		}
	} else {
		// Group recursively
		variables, err = client.GetProjectVariablesRecursively(groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch variables: %w", err)
		}
	}

	return variables, nil
}

func formatVariables(variables []*glclient.ProjectVariableWithProject, formatter output.Formatter) error {
	if err := formatter.FormatProjectVariables(variables); err != nil {
		return fmt.Errorf("failed to format variables: %w", err)
	}

	return nil
}
