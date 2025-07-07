package cmd

import (
	"strings"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Fetches and displays information about projects",
	Long: `Fetches and displays information about GitLab projects. If a group ID is provided,
it will fetch all projects from that group and its subgroups.
If no group ID is provided, it will fetch projects from all accessible groups.`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		groupID = strings.Trim(groupID, "/")
	},
	RunE: runProjects,
}

func init() {
	projectsCmd.PersistentFlags().StringVar(&groupID, "group-id", "",
		"The ID or path of the top-level GitLab group to start the search from. "+
			"Can be a numeric ID, a path with namespace (org/subgroup/project). "+
			"(optional, fetches from all accessible groups if not provided)")

	RootCmd.AddCommand(projectsCmd)
}

func runProjects(_ *cobra.Command, _ []string) error {
	return runReportCommand(
		func(client *glclient.Client, groupID string) ([]*gitlab.Project, error) {
			return client.GetProjectsRecursively(groupID)
		},
		func(formatter output.Formatter, data []*gitlab.Project) error {
			return formatter.FormatProjects(data)
		},
		ErrGitLabTokenRequired,
		"Fetching projects...",
	)
}
