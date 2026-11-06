package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/cgartlab/SwiftInstall/internal/audit"
	"github.com/cgartlab/SwiftInstall/internal/toollist"
	"github.com/spf13/cobra"
)

func newGuideCmd() *cobra.Command {
	var catalogFile, toollistID string
	var list bool
	cmd := &cobra.Command{
		Use:   "guide [软件名]",
		Short: "查看从某软件迁移到免费/开源平替的指导（只给指导，不动你的数据）",
		Long: `显示迁移 Playbook：从某软件换到其免费/开源平替的步骤、数据保真度与来源。

只提供指导，不会安装或卸载任何软件，也不会搬动你的数据。

示例:
  sis guide "CleanShot X"
  sis guide Sketch
  sis guide --list
  sis guide Typora --catalog my-list.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cat, err := loadToollist(catalogFile, toollistID)
			if err != nil {
				return err
			}
			if list || len(args) == 0 {
				return listGuides(cat)
			}
			name := args[0]
			eq := audit.FindEquivalence(cat, name)
			if eq == nil {
				fmt.Fprintf(os.Stderr, "未找到 %q 的迁移指导。可用的有：\n", name)
				return listGuides(cat)
			}
			printGuide(eq)
			return nil
		},
	}
	cmd.Flags().StringVar(&catalogFile, "catalog", "", "自定义软件清单 YAML 路径（默认使用内置软件清单）")
	cmd.Flags().StringVar(&toollistID, "toollist", "", "内置软件清单 id（如 mac-designers / developers）")
	cmd.Flags().BoolVar(&list, "list", false, "列出所有可用的迁移指导")
	return cmd
}

func listGuides(cat *toollist.Catalog) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "从\t→ 到\t许可\t含指导")
	for i := range cat.Equivalences {
		eq := &cat.Equivalences[i]
		has := "—"
		if eq.Playbook != nil {
			has = "✓"
		}
		from := ""
		if len(eq.From) > 0 {
			from = eq.From[0]
		}
		fmt.Fprintf(tw, "%s\t→ %s\t%s\t%s\n", from, eq.To.Name, licenseLabel(eq.To.License), has)
	}
	tw.Flush()
	return nil
}

func printGuide(eq *toollist.Equivalence) {
	from := ""
	if len(eq.From) > 0 {
		from = eq.From[0]
	}
	fmt.Printf("\n%s → %s（%s）\n", from, eq.To.Name, licenseLabel(eq.To.License))
	if eq.Playbook == nil {
		fmt.Printf("\n  暂无迁移指导。可先参考：%s\n\n", joinURLs(eq.Sources))
		return
	}
	pb := eq.Playbook
	fmt.Printf("  %s\n", pb.Summary)
	fmt.Printf("  数据保真：%s\n", pb.DataFidelity)
	fmt.Printf("  核对日期：%s\n\n", pb.VerifiedAt)
	for i, st := range pb.Steps {
		fmt.Printf("  %d. %s\n", i+1, st.Title)
		if st.Detail != "" {
			fmt.Printf("     %s\n", st.Detail)
		}
		if st.SourceURL != "" {
			fmt.Printf("     来源：%s\n", st.SourceURL)
		}
	}
	fmt.Printf("\n  更多参考：%s\n\n", joinURLs(eq.Sources))
}

func joinURLs(us []string) string {
	if len(us) == 0 {
		return "—"
	}
	out := us[0]
	for _, u := range us[1:] {
		out += "  ·  " + u
	}
	return out
}
