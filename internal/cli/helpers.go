package cli

import (
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/preset"
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

// manifestPathFrom resolves the effective manifest path for a command, in this
// order of precedence: --preset (a saved preset) > --file > auto-discovery.
func manifestPathFrom(cmd *cobra.Command) (string, error) {
	if name, _ := cmd.Flags().GetString("preset"); name != "" {
		p, err := preset.Path(name)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("预设 %q 不存在（用 sis preset list 查看已保存的预设）", name)
		}
		return p, nil
	}
	path, _ := cmd.Flags().GetString("file")
	if path == "" {
		path = resolveManifest()
	}
	if path == "" {
		return "", fmt.Errorf("未找到清单文件\n\n" +
			"请使用 -f 指定文件路径、--preset 指定预设，或在当前目录创建 sis.yaml / software_list.txt")
	}
	return path, nil
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
