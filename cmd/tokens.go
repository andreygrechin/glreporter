package cmd

import (
	"github.com/spf13/cobra"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage tokens operations",
	Long:  `Manage various token operations for GitLab groups and projects.`,
}

func init() {
	tokensCmd.PersistentFlags().IntVar(&groupID, "group-id", 0,
		"The ID of a GitLab group to start the search from "+
			"(optional for pat/ptt, fetches from all accessible groups if neither group-id nor project-id is provided).")

	tokensCmd.PersistentFlags().IntVar(&projectID, "project-id", 0,
		"Project ID to fetch tokens for.")

	RootCmd.AddCommand(tokensCmd)
	tokensCmd.AddCommand(gatCmd)
	tokensCmd.AddCommand(patCmd)
	tokensCmd.AddCommand(pttCmd)
}
