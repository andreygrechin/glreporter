package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage tokens operations",
	Long:  `Manage various token operations for GitLab groups and projects.`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		groupID = strings.Trim(groupID, "/")
		projectID = strings.Trim(projectID, "/")
	},
}

func init() {
	tokensCmd.PersistentFlags().StringVar(&groupID, "group-id", "",
		"The ID or path of a GitLab group to start the search from. "+
			"Can be a numeric ID or a path with namespace (org/subgroup). "+
			"(optional for pat/ptt, fetches from all accessible groups if neither group-id nor project-id is provided).")

	tokensCmd.PersistentFlags().StringVar(&projectID, "project-id", "",
		"The ID or path of a GitLab project to fetch tokens for. "+
			"Can be a numeric ID or a path with namespace (org/subgroup/project).")

	RootCmd.AddCommand(tokensCmd)
	tokensCmd.AddCommand(gatCmd)
	tokensCmd.AddCommand(patCmd)
	tokensCmd.AddCommand(pttCmd)
}
