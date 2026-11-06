package cli

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/audit"
	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/llm"
	"github.com/cgartlab/SwiftInstall/internal/toollist"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "自检：包管理器 / 已装扫描 / 软件清单新鲜度 / LLM 配置",
		Long: `一次性检查运行环境是否正常，帮助你自行诊断问题：

  · 平台与包管理器（Homebrew / winget）是否可用
  · 是否能否扫描到本机已安装的软件
  · 内置软件清单的数量与资料新鲜度（有几项需要复核）
  · AI / LLM 是否已配置`,
		RunE: runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Printf("\nSwiftInstall 自检（%s/%s）\n\n", runtime.GOOS, runtime.GOARCH)

	// 1) 包管理器
	be, err := backend.ForCurrentOS()
	switch {
	case err != nil:
		fmt.Printf("  包管理器        ✗ %v\n", err)
	case be.Detect() != nil:
		fmt.Printf("  包管理器 %-8s ✗ %v\n", be.Name(), be.Detect())
	default:
		fmt.Printf("  包管理器 %-8s ✓\n", be.Name())
	}

	// 2) 已装扫描
	if names, err := (audit.SystemLister{}).Installed(context.Background()); err != nil {
		fmt.Printf("  已装扫描        ✗ %v\n", err)
	} else {
		fmt.Printf("  已装扫描        ✓ 识别到 %d 个应用/包\n", len(names))
	}

	// 3) 软件清单 + 新鲜度
	if names, err := toollist.EmbeddedNames(); err != nil {
		fmt.Printf("  软件清单        ✗ %v\n", err)
	} else {
		total, fresh, aging, stale := summarizeFreshness(names, time.Now())
		fmt.Printf("  软件清单        ✓ %d 份，共 %d 项（含平替）\n", len(names), total)
		fmt.Printf("    新鲜 %d · 较久 %d · 需复核 %d\n", fresh, aging, stale)
		if stale > 0 {
			fmt.Printf("    提示：有 %d 项资料超过一年未复核，可用 `sis toollist verify` 查看。\n", stale)
		}
	}

	// 4) LLM
	if cfg, _ := llm.Load(); cfg.Configured() {
		fmt.Printf("  AI / LLM        ✓ 已配置（%s）\n", cfg.Model)
	} else {
		fmt.Printf("  AI / LLM        ○ 未配置（`sis llm set` 可启用 AI 补全）\n")
	}

	fmt.Println()
	return nil
}

// summarizeFreshness counts items + equivalences across the named toollists by
// freshness state. Kept pure for testing.
func summarizeFreshness(names []string, now time.Time) (total, fresh, aging, stale int) {
	for _, n := range names {
		c, err := toollist.LoadBuiltinNamed(n)
		if err != nil {
			continue
		}
		add := func(verifiedAt string) {
			total++
			switch st, _ := toollist.Staleness(verifiedAt, now); st {
			case toollist.FreshnessFresh:
				fresh++
			case toollist.FreshnessAging:
				aging++
			case toollist.FreshnessStale:
				stale++
			}
		}
		for _, it := range c.Items {
			add(it.VerifiedAt)
		}
		for i := range c.Equivalences {
			add(c.Equivalences[i].VerifiedAt)
		}
	}
	return
}
