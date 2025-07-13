package cmd

import (
	"fmt"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/spf13/cobra"
)

var variablesProjectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"projects"},
	Short:   "Fetches project-level CI/CD variables",
	Long: `Fetches project-level CI/CD variables.
- Use --project-id to fetch variables from a specific project.
- Use --group-id to fetch variables from all projects in a specific group.
- If no flags are provided, it fetches variables from all accessible projects.`,
	RunE: runProjectVariables,
}

func init() {
	variablesCmd.AddCommand(variablesProjectCmd)
	variablesProjectCmd.Flags().StringVar(&variablesProjectID, "project-id", "",
		"The ID or path of the GitLab project to fetch variables from.")
	variablesProjectCmd.Flags().StringVar(&variablesGroupID, "group-id", "",
		"The ID or path of the GitLab group to fetch project variables from.")
}

func runProjectVariables(_ *cobra.Command, _ []string) error {
	tokenValue := getToken()
	if tokenValue == "" {
		return ErrGitLabTokenRequired
	}

	client, err := glclient.NewClient(tokenValue, debug)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	formatter, err := output.NewFormatter(output.Format(format))
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	var variables []*glclient.ProjectVariableWithProject

	if variablesProjectID != "" {
		variables, err = client.GetProjectVariables(variablesProjectID)
	} else {
		variables, err = client.GetProjectVariablesRecursively(variablesGroupID)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch project variables: %w", err)
	}

	if err := formatter.FormatProjectVariables(variables); err != nil {
		return fmt.Errorf("failed to format project variables: %w", err)
	}

	return nil
}
