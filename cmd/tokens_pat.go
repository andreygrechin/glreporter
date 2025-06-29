package cmd

import (
	"fmt"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var includeInactivePAT bool

var patCmd = &cobra.Command{
	Use:     "pat",
	Aliases: []string{"project-access-tokens"},
	Short:   "Fetches and displays project access tokens",
	Long:    `Fetches and displays project access tokens for the specified GitLab project or all projects in a group.`,
	RunE:    runPAT,
}

func init() {
	patCmd.Flags().BoolVar(&includeInactivePAT, "include-inactive", false,
		"Include inactive tokens in the output")
}

func runPAT(_ *cobra.Command, _ []string) error {
	tokenValue := getToken()
	if tokenValue == "" {
		return ErrGitLabTokenRequired
	}

	client, err := glclient.NewClient(tokenValue, debug)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerDelay*time.Millisecond)
	s.Suffix = " Fetching project access tokens..."
	s.Start()

	tokens, err := fetchTokens(client)

	s.Stop()

	if err != nil {
		return err
	}

	formatter, err := output.NewFormatter(output.Format(format))
	if err != nil {
		return fmt.Errorf("invalid output format: %w", err)
	}

	if err := formatter.FormatProjectAccessTokens(tokens); err != nil {
		return fmt.Errorf("failed to format project access tokens: %w", err)
	}

	return nil
}

func fetchTokens(client *glclient.Client) ([]*glclient.ProjectAccessTokenWithProject, error) {
	if groupID == 0 && projectID == 0 {
		return nil, ErrGroupOrProjectIDRequired
	}

	if groupID != 0 && projectID != 0 {
		return nil, ErrBothGroupIDAndProjectIDProvided
	}

	if groupID != 0 {
		tokens, err := client.GetProjectAccessTokensRecursively(groupID, includeInactivePAT)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch project access tokens recursively: %w", err)
		}

		return tokens, nil
	}

	tokens, err := client.GetProjectAccessTokens(projectID, includeInactivePAT)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project access tokens: %w", err)
	}

	return tokens, nil
}
