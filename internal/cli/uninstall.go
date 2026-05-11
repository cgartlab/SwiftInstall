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

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "从清单文件批量卸载软件",
		Long: `根据清单文件批量卸载已安装的软件包。

未安装的软件会自动跳过。需要 --yes 确认才会执行卸载。

示例:
  sis uninstall -f apps.yaml --yes     确认卸载
  sis uninstall --dry-run              仅预览，不实际卸载`,
		RunE: runUninstall,
	}

	cmd.Flags().BoolP("dry-run", "n", false, "仅预览卸载计划，不实际执行")
	cmd.Flags().Bool("yes", false, "确认执行卸载（必须指定）")

	return cmd
}

func runUninstall(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	confirmed, _ := cmd.Flags().GetBool("yes")

	if !dryRun && !confirmed {
		return fmt.Errorf("卸载操作需要 --yes 确认\n\n" +
			"使用 --dry-run 先预览卸载计划，确认后再加 --yes 执行")
	}

	// Resolve manifest
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
	if err := manifest.Validate(m); err != nil {
		return fmt.Errorf("清单文件校验失败: %w", err)
	}

	// Detect backend
	be := backend.NewWingetBackend()
	if err := be.Detect(); err != nil {
		return err
	}

	opts := engine.Options{DryRun: dryRun}
	renderer := newRenderer(cmd)

	if dryRun {
		fmt.Fprintf(os.Stderr, "  ⚡ DRY-RUN 模式 — 不会实际卸载任何软件\n")
	}

	eng := engine.New(be, opts)
	renderer.Header(len(m.Packages), manifestPath, "Uninstalling")

	eng.OnProgress(func(index int, total int, result engine.Result) {
		renderer.Progress(index, total, result.Package, result)
	})

	summary, err := eng.Uninstall(context.Background(), m)
	if err != nil {
		return err
	}

	renderer.Summary(summary)

	if summary.Failed > 0 {
		return fmt.Errorf("%d 个软件包卸载失败", summary.Failed)
	}
	return nil
}
