package cmd

import (
	"fmt"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/spf13/cobra"
)

var variablesAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Fetches all CI/CD variables (project and group)",
	Long: `Fetches all CI/CD variables, from both projects and groups.
- Use --project-id to filter by a specific project.
- Use --group-id to filter by a specific group.
- If no flags are provided, it fetches all variables from all accessible projects and groups.`,
	RunE: runAllVariables,
}

func init() {
	variablesCmd.AddCommand(variablesAllCmd)
	variablesAllCmd.Flags().StringVar(&variablesProjectID, "project-id", "",
		"The ID or path of the GitLab project to fetch variables from.")
	variablesAllCmd.Flags().StringVar(&variablesGroupID, "group-id", "",
		"The ID or path of the GitLab group to fetch variables from.")
}

func runAllVariables(_ *cobra.Command, _ []string) error {
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

	projectVariables, err := client.GetProjectVariablesRecursively(variablesGroupID)
	if err != nil {
		return fmt.Errorf("failed to fetch project variables: %w", err)
	}

	groupVariables, err := client.GetGroupVariables(variablesGroupID)
	if err != nil {
		return fmt.Errorf("failed to fetch group variables: %w", err)
	}

	if err := formatter.FormatProjectVariables(projectVariables); err != nil {
		return fmt.Errorf("failed to format project variables: %w", err)
	}

	if err := formatter.FormatGroupVariables(groupVariables); err != nil {
		return fmt.Errorf("failed to format group variables: %w", err)
	}

	return nil
}
