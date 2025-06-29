package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/andreygrechin/glreporter/internal/output"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	groupID   int
	projectID int
	format    string
	token     string
	debug     bool
)

var (
	ErrGitLabTokenRequired = errors.New(
		"gitlab token is required. Use --token flag or set GITLAB_TOKEN environment variable")
	ErrGroupOrProjectIDRequired = errors.New(
		"either --group-id or --project-id must be specified")
	ErrBothGroupIDAndProjectIDProvided = errors.New(
		"cannot specify both --group-id and --project-id")
)

var RootCmd = &cobra.Command{
	Use:   "glreporter",
	Short: "A CLI tool to fetch and display GitLab groups and projects",
	Long: `A CLI tool that asynchronously fetches and displays information about ` +
		`GitLab groups and their associated projects.`,
}

const (
	spinnerDelay   = 100
	spinnerCharSet = 11
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(v, buildTime, commit string) {
	RootCmd.Version = fmt.Sprintf("%s (built %s, commit %s)", v, buildTime, commit)
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&format, "format", "table",
		"Output format: table, json, or csv")
	RootCmd.PersistentFlags().StringVar(&token, "token", "",
		"GitLab personal access token (can also be set via GITLAB_TOKEN env var)")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}

// IsDebugEnabled returns whether debug mode is enabled.
func IsDebugEnabled() bool {
	return debug
}

// runReportCommand is a generic function to handle common logic for fetching and formatting data.
func runReportCommand[T any](
	fetchFunc func(client *glclient.Client, groupID int) ([]T, error),
	formatFunc func(formatter output.Formatter, data []T) error,
	tokenErr error,
	spinnerSuffix string,
) error {
	tokenValue := getToken()
	if tokenValue == "" {
		return tokenErr
	}

	client, err := glclient.NewClient(tokenValue, debug)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerDelay*time.Millisecond)
	s.Suffix = " " + spinnerSuffix
	s.Start()

	data, err := fetchFunc(client, groupID)

	s.Stop()

	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}

	formatter, err := output.NewFormatter(output.Format(format))
	if err != nil {
		return fmt.Errorf("invalid output format: %w", err)
	}

	if err := formatFunc(formatter, data); err != nil {
		return fmt.Errorf("failed to format data: %w", err)
	}

	return nil
}

func getToken() string {
	if token != "" {
		return token
	}

	return os.Getenv("GITLAB_TOKEN")
}
