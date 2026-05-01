package main

import (
	"os"

	"github.com/cgartlab/SwiftInstall/internal/cli"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	if err := cli.Execute(Version, Commit, Date); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
