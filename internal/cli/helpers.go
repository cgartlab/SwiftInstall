package cli

import (
	"os"

	"github.com/cgartlab/SwiftInstall/internal/ui"
	"github.com/spf13/cobra"
)

// resolveManifest searches for a manifest file in the current directory.
// Returns the first found path, or empty string if none exist.
func resolveManifest() string {
	candidates := []string{
		"sis.yaml",
		"sis.yml",
		"software_list.txt",
		"packages.txt",
	}
	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

// newRenderer creates the appropriate Renderer based on CLI flags.
func newRenderer(cmd *cobra.Command) ui.Renderer {
	format, _ := cmd.Flags().GetString("format")
	noColor, _ := cmd.Flags().GetBool("no-color")

	switch format {
	case "json":
		return ui.NewJSONRenderer()
	case "silent":
		return ui.NewSilentRenderer()
	default:
		return ui.NewTerminalRenderer(!noColor)
	}
}
