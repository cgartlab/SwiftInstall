package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/mirror"
	"github.com/spf13/cobra"
)

func newMirrorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "配置国内镜像加速（Homebrew / npm / pip / GitHub）",
		Long: `在使用 sis 安装软件前，先搞定国内网络环境：一键配置 Homebrew（中科大）、
npm（npmmirror）、pip（清华）镜像，以及 GitHub 下载加速。配置干净可回滚——
写一个受管理的 env 文件并在 shell rc 里用标记块引用，不会重复追加。

  sis mirror apply     配置全部镜像（幂等，可重复执行）
  sis mirror show      查看当前镜像状态
  sis mirror restore   恢复为默认源`,
	}
	cmd.AddCommand(
		&cobra.Command{Use: "apply", Short: "配置国内镜像", RunE: func(cmd *cobra.Command, args []string) error {
			done, err := mirror.Apply()
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "\n  ✓ 镜像已配置：")
			for _, d := range done {
				fmt.Fprintf(os.Stderr, "    %s\n", d)
			}
			fmt.Fprintln(os.Stderr, "\n  新开终端或 `source ~/.config/sis/mirror.env` 生效。")
			return nil
		}},
		&cobra.Command{Use: "show", Short: "查看当前镜像状态", RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := mirror.Show()
			if err != nil {
				return err
			}
			format, _ := cmd.Flags().GetString("format")
			if format == "silent" {
				return nil
			}
			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(map[string]any{"lines": lines})
			}
			fmt.Fprintln(os.Stderr)
			for _, l := range lines {
				fmt.Fprintf(os.Stderr, "  %s\n", l)
			}
			fmt.Fprintln(os.Stderr)
			return nil
		}},
		&cobra.Command{Use: "restore", Short: "恢复默认源", RunE: func(cmd *cobra.Command, args []string) error {
			done, err := mirror.Restore()
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "\n  ✓ 已恢复默认源：")
			for _, d := range done {
				fmt.Fprintf(os.Stderr, "    %s\n", d)
			}
			return nil
		}},
	)
	return cmd
}
