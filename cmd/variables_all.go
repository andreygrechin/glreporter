package cmd

import (
	"fmt"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var variablesAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Fetch both project and group CI/CD variables",
	Long: `Fetch both project-level and group-level CI/CD variables from GitLab. You can:
- Specify a group ID to fetch project and group variables starting from that group recursively
- Leave blank to fetch all accessible variables`,
	RunE: runVariablesAll,
}

func runVariablesAll(_ *cobra.Command, _ []string) error {
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
	s.Suffix = " Fetching all variables..."
	s.Start()

	projectVariables, groupVariables, err := fetchAllVariables(client)
	if err != nil {
		s.Stop()

		return err
	}

	s.Stop()

	return formatAllVariables(formatter, projectVariables, groupVariables)
}

func fetchAllVariables(client *glclient.Client) (
	[]*glclient.ProjectVariableWithProject,
	[]*glclient.GroupVariableWithGroup,
	error,
) {
	var (
		projectVariables []*glclient.ProjectVariableWithProject
		groupVariables   []*glclient.GroupVariableWithGroup
		err              error
	)

	switch {
	case projectID != "":
		// Single project and its parent groups
		projectVariables, err = client.GetProjectVariables(projectID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch project variables: %w", err)
		}

	case groupID != "":
		// All variables from a group recursively
		projectVariables, err = client.GetProjectVariablesRecursively(groupID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch project variables: %w", err)
		}

		groupVariables, err = client.GetGroupVariablesRecursively(groupID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch group variables: %w", err)
		}

	default:
		// All accessible variables
		projectVariables, err = client.GetProjectVariablesRecursively("")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch project variables: %w", err)
		}

		groupVariables, err = client.GetGroupVariablesRecursively("")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch group variables: %w", err)
		}
	}

	return projectVariables, groupVariables, nil
}

func formatAllVariables(
	formatter output.Formatter,
	projectVariables []*glclient.ProjectVariableWithProject,
	groupVariables []*glclient.GroupVariableWithGroup,
) error {
	allVariables := make([]*glclient.VariableWithSource, 0, len(projectVariables)+len(groupVariables))

	for _, pv := range projectVariables {
		allVariables = append(allVariables, glclient.ConvertProjectVariableToUnified(pv))
	}

	for _, gv := range groupVariables {
		allVariables = append(allVariables, glclient.ConvertGroupVariableToUnified(gv))
	}

	if len(allVariables) == 0 {
		fmt.Println("No variables found")

		return nil
	}

	if err := formatter.FormatUnifiedVariables(allVariables); err != nil {
		return fmt.Errorf("failed to format variables: %w", err)
	}

	return nil
}
