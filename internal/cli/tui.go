package cli

import (
	"fmt"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/cgartlab/SwiftInstall/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "启动交互式终端界面",
		Long: `启动全屏交互式终端界面，用键盘浏览、勾选并安装/卸载软件包。

界面会异步探测每个软件包的安装状态，支持按分类分组浏览、
批量选择、安装与卸载，并实时展示进度与最终汇总。

常用按键:
  ↑/↓ 或 j/k    移动光标
  space         勾选/取消当前软件
  a             全选 / 全不选
  i             安装所选软件
  u             卸载所选软件
  r             重新探测安装状态
  q / ctrl+c    退出

示例:
  sis tui                    使用默认清单启动
  sis tui -f my_apps.yaml    指定清单文件
  sis tui --proxy http://127.0.0.1:10809`,
		RunE: runTUI,
	}

	cmd.Flags().String("proxy", "", "使用指定代理 (例: http://127.0.0.1:10809)")
	cmd.Flags().Int("retry", 0, "失败重试次数 (默认不重试)")
	cmd.Flags().Int("retry-delay", 3, "重试间隔秒数")

	return cmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Resolve manifest path (--preset > --file > auto-discovery).
	manifestPath, err := manifestPathFrom(cmd)
	if err != nil {
		return err
	}

	// Parse and validate.
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("解析清单文件失败: %w", err)
	}
	if err := manifest.Validate(m); err != nil {
		return fmt.Errorf("清单文件校验失败: %w", err)
	}

	// Select and detect the backend for this OS.
	be, err := backend.ForCurrentOS()
	if err != nil {
		return err
	}
	if err := be.Detect(); err != nil {
		return err
	}

	// Effective settings: flags > manifest settings > defaults.
	proxy, _ := cmd.Flags().GetString("proxy")
	retryCount, _ := cmd.Flags().GetInt("retry")
	retryDelay, _ := cmd.Flags().GetInt("retry-delay")

	if !cmd.Flags().Changed("proxy") && m.Settings.Proxy != "" {
		proxy = m.Settings.Proxy
	}
	if !cmd.Flags().Changed("retry") && m.Settings.RetryCount > 0 {
		retryCount = m.Settings.RetryCount
	}
	if !cmd.Flags().Changed("retry-delay") && m.Settings.RetryDelay > 0 {
		retryDelay = m.Settings.RetryDelay
	}

	noColor, _ := cmd.Flags().GetBool("no-color")

	return tui.Run(tui.Options{
		Manifest:   m,
		Source:     manifestPath,
		Backend:    be,
		Proxy:      proxy,
		RetryCount: retryCount,
		RetryDelay: retryDelay,
		NoColor:    noColor,
	})
}
