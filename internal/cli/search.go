package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <关键词>",
		Short: "在 Homebrew 仓库中搜索软件包",
		Long: `在当前后端的软件仓库中搜索软件包（macOS 上即 Homebrew 全部 formula 与 cask）。
返回名称列表，标注频道与是否已安装。

示例:
  sis search figma
  sis search python --format json`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}
	return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return fmt.Errorf("请提供搜索关键词")
	}

	be, err := backend.ForCurrentOS()
	if err != nil {
		return err
	}
	if err := be.Detect(); err != nil {
		return err
	}

	searcher, ok := be.(backend.Searcher)
	if !ok {
		return fmt.Errorf("当前后端 (%s) 不支持仓库搜索", be.Name())
	}

	hits, err := searcher.Search(context.Background(), query)
	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	noColor, _ := cmd.Flags().GetBool("no-color")

	installed := map[string]bool{}
	if provider, ok := be.(backend.InstalledSetProvider); ok {
		if set, err := provider.InstalledSet(context.Background()); err == nil {
			installed = set
		}
	}

	if format == "silent" {
		return nil
	}
	if format == "json" {
		type hit struct {
			Name      string `json:"name"`
			Kind      string `json:"kind,omitempty"`
			Installed bool   `json:"installed"`
		}
		out := struct {
			Query string `json:"query"`
			Total int    `json:"total"`
			Hits  []hit  `json:"hits"`
		}{Query: query, Total: len(hits), Hits: make([]hit, 0, len(hits))}
		for _, h := range hits {
			out.Hits = append(out.Hits, hit{Name: h.Name, Kind: h.Kind, Installed: installed[h.Name]})
		}
		printJSON(out)
		return nil
	}

	// table
	dim, green, reset := "", "", ""
	if !noColor {
		dim, green, reset = "\033[2m", "\033[32m", "\033[0m"
	}
	fmt.Fprintf(os.Stderr, "\n搜索 “%s” — %d 个结果\n\n", query, len(hits))
	if len(hits) == 0 {
		fmt.Fprintf(os.Stderr, "  (无匹配)\n\n")
		return nil
	}
	// sort: installed first, then name
	sort.SliceStable(hits, func(i, j int) bool {
		ti, tj := installed[hits[i].Name], installed[hits[j].Name]
		if ti != tj {
			return ti
		}
		return hits[i].Name < hits[j].Name
	})
	for _, h := range hits {
		mark := "  "
		if installed[h.Name] {
			mark = green + "✓ " + reset
		}
		kind := h.Kind
		if kind == "" {
			kind = "auto"
		}
		fmt.Fprintf(os.Stderr, "  %s%-28s %s(%s)%s\n", mark, h.Name, dim, kind, reset)
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
