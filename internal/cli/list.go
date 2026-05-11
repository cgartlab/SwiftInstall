package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "查看清单中的软件列表",
		Long: `解析并显示清单文件中的所有软件包，按分类分组展示。

示例:
  sis list                       使用默认清单
  sis list -f apps.yaml          指定清单文件
  sis list --format json         JSON 格式输出
  sis list --category dev        按分类过滤`,
		RunE: runList,
	}

	cmd.Flags().String("category", "", "按分类过滤 (仅显示指定分类的软件)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	manifestPath, _ := cmd.Flags().GetString("file")
	if manifestPath == "" {
		manifestPath = resolveManifest()
	}
	if manifestPath == "" {
		return fmt.Errorf("未找到清单文件")
	}

	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("解析清单文件失败: %w", err)
	}

	categoryFilter, _ := cmd.Flags().GetString("category")
	format, _ := cmd.Flags().GetString("format")

	// Filter by category if specified
	packages := m.Packages
	if categoryFilter != "" {
		var filtered []manifest.Package
		for _, p := range packages {
			if p.Category == categoryFilter {
				filtered = append(filtered, p)
			}
		}
		packages = filtered
	}

	if format == "json" {
		return listJSON(packages, manifestPath)
	}

	return listTable(packages, manifestPath)
}

func listTable(packages []manifest.Package, source string) error {
	fmt.Fprintf(os.Stderr, "\nPackages from %s\n\n", source)

	if len(packages) == 0 {
		fmt.Fprintf(os.Stderr, "  (空)\n\n")
		return nil
	}

	// Group by category for nice display
	type group struct {
		category string
		packages []manifest.Package
	}

	var groups []group
	groupMap := make(map[string]int)

	for _, pkg := range packages {
		cat := pkg.Category
		if cat == "" {
			cat = "(未分类)"
		}
		if idx, ok := groupMap[cat]; ok {
			groups[idx].packages = append(groups[idx].packages, pkg)
		} else {
			groupMap[cat] = len(groups)
			groups = append(groups, group{category: cat, packages: []manifest.Package{pkg}})
		}
	}

	for _, g := range groups {
		fmt.Fprintf(os.Stderr, "  \033[1m%s\033[0m\n", g.category)
		for _, pkg := range g.packages {
			optional := ""
			if pkg.Optional {
				optional = " \033[2m(optional)\033[0m"
			}
			fmt.Fprintf(os.Stderr, "    %s%s\n", pkg.ID, optional)
		}
		fmt.Fprintln(os.Stderr)
	}

	fmt.Fprintf(os.Stderr, "  Total: %d packages\n\n", len(packages))
	return nil
}

func listJSON(packages []manifest.Package, source string) error {
	type output struct {
		Source   string             `json:"source"`
		Total    int                `json:"total"`
		Packages []manifest.Package `json:"packages"`
	}

	out := output{
		Source:   source,
		Total:    len(packages),
		Packages: packages,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 编码失败: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
