package cmd

import (
	"fmt"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/spf13/cobra"
)

var variablesGroupCmd = &cobra.Command{
	Use:     "group",
	Aliases: []string{"groups"},
	Short:   "Fetches group-level CI/CD variables",
	Long: `Fetches group-level CI/CD variables.
- Use --group-id to fetch variables from a specific group.
- If no flag is provided, it fetches variables from all accessible groups.`,
	RunE: runGroupVariables,
}

func init() {
	variablesCmd.AddCommand(variablesGroupCmd)
	variablesGroupCmd.Flags().StringVar(&variablesGroupID, "group-id", "",
		"The ID or path of the GitLab group to fetch variables from.")
}

func runGroupVariables(_ *cobra.Command, _ []string) error {
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

	variables, err := client.GetGroupVariables(variablesGroupID)
	if err != nil {
		return fmt.Errorf("failed to fetch group variables: %w", err)
	}

	if err := formatter.FormatGroupVariables(variables); err != nil {
		return fmt.Errorf("failed to format group variables: %w", err)
	}

	return nil
}
