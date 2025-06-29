package cmd

import (
	"fmt"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var pttCmd = &cobra.Command{
	Use:     "ptt",
	Aliases: []string{"pipeline-trigger-tokens"},
	Short:   "Fetch pipeline trigger tokens",
	Long:    `Fetch pipeline trigger tokens for a specific project or all projects in a group`,
	RunE:    runPTT,
}

func init() {
}

func runPTT(_ *cobra.Command, _ []string) error {
	token := getToken()
	if token == "" {
		return ErrGitLabTokenRequired
	}

	// Create GitLab client
	client, err := glclient.NewClient(token, debug)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Create spinner for visual feedback
	s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerDelay*time.Millisecond)
	s.Suffix = " Fetching pipeline trigger tokens..."
	s.Start()

	// Fetch triggers
	triggers, err := fetchTriggers(client)

	s.Stop()

	if err != nil {
		return fmt.Errorf("failed to fetch pipeline triggers in runPTT: %w", err)
	}

	// Format output
	formatter, err := output.NewFormatter(output.Format(format))
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	if err := formatter.FormatPipelineTriggers(triggers); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func fetchTriggers(client *glclient.Client) ([]*glclient.PipelineTriggerWithProject, error) {
	if groupID == 0 && projectID == 0 {
		return nil, ErrGroupOrProjectIDRequired
	}

	if groupID != 0 && projectID != 0 {
		return nil, ErrBothGroupIDAndProjectIDProvided
	}

	if groupID != 0 {
		triggers, err := client.GetPipelineTriggersRecursively(groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pipeline triggers: %w", err)
		}

		return triggers, nil
	}

	triggers, err := client.GetPipelineTriggers(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline triggers: %w", err)
	}

	return triggers, nil
}
