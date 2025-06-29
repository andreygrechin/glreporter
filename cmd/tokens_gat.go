package cmd

import (
	"fmt"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	includeInactive bool
	fetchAll        bool
)

var gatCmd = &cobra.Command{
	Use:     "gat",
	Aliases: []string{"group-access-tokens"},
	Short:   "Fetches and displays group access tokens",
	Long:    `Fetches and displays group access tokens for the specified GitLab group.`,
	RunE:    runGAT,
}

func init() {
	gatCmd.Flags().BoolVar(&includeInactive, "include-inactive", false, "Include inactive tokens in the output")
	gatCmd.Flags().BoolVar(&fetchAll, "all", true, "Fetch tokens from all subgroups")
}

func runGAT(_ *cobra.Command, _ []string) error {
	tokenValue := getToken()
	if tokenValue == "" {
		return ErrGitLabTokenRequired
	}

	client, err := glclient.NewClient(tokenValue, debug)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerDelay*time.Millisecond)
	s.Suffix = " Fetching group access tokens..."
	s.Start()

	var tokens []*glclient.GroupAccessTokenWithGroup
	if fetchAll {
		tokens, err = client.GetGroupAccessTokensRecursively(groupID, includeInactive)
	} else {
		tokens, err = client.GetGroupAccessTokens(groupID, includeInactive)
	}

	s.Stop()

	if err != nil {
		if fetchAll {
			return fmt.Errorf("failed to fetch group access tokens recursively: %w", err)
		}

		return fmt.Errorf("failed to fetch group access tokens: %w", err)
	}

	formatter, err := output.NewFormatter(output.Format(format))
	if err != nil {
		return fmt.Errorf("invalid output format: %w", err)
	}

	if err := formatter.FormatGroupAccessTokens(tokens); err != nil {
		return fmt.Errorf("failed to format group access tokens: %w", err)
	}

	return nil
}
