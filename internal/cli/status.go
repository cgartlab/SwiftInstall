package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "检查清单中各软件的安装状态",
		Long: `对比清单文件与系统已安装软件，显示每个包的当前状态。

示例:
  sis status                     使用默认清单
  sis status -f apps.yaml        指定清单文件
  sis status --format json       JSON 格式输出`,
		RunE: runStatus,
	}

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	manifestPath, err := manifestPathFrom(cmd)
	if err != nil {
		return err
	}

	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("解析清单文件失败: %w", err)
	}

	be, err := backend.ForCurrentOS()
	if err != nil {
		return err
	}
	if err := be.Detect(); err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	noColor, _ := cmd.Flags().GetBool("no-color")
	silent := format == "silent"
	useColor := !noColor && format != "json" && !silent

	type pkgStatus struct {
		Package   manifest.Package `json:"package"`
		Installed bool             `json:"installed"`
		Error     string           `json:"error,omitempty"`
	}

	if !silent && format != "json" {
		fmt.Fprintf(os.Stderr, "\n正在检查 %d 个软件的安装状态（清单：%s）\n\n", len(m.Packages), manifestPath)
	}

	var results []pkgStatus
	installed := 0
	missing := 0
	errors := 0

	// Prefer a single bulk probe (one `brew list`) over per-package IsInstalled,
	// which spawns a backend call per package and is very slow on large manifests.
	var bulkSet map[string]bool
	if provider, ok := be.(backend.InstalledSetProvider); ok {
		if set, err := provider.InstalledSet(context.Background()); err == nil {
			bulkSet = set
		}
	}

	for i, pkg := range m.Packages {
		var isInstalled bool
		var err error
		if bulkSet != nil {
			isInstalled = bulkSet[pkg.ID]
		} else {
			isInstalled, err = be.IsInstalled(context.Background(), pkg.ID)
		}

		status := pkgStatus{Package: pkg, Installed: isInstalled}
		if err != nil {
			status.Error = err.Error()
			errors++
		} else if isInstalled {
			installed++
		} else {
			missing++
		}
		results = append(results, status)

		// Display progress
		if !silent && format != "json" {
			icon := "\033[32m✓\033[0m"
			if !isInstalled {
				icon = "\033[31m✗\033[0m"
			}
			if err != nil {
				icon = "\033[33m?\033[0m"
			}
			if !useColor {
				switch {
				case err != nil:
					icon = "?"
				case isInstalled:
					icon = "+"
				default:
					icon = "-"
				}
			}
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s %s\n", i+1, len(m.Packages), icon, pkg.ID)
		}
	}

	// Summary (silent: no output — only the exit code matters)
	if format == "json" {
		type jsonOutput struct {
			Source    string      `json:"source"`
			Total     int         `json:"total"`
			Installed int         `json:"installed"`
			Missing   int         `json:"missing"`
			Errors    int         `json:"errors,omitempty"`
			Packages  []pkgStatus `json:"packages"`
		}
		out := jsonOutput{
			Source:    manifestPath,
			Total:     len(m.Packages),
			Installed: installed,
			Missing:   missing,
			Errors:    errors,
			Packages:  results,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
	} else if !silent {
		green := "\033[32m"
		red := "\033[31m"
		reset := "\033[0m"
		if !useColor {
			green = ""
			red = ""
			reset = ""
		}
		fmt.Fprintf(os.Stderr, "\n  %s已安装 %d%s，%s未安装 %d%s\n\n",
			green, installed, reset, red, missing, reset)
	}

	return nil
}
