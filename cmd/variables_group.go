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

var variablesGroupCmd = &cobra.Command{
	Use:     "group",
	Aliases: []string{"groups"},
	Short:   "Fetch group-level CI/CD variables",
	Long: `Fetch group-level CI/CD variables from GitLab. You can:
- Specify a group ID to fetch group variables starting from that group recursively
- Leave blank to fetch group variables from all accessible groups`,
	RunE: runVariablesGroup,
}

func runVariablesGroup(_ *cobra.Command, _ []string) error {
	// Trim slashes from ID
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
	s.Suffix = " Fetching group variables..."
	s.Start()

	var variables []*glclient.GroupVariableWithGroup

	if groupID != "" {
		// Single group
		variables, err = client.GetGroupVariables(groupID)
		if err != nil {
			s.Stop()

			return fmt.Errorf("failed to fetch variables: %w", err)
		}
	} else {
		// All accessible groups recursively
		variables, err = client.GetGroupVariablesRecursively("")
		if err != nil {
			s.Stop()

			return fmt.Errorf("failed to fetch variables: %w", err)
		}
	}

	s.Stop()

	// Format variables
	if err := formatter.FormatGroupVariables(variables); err != nil {
		return fmt.Errorf("failed to format variables: %w", err)
	}

	return nil
}
