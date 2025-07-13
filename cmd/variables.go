package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var variablesCmd = &cobra.Command{
	Use:   "variables",
	Short: "Manage CI/CD variables",
	Long:  "Manage CI/CD variables from GitLab projects and groups.",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		projectID = strings.Trim(projectID, "/")
		groupID = strings.Trim(groupID, "/")
	},
}

func init() {
	RootCmd.AddCommand(variablesCmd)
	variablesCmd.AddCommand(variablesAllCmd)
	variablesCmd.AddCommand(variablesGroupCmd)
	variablesCmd.AddCommand(variablesProjectCmd)

	variablesCmd.PersistentFlags().StringVar(&groupID, "group-id", "",
		`The ID or path of a GitLab group to start the search from.
Can be a numeric ID or a path with namespace (org/subgroup).`)

	variablesCmd.PersistentFlags().StringVar(&projectID, "project-id", "",
		`The ID or path of a GitLab project to fetch tokens for.
Can be a numeric ID or a path with namespace (org/subgroup/project).`)

	variablesCmd.MarkFlagsMutuallyExclusive("group-id", "project-id")

	variablesAllCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		if err := command.InheritedFlags().MarkHidden("project-id"); err != nil {
			fmt.Fprint(os.Stderr, err)
		}
		command.Parent().HelpFunc()(command, strings)
	})

	variablesGroupCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		if err := command.InheritedFlags().MarkHidden("project-id"); err != nil {
			fmt.Fprint(os.Stderr, err)
		}
		command.Parent().HelpFunc()(command, strings)
	})
}
