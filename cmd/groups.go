package cmd

import (
	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Fetches and displays information about groups",
	Long: `Fetches and displays information about GitLab groups. If a group ID is provided, 
it will fetch all groups and subgroups starting from that group. 
If no group ID is provided, it will fetch all accessible groups.`,
	RunE: runGroups,
}

func init() {
	groupsCmd.PersistentFlags().IntVar(&groupID, "group-id", 0,
		"The ID of the top-level GitLab group to start the search from "+
			"(optional, fetches all accessible groups if not provided)")

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
