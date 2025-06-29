package cmd

import (
	"fmt"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Fetches and displays information about groups",
	Long:  `Fetches and displays information about GitLab groups starting from the specified group ID.`,
	RunE:  runGroups,
}

func init() {
	groupsCmd.PersistentFlags().IntVar(&groupID, "group-id", 0,
		"The ID of the top-level GitLab group to start the search from (required)")

	if err := groupsCmd.MarkPersistentFlagRequired("group-id"); err != nil {
		panic(fmt.Sprintf("failed to mark group-id flag as required: %v", err))
	}

	RootCmd.AddCommand(groupsCmd)
}

func runGroups(_ *cobra.Command, _ []string) error {
	return runReportCommand(
		func(client *glclient.Client, groupID int) ([]*gitlab.Group, error) {
			return client.GetGroupsRecursively(groupID)
		},
		func(formatter output.Formatter, data []*gitlab.Group) error {
			return formatter.FormatGroups(data)
		},
		ErrGitLabTokenRequired,
		"Fetching groups...",
	)
}
