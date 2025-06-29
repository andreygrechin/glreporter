package main

import (
	"github.com/andreygrechin/glreporter/cmd"
)

var (
	// Version of the glreporter application.
	Version = "unknown"
	// BuildTime represents the time when the application was built.
	BuildTime = "unknown"
	// Commit represents the git commit hash of the build.
	Commit = "unknown"
)

func main() {
	cmd.Execute(Version, BuildTime, Commit)
}
