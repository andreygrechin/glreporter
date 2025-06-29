package cmd

import (
	"fmt"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Fetches and displays information about projects",
	Long:  `Fetches and displays information about GitLab projects by recursively traversing all subgroups.`,
	RunE:  runProjects,
}

func init() {
	projectsCmd.PersistentFlags().IntVar(&groupID, "group-id", 0,
		"The ID of the top-level GitLab group to start the search from (required)")

	if err := projectsCmd.MarkPersistentFlagRequired("group-id"); err != nil {
		panic(fmt.Sprintf("failed to mark group-id flag as required: %v", err))
	}

	RootCmd.AddCommand(projectsCmd)
}

func runProjects(_ *cobra.Command, _ []string) error {
	return runReportCommand(
		func(client *glclient.Client, groupID int) ([]*gitlab.Project, error) {
			return client.GetProjectsRecursively(groupID)
		},
		func(formatter output.Formatter, data []*gitlab.Project) error {
			return formatter.FormatProjects(data)
		},
		ErrGitLabTokenRequired,
		"Fetching projects...",
	)
}
