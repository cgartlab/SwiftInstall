package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/llm"
	"github.com/spf13/cobra"
)

func newLLMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llm",
		Short: "配置 AI / LLM（OpenAI 兼容，用于补全软件信息等）",
		Long: `配置一个 OpenAI 兼容的 LLM 端点，供 sis 的 AI 能力调用（如为信息不全的
软件或字体生成简介）。兼容 OpenAI、DeepSeek、GLM、Kimi、Ollama、magpie 网关等。

配置存在 ~/Library/Application Support/sis/llm.json（macOS）。

  sis llm set --base-url https://api.deepseek.com/v1 --api-key sk-... --model deepseek-chat
  sis llm show
  sis llm test "用一句话介绍 git"`,
	}

	set := &cobra.Command{
		Use:   "set",
		Short: "设置 LLM 配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := llm.Load()
			if cmd.Flags().Changed("base-url") {
				cfg.BaseURL, _ = cmd.Flags().GetString("base-url")
			}
			if cmd.Flags().Changed("api-key") {
				cfg.APIKey, _ = cmd.Flags().GetString("api-key")
			}
			if cmd.Flags().Changed("model") {
				cfg.Model, _ = cmd.Flags().GetString("model")
			}
			if err := llm.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "\n  ✓ LLM 配置已保存。")
			printCfg(cfg)
			return nil
		},
	}
	set.Flags().String("base-url", "", "API base URL（如 https://api.openai.com/v1）")
	set.Flags().String("api-key", "", "API key")
	set.Flags().String("model", "", "模型名（如 gpt-4o-mini）")

	show := &cobra.Command{
		Use:   "show",
		Short: "查看当前 LLM 配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := llm.Load()
			if err != nil {
				return err
			}
			printCfg(cfg)
			return nil
		},
	}

	test := &cobra.Command{
		Use:   "test [prompt]",
		Short: "测试 LLM 连通性",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := llm.Load()
			if err != nil {
				return err
			}
			if !cfg.Configured() {
				return fmt.Errorf("LLM 未配置，先用 sis llm set 设置")
			}
			prompt := "用一句话介绍 git"
			if len(args) > 0 {
				prompt = args[0]
			}
			fmt.Fprintf(os.Stderr, "\n  调用 %s (%s) …\n", cfg.BaseURL, cfg.Model)
			cl := llm.NewClient(cfg)
			out, err := cl.Complete(context.Background(), prompt)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "  回复：%s\n\n", out)
			return nil
		},
	}

	cmd.AddCommand(set, show, test)
	return cmd
}

func printCfg(cfg *llm.Config) {
	fmt.Fprintln(os.Stderr)
	if !cfg.Configured() {
		fmt.Fprintln(os.Stderr, "  LLM：未配置（用 sis llm set 设置）")
		return
	}
	fmt.Fprintf(os.Stderr, "  base-url: %s\n", cfg.BaseURL)
	fmt.Fprintf(os.Stderr, "  model:    %s\n", cfg.Model)
	fmt.Fprintf(os.Stderr, "  api-key:  %s\n", cfg.MaskedKey())
	fmt.Fprintln(os.Stderr)
}
