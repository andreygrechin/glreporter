package cmd

import (
	"strings"

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
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		groupID = strings.Trim(groupID, "/")
	},
	RunE: runGroups,
}

func init() {
	groupsCmd.PersistentFlags().StringVar(&groupID, "group-id", "",
		"The ID or path of the top-level GitLab group to start the search from. "+
			"Can be a numeric ID or a path with namespace (org/subgroup). "+
			"(optional, fetches all accessible groups if not provided)")

	RootCmd.AddCommand(groupsCmd)
}

func runGroups(_ *cobra.Command, _ []string) error {
	return runReportCommand(
		func(client *glclient.Client, groupID string) ([]*gitlab.Group, error) {
			return client.GetGroupsRecursively(groupID)
		},
		func(formatter output.Formatter, data []*gitlab.Group) error {
			return formatter.FormatGroups(data)
		},
		ErrGitLabTokenRequired,
		"Fetching groups...",
	)
}
