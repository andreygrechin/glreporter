package cmd

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

var (
	variablesProjectID string
	variablesGroupID   string
)

var ErrSubcommandRequired = errors.New("a subcommand (project, group, or all) must be specified")

var variablesCmd = &cobra.Command{
	Use:   "variables",
	Short: "Fetches and displays CI/CD variables",
	Long: `Provides commands to fetch CI/CD variables from GitLab projects and groups.
Use one of the subcommands to specify the scope of variables to fetch.`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		variablesGroupID = strings.Trim(variablesGroupID, "/")
		variablesProjectID = strings.Trim(variablesProjectID, "/")
	},
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return ErrSubcommandRequired
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(variablesCmd)
}
