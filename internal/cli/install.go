package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "从清单文件批量安装软件",
		Long: `从 YAML 或 TXT 清单文件批量安装所有软件包。

单个包安装失败不会中断整个批次，所有错误会在最后汇总报告。
支持自动跳过已安装软件、失败重试、代理加速等功能。

示例:
  sis install                    使用默认清单安装
  sis install -f my_apps.yaml    指定清单文件
  sis install --dry-run          仅预览，不实际安装
  sis install --skip-existing    跳过已安装的软件`,
		RunE: runInstall,
	}

	cmd.Flags().BoolP("dry-run", "n", false, "仅预览安装计划，不实际执行")
	cmd.Flags().Bool("skip-existing", false, "跳过已安装的软件")
	cmd.Flags().String("proxy", "", "使用指定代理 (例: http://127.0.0.1:10809)")
	cmd.Flags().Int("retry", 0, "失败重试次数 (默认不重试)")
	cmd.Flags().Int("retry-delay", 3, "重试间隔秒数")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Resolve manifest path
	manifestPath, _ := cmd.Flags().GetString("file")
	if manifestPath == "" {
		manifestPath = resolveManifest()
	}
	if manifestPath == "" {
		return fmt.Errorf("未找到清单文件\n\n" +
			"请使用 -f 指定文件路径，或在当前目录创建 sis.yaml / software_list.txt")
	}

	// Parse manifest
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("解析清单文件失败: %w", err)
	}
	if err := manifest.Validate(m); err != nil {
		return fmt.Errorf("清单文件校验失败: %w", err)
	}

	// Detect backend
	be := backend.NewWingetBackend()
	if err := be.Detect(); err != nil {
		return err
	}

	// Build options (CLI flags > manifest settings > defaults)
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	skipExisting, _ := cmd.Flags().GetBool("skip-existing")
	proxy, _ := cmd.Flags().GetString("proxy")
	retryCount, _ := cmd.Flags().GetInt("retry")
	retryDelay, _ := cmd.Flags().GetInt("retry-delay")

	// Manifest settings as fallback
	if !cmd.Flags().Changed("skip-existing") && m.Settings.SkipExisting {
		skipExisting = true
	}
	if !cmd.Flags().Changed("proxy") && m.Settings.Proxy != "" {
		proxy = m.Settings.Proxy
	}
	if !cmd.Flags().Changed("retry") && m.Settings.RetryCount > 0 {
		retryCount = m.Settings.RetryCount
	}
	if !cmd.Flags().Changed("retry-delay") && m.Settings.RetryDelay > 0 {
		retryDelay = m.Settings.RetryDelay
	}

	opts := engine.Options{
		DryRun:       dryRun,
		SkipExisting: skipExisting,
		Proxy:        proxy,
		RetryCount:   retryCount,
		RetryDelay:   retryDelay,
	}

	// Setup renderer
	renderer := newRenderer(cmd)

	// Show dry-run notice
	if dryRun {
		fmt.Fprintf(os.Stderr, "  ⚡ DRY-RUN 模式 — 不会实际安装任何软件\n")
	}

	// Run
	eng := engine.New(be, opts)
	renderer.Header(len(m.Packages), manifestPath, "Installing")

	eng.OnProgress(func(index int, total int, result engine.Result) {
		renderer.Progress(index, total, result.Package, result)
	})

	summary, err := eng.Install(context.Background(), m)
	if err != nil {
		return err
	}

	renderer.Summary(summary)

	if summary.Failed > 0 {
		return fmt.Errorf("%d 个软件包安装失败", summary.Failed)
	}
	return nil
}
