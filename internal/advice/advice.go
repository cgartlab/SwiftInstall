// Package advice turns a bare app name into a plain-language explanation and a
// free/open-alternative suggestion using an LLM. It is the "long tail" AI
// complement to the curated toollists: anything not hand-verified can still get
// a first-pass answer, clearly marked as AI-generated and to be verified.
package advice

import (
	"context"
	"fmt"
	"strings"
)

// Chatter is the seam used to call a model; *llm.Client implements it.
type Chatter interface {
	Chat(ctx context.Context, system, user string) (string, error)
}

// SystemPrompt is the fixed role instruction for the explainer.
const SystemPrompt = "你是一个熟悉 macOS 软件与免费/开源替代方案的助手。回答用简体中文，客观、简洁、基于事实；不确定就直说，不要编造。"

// Explain asks the model to describe an app and propose free/open alternatives.
func Explain(ctx context.Context, ch Chatter, app string) (string, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return "", fmt.Errorf("请提供应用名，例如 sis explain \"CleanShot X\"")
	}
	if ch == nil {
		return "", fmt.Errorf("未提供 LLM 客户端")
	}
	user := fmt.Sprintf(`应用：%s

请分三部分回答：
1) 一句话说明它是什么、主要用途；
2) 是否有成熟的免费或开源平替（若有，列出 1–2 个，并说明免费版是否够用；若无，说明原因）；
3) 迁移或替换时的注意事项。`, app)
	out, err := ch.Chat(ctx, SystemPrompt, user)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
