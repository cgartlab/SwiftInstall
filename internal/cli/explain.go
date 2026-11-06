package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/cgartlab/SwiftInstall/internal/advice"
	"github.com/cgartlab/SwiftInstall/internal/llm"
	"github.com/spf13/cobra"
)

func newExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <软件>",
		Short: "用 AI 简单介绍某个软件，并找它的免费/开源平替（可选，需配置 LLM）",
		Long: `当某个软件不在我们人工核验的软件清单里时，可以用 AI 给一个初步答案：
它是什么、有没有成熟的免费/开源平替、迁移要注意什么。

这是 AI 生成的参考，请以官方信息为准；需要先配置一个 OpenAI 兼容端点：

  sis llm set --base-url https://api.deepseek.com/v1 --api-key sk-... --model deepseek-chat
  sis explain "CleanShot X"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := strings.Join(args, " ")
			cfg, err := llm.Load()
			if err != nil {
				return err
			}
			if !cfg.Configured() {
				return fmt.Errorf("此功能需要先配置 LLM：运行 `sis llm set --base-url ... --api-key ... --model ...`（支持 OpenAI / DeepSeek / Ollama / magpie 等）")
			}
			out, err := advice.Explain(context.Background(), llm.NewClient(cfg), app)
			if err != nil {
				return err
			}
			fmt.Printf("\n%s\n\n", out)
			fmt.Println("（以上为 AI 生成，仅供参考，请以官方信息为准。）")
			return nil
		},
	}
}
