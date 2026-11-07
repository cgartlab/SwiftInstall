package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/toollist"
	"github.com/spf13/cobra"
)

func newToollistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toollist",
		Short: "查看 / 导出软件清单（可分享的 YAML，别人可用 sis audit --catalog 加载）",
		Long: `软件清单是 SwiftInstall 的核心对象：一组只收「免费/开源、免费版够用」的软件，
每一项带理由与版本核验日期，平替还带迁移指导。

可以把软件清单导出成 YAML 分享给别人；对方用 sis audit --catalog <文件> 即可加载，
体检结果会按他自己的机器呈现——软件清单是"活"的。`,
	}
	cmd.AddCommand(newToollistListCmd())
	cmd.AddCommand(newToollistShowCmd())
	cmd.AddCommand(newToollistExportCmd())
	cmd.AddCommand(newToollistVerifyCmd())
	return cmd
}

func newToollistListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出内置软件清单",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := toollist.AvailableNames()
			if err != nil {
				return err
			}
			type row struct {
				ID           string `json:"id"`
				Title        string `json:"title"`
				Items        int    `json:"items"`
				Equivalences int    `json:"equivalences"`
				Playbooks    int    `json:"playbooks"`
			}
			rows := make([]row, 0, len(names))
			for _, n := range names {
				c, err := toollist.LoadBuiltinNamed(n)
				if err != nil {
					continue
				}
				pb := 0
				for _, e := range c.Equivalences {
					if e.Playbook != nil {
						pb++
					}
				}
				rows = append(rows, row{c.ID, c.Title, len(c.Items), len(c.Equivalences), pb})
			}
			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(rows)
			}
			if format == "silent" {
				return nil
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "软件清单 id\t标题\t条目\t平替\t含迁移指导")
			for _, r := range rows {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\n", r.ID, r.Title, r.Items, r.Equivalences, r.Playbooks)
			}
			tw.Flush()
			return nil
		},
	}
}

func newToollistShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [id]",
		Short: "查看某个内置软件清单的内容（默认当前平台清单）",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := toollist.DefaultNameFor(runtime.GOOS)
			if len(args) > 0 {
				id = args[0]
			}
			c, err := toollist.LoadBuiltinNamedFor(id, runtime.GOOS)
			if err != nil {
				return err
			}
			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(c)
			}
			if format == "silent" {
				return nil
			}
			fmt.Printf("\n%s（%s）\n", c.Title, c.ID)
			fmt.Printf("面向：%s · 更新：%s · 条目：%d · 平替：%d\n", c.Audience, c.Updated, len(c.Items), len(c.Equivalences))

			fmt.Println("\n条目：")
			seen := map[string]bool{}
			var cats []string
			for _, it := range c.Items {
				if !seen[it.Category] {
					seen[it.Category] = true
					cats = append(cats, it.Category)
				}
			}
			for _, cat := range cats {
				fmt.Printf("  [%s]\n", cat)
				for _, it := range c.Items {
					if it.Category != cat {
						continue
					}
					role := it.Role
					if role == "" {
						role = "-"
					}
					fmt.Printf("    · %s（%s）— %s\n", it.Name, licenseLabel(it.License), role)
				}
			}

			fmt.Println("\n平替：")
			for i := range c.Equivalences {
				eq := &c.Equivalences[i]
				from := ""
				if len(eq.From) > 0 {
					from = eq.From[0]
				}
				mark := ""
				if eq.Playbook != nil {
					mark = "（含迁移指导）"
				}
				fmt.Printf("  %s → %s（%s）%s\n", from, eq.To.Name, licenseLabel(eq.To.License), mark)
			}
			fmt.Println()
			return nil
		},
	}
}

func newToollistExportCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "export [id]",
		Short: "把软件清单导出为 YAML（可分享）",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := toollist.DefaultNameFor(runtime.GOOS)
			if len(args) > 0 {
				id = args[0]
			}
			c, err := toollist.LoadBuiltinNamedFor(id, runtime.GOOS)
			if err != nil {
				return err
			}
			path := out
			if path == "" {
				path = c.ID + ".yaml"
			}
			if err := toollist.SaveFile(c, path); err != nil {
				return err
			}
			fmt.Printf("\n已导出：%s\n", path)
			fmt.Printf("分享给别人后，他们可运行：\n  sis audit --catalog %s\n\n", path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "输出文件路径（默认 <id>.yaml）")
	return cmd
}

func newToollistVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify [id]",
		Short: "检查软件清单里各项资料的「新鲜度」（是否需要复核）",
		Long: `逐项显示距今多久、是否需要复核。超过 1 年未复核会标为「需复核」，
用于维护「免费版够用」这类会随时间变化的判断。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := toollist.DefaultNameFor(runtime.GOOS)
			if len(args) > 0 {
				id = args[0]
			}
			c, err := toollist.LoadBuiltinNamedFor(id, runtime.GOOS)
			if err != nil {
				return err
			}
			if f, _ := cmd.Flags().GetString("format"); f == "silent" {
				return nil
			}
			now := time.Now()
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "项目\t类型\t状态\t距上次复核")
			stale, unknown := 0, 0
			for _, it := range c.Items {
				st, days := toollist.Staleness(it.VerifiedAt, now)
				if st == toollist.FreshnessStale {
					stale++
				}
				if st == toollist.FreshnessUnknown {
					unknown++
				}
				fmt.Fprintf(tw, "%s\t条目\t%s\t%s\n", it.Name, freshnessLabel(st), ageStr(days))
			}
			for i := range c.Equivalences {
				eq := &c.Equivalences[i]
				from := ""
				if len(eq.From) > 0 {
					from = eq.From[0]
				}
				st, days := toollist.Staleness(eq.VerifiedAt, now)
				if st == toollist.FreshnessStale {
					stale++
				}
				if st == toollist.FreshnessUnknown {
					unknown++
				}
				fmt.Fprintf(tw, "%s → %s\t平替\t%s\t%s\n", from, eq.To.Name, freshnessLabel(st), ageStr(days))
			}
			tw.Flush()
			fmt.Printf("\n共 %d 条目 + %d 平替；需复核 %d 项，未知 %d 项。\n", len(c.Items), len(c.Equivalences), stale, unknown)
			return nil
		},
	}
}

func freshnessLabel(st string) string {
	switch st {
	case toollist.FreshnessFresh:
		return "新鲜"
	case toollist.FreshnessAging:
		return "较久"
	case toollist.FreshnessStale:
		return "需复核"
	default:
		return "未知"
	}
}

func ageStr(days int) string {
	if days < 0 {
		return "—"
	}
	return fmt.Sprintf("%d 天前", days)
}
