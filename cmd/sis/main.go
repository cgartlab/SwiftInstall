package main

import (
	"os"

	"github.com/cgartlab/SwiftInstall/internal/cli"
)

var (
	// Version is the product version. Release builds override it via
	// -ldflags "-X main.Version=<tag>"; a plain `go build` reports the
	// current development version.
	Version = "0.2.3"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	if err := cli.Execute(Version, Commit, Date); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
