package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/gui"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/spf13/cobra"
)

func newGUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gui",
		Short: "启动本地图形界面（浏览器窗口）",
		Long: `启动 SwiftInstall 的本地图形界面，在浏览器窗口中浏览清单、
勾选软件并一键安装或卸载，安装状态与进度实时更新。

界面默认只监听本机回环地址（127.0.0.1），不会暴露到网络。
使用 --lan 可监听所有网卡（0.0.0.0），供同一局域网内的设备预览；
此时安装/卸载会真实修改本机软件，请仅在可信网络使用。按 Ctrl+C 关闭。`,
		RunE: runGUI,
	}

	cmd.Flags().Bool("no-open", false, "启动后不自动打开浏览器")
	cmd.Flags().Bool("lan", false, "监听所有网卡（0.0.0.0），供同一局域网内的设备预览/访问")
	cmd.Flags().Int("port", 0, "监听端口（默认 0 = 随机端口）")
	cmd.Flags().String("proxy", "", "使用指定代理 (例: http://127.0.0.1:10809)")
	cmd.Flags().Int("retry", 0, "失败重试次数")
	cmd.Flags().Int("retry-delay", 3, "重试间隔秒数")

	return cmd
}

func runGUI(cmd *cobra.Command, args []string) error {
	manifestPath, err := manifestPathFrom(cmd)
	if err != nil {
		return err
	}

	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("解析清单文件失败: %w", err)
	}
	if err := manifest.Validate(m); err != nil {
		return fmt.Errorf("清单文件校验失败: %w", err)
	}

	be, err := backend.ForCurrentOS()
	if err != nil {
		return err
	}
	if err := be.Detect(); err != nil {
		return err
	}

	proxy, _ := cmd.Flags().GetString("proxy")
	retryCount, _ := cmd.Flags().GetInt("retry")
	retryDelay, _ := cmd.Flags().GetInt("retry-delay")
	if m.Settings.Proxy != "" && !cmd.Flags().Changed("proxy") {
		proxy = m.Settings.Proxy
	}
	if m.Settings.RetryCount > 0 && !cmd.Flags().Changed("retry") {
		retryCount = m.Settings.RetryCount
	}
	if m.Settings.RetryDelay > 0 && !cmd.Flags().Changed("retry-delay") {
		retryDelay = m.Settings.RetryDelay
	}

	open, _ := cmd.Flags().GetBool("no-open")
	lan, _ := cmd.Flags().GetBool("lan")
	port, _ := cmd.Flags().GetInt("port")

	// Loopback by default; --lan binds all interfaces for LAN preview.
	host := "127.0.0.1"
	if lan {
		host = "0.0.0.0"
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return gui.Run(ctx, gui.Options{
		Manifest:     m,
		Source:       manifestPath,
		Backend:      be,
		Proxy:        proxy,
		RetryCount:   retryCount,
		RetryDelay:   retryDelay,
		SkipExisting: true,
		Addr:         addr,
		Open:         !open,
	})
}
