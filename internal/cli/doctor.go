package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	format, _ := cmd.Flags().GetString("format")
	if format == "silent" {
		return nil
	}

	pmName := ""
	var pmErr error
	if be, err := backend.ForCurrentOS(); err != nil {
		pmErr = err
	} else {
		pmName = be.Name()
		if derr := be.Detect(); derr != nil {
			pmErr = derr
		}
	}

	instCount := 0
	var instErr error
	if names, err := (audit.SystemLister{}).Installed(context.Background()); err != nil {
		instErr = err
	} else {
		instCount = len(names)
	}

	listNames, listErr := toollist.EmbeddedNames()
	total, fresh, aging, stale := 0, 0, 0, 0
	if listErr == nil {
		total, fresh, aging, stale = summarizeFreshness(listNames, time.Now())
	}
	cfg, _ := llm.Load()

	if format == "json" {
		pm := map[string]any{"name": pmName, "ok": pmErr == nil}
		if pmErr != nil {
			pm["error"] = pmErr.Error()
		}
		inst := map[string]any{"ok": instErr == nil}
		if instErr != nil {
			inst["error"] = instErr.Error()
		} else {
			inst["count"] = instCount
		}
		var tl map[string]any
		if listErr != nil {
			tl = map[string]any{"ok": false, "error": listErr.Error()}
		} else {
			tl = map[string]any{"ok": true, "lists": len(listNames), "items": total, "fresh": fresh, "aging": aging, "stale": stale}
		}
		out := map[string]any{
			"os": runtime.GOOS, "arch": runtime.GOARCH,
			"package_manager": pm, "installed": inst, "toollists": tl,
			"llm": map[string]any{"configured": cfg.Configured(), "model": cfg.Model},
		}
		return json.NewEncoder(os.Stdout).Encode(out)
	}

	fmt.Printf("\nSwiftInstall 自检（%s/%s）\n\n", runtime.GOOS, runtime.GOARCH)
	if pmErr != nil {
		fmt.Printf("  包管理器        ✗ %v\n", pmErr)
	} else {
		fmt.Printf("  包管理器 %-8s ✓\n", pmName)
	}
	if instErr != nil {
		fmt.Printf("  已装扫描        ✗ %v\n", instErr)
	} else {
		fmt.Printf("  已装扫描        ✓ 识别到 %d 个应用/包\n", instCount)
	}
	if listErr != nil {
		fmt.Printf("  软件清单        ✗ %v\n", listErr)
	} else {
		fmt.Printf("  软件清单        ✓ %d 份，共 %d 项（含平替）\n", len(listNames), total)
		fmt.Printf("    新鲜 %d · 较久 %d · 需复核 %d\n", fresh, aging, stale)
		if stale > 0 {
			fmt.Printf("    提示：有 %d 项资料超过一年未复核，可用 `sis toollist verify` 查看。\n", stale)
		}
	}
	if cfg.Configured() {
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
