package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/cgartlab/SwiftInstall/internal/audit"
	"github.com/cgartlab/SwiftInstall/internal/toollist"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	var catalogFile, toollistID string
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "体检本机已装软件，给出可核验评价与免费/开源平替建议",
		Long: `扫描本机已安装的软件，结合内置软件清单与平替表，输出：

  · 可替换   —— 已装的付费软件存在更好的免费/开源平替
  · 建议补充 —— 软件清单推荐但尚未安装
  · 已装且够用

判断基于可核验维度（授权、免费版范围、维护活跃度、隐私、可导出性），
非主观好坏。扫描全程在本机完成，安装清单不会离开这台电脑。

示例:
  sis audit
  sis audit --catalog my-list.yaml
  sis audit --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cat, err := loadToollist(catalogFile, toollistID)
			if err != nil {
				return err
			}
			rep, err := audit.Run(context.Background(), audit.SystemLister{}, cat)
			if err != nil {
				return fmt.Errorf("体检失败: %w", err)
			}
			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				b, _ := json.MarshalIndent(rep, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			noColor, _ := cmd.Flags().GetBool("no-color")
			printAudit(rep, !noColor && format != "silent")
			return nil
		},
	}
	cmd.Flags().StringVar(&catalogFile, "catalog", "", "自定义软件清单 YAML 路径（默认使用内置软件清单）")
	cmd.Flags().StringVar(&toollistID, "toollist", "", "内置软件清单 id（默认按当前平台选择；`sis toollist list` 查看可用清单）")
	return cmd
}

func loadToollist(path, id string) (*toollist.Catalog, error) {
	if path != "" {
		return toollist.LoadFile(path)
	}
	if id != "" {
		return toollist.LoadBuiltinNamedFor(id, runtime.GOOS)
	}
	return toollist.LoadBuiltinForPlatform()
}

func licenseLabel(lic string) string {
	switch lic {
	case "oss":
		return "开源"
	case "free":
		return "免费"
	case "freemium":
		return "免费版够用"
	case "paid":
		return "付费"
	default:
		return lic
	}
}

func printAudit(rep *audit.Report, color bool) {
	green := func(s string) string {
		if color {
			return "\x1b[32m" + s + "\x1b[0m"
		}
		return s
	}
	dim := func(s string) string {
		if color {
			return "\x1b[2m" + s + "\x1b[0m"
		}
		return s
	}

	fmt.Printf("\n%s\n", green("SwiftInstall 体检报告 — "+rep.CatalogTitle))
	fmt.Printf("扫描到 %d 个已安装应用\n\n", rep.InstalledCount)

	fmt.Println(green(fmt.Sprintf("可替换 · %d（有更好的免费/开源平替）", len(rep.Replaces))))
	if len(rep.Replaces) == 0 {
		fmt.Println("  （未发现）")
	} else {
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		for _, r := range rep.Replaces {
			fmt.Fprintf(tw, "  %s\t→ %s（%s）\t%s\n", r.From, r.Name, licenseLabel(r.License), r.Why)
		}
		tw.Flush()
	}
	fmt.Println()

	fmt.Println(green(fmt.Sprintf("建议补充 · %d（软件清单推荐，尚未安装）", len(rep.Adds))))
	if len(rep.Adds) == 0 {
		fmt.Println("  （无）")
	} else {
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		for _, r := range rep.Adds {
			role := r.Role
			if role == "" {
				role = "-"
			}
			fmt.Fprintf(tw, "  %s（%s）\t%s\t%s\n", r.Name, licenseLabel(r.License), role, r.Why)
		}
		tw.Flush()
	}
	fmt.Println()

	fmt.Println(green(fmt.Sprintf("已装且够用 · %d", len(rep.Keeps))))
	if len(rep.Keeps) == 0 {
		fmt.Println("  （无）")
	} else {
		fmt.Println("  " + strings.Join(rep.Keeps, "  ·  "))
	}
	if rep.UnmatchedCount > 0 {
		fmt.Println()
		fmt.Printf("另有 %d 个已安装应用未收录在软件清单中；可用 `sis explain <名字>` 用 AI 了解并找免费平替。\n", rep.UnmatchedCount)
	}

	fmt.Println()
	fmt.Println(dim("图例：判断基于可核验维度（授权、免费版范围、维护活跃度、隐私、可导出性），非主观好坏。"))
	if rep.StaleCount > 0 {
		fmt.Println(dim(fmt.Sprintf("提示：其中 %d 项资料已超过一年未复核，仅供参考，请以官方为准。", rep.StaleCount)))
	}
}
