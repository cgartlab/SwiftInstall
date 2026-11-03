package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/cgartlab/SwiftInstall/internal/preset"
	"github.com/spf13/cobra"
)

func newPresetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preset",
		Short: "管理保存的软件清单预设",
		Long: `保存、列出、查看与删除软件清单预设。预设是按你的习惯定制的清单，
存在系统配置目录（macOS: ~/Library/Application Support/sis/presets）。

可用 ` + "`sis -p/--preset <名称> install|list|status|gui|tui`" + ` 直接加载某个预设。

示例:
  sis preset save work -f macOS/packages.txt   # 把当前清单另存为预设 work
  sis preset list
  sis preset show work
  sis preset delete work
  sis install --preset work                    # 用 work 预设安装`,
	}

	cmd.AddCommand(newPresetListCmd())
	cmd.AddCommand(newPresetSaveCmd())
	cmd.AddCommand(newPresetShowCmd())
	cmd.AddCommand(newPresetDeleteCmd())

	return cmd
}

func newPresetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出所有预设",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := preset.List()
			if err != nil {
				return err
			}
			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				out, _ := json.MarshalIndent(map[string]any{"presets": names}, "", "  ")
				fmt.Println(string(out))
				return nil
			}
			if len(names) == 0 {
				fmt.Fprintln(os.Stderr, "  (还没有保存的预设)")
				return nil
			}
			fmt.Fprintf(os.Stderr, "\n  已保存 %d 个预设：\n\n", len(names))
			for _, n := range names {
				fmt.Fprintf(os.Stderr, "    %s\n", n)
			}
			fmt.Fprintln(os.Stderr)
			return nil
		},
	}
}

func newPresetSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <名称>",
		Short: "把当前清单保存为预设",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			path, err := manifestPathFrom(cmd)
			if err != nil {
				return err
			}
			m, err := manifest.ParseManifest(path)
			if err != nil {
				return fmt.Errorf("解析清单失败: %w", err)
			}
			if err := preset.Save(name, m); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "  ✓ 已保存预设 “%s”（%d 个包）\n", name, len(m.Packages))
			return nil
		},
	}
	return cmd
}

func newPresetShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <名称>",
		Short: "显示某个预设的内容",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			m, err := preset.Load(name)
			if err != nil {
				return err
			}
			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				out, _ := json.MarshalIndent(m, "", "  ")
				fmt.Println(string(out))
				return nil
			}
			fmt.Fprintf(os.Stderr, "\n  预设 “%s” — %d 个包：\n\n", name, len(m.Packages))
			for _, p := range m.Packages {
				kind := p.Kind
				if kind == "" {
					kind = "auto"
				}
				fmt.Fprintf(os.Stderr, "    %-28s (%s)\n", p.ID, kind)
			}
			fmt.Fprintln(os.Stderr)
			return nil
		},
	}
}

func newPresetDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <名称>",
		Short: "删除一个预设",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := preset.Delete(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "  ✓ 已删除预设 “%s”\n", args[0])
			return nil
		},
	}
}
