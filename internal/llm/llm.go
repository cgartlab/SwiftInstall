// Package llm provides a minimal OpenAI-compatible chat client and a persisted
// configuration (base URL, API key, model) stored under the sis config dir.
// It is the shared seam every "AI" feature in sis calls through, so future
// surfaces (descriptions, categorisation, suggestions) reuse one config.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config is the persisted LLM configuration.
type Config struct {
	BaseURL string `json:"base_url"` // e.g. https://api.openai.com/v1 or http://127.0.0.1:3425/v1 (magpie)
	APIKey  string `json:"api_key"`
	Model   string `json:"model"` // e.g. gpt-4o-mini, deepseek-chat, glm-4-flash
}

// ConfigPath returns the path to llm.json under the sis config dir.
func ConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		base = filepath.Join(os.Getenv("HOME"), ".config")
	}
	dir := filepath.Join(base, "sis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "llm.json"), nil
}

// Load reads the config; returns a zero config (not an error) if absent.
func Load() (*Config, error) {
	p, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save writes the config.
func Save(c *Config) error {
	p, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// Configured reports whether the config has enough to call a model.
func (c *Config) Configured() bool {
	return c != nil && c.BaseURL != "" && c.APIKey != "" && c.Model != ""
}

// Masked returns the API key masked for display.
func (c *Config) MaskedKey() string {
	k := c.APIKey
	if len(k) <= 8 {
		return strings.Repeat("•", len(k))
	}
	return k[:4] + strings.Repeat("•", len(k)-8) + k[len(k)-4:]
}

// Client is an OpenAI-compatible chat client.
type Client struct {
	cfg  *Config
	http *http.Client
}

func NewClient(c *Config) *Client {
	return &Client{cfg: c, http: &http.Client{Timeout: 40 * time.Second}}
}

// Chat sends a system + user message and returns the assistant reply text.
func (cl *Client) Chat(ctx context.Context, system, user string) (string, error) {
	if cl.cfg == nil || !cl.cfg.Configured() {
		return "", fmt.Errorf("LLM 未配置（用 `sis llm set` 设置 base-url / api-key / model）")
	}
	base := strings.TrimRight(cl.cfg.BaseURL, "/")
	url := base + "/chat/completions"

	body := map[string]any{
		"model":       cl.cfg.Model,
		"temperature": 0.2,
		"stream":      false,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cl.cfg.APIKey)

	resp, err := cl.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 LLM: %w", err)
	}
	defer resp.Body.Close()
	rbody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM 返回 HTTP %d: %s", resp.StatusCode, truncate(string(rbody), 200))
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rbody, &out); err != nil {
		return "", fmt.Errorf("解析 LLM 响应: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空响应")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// Complete returns a short completion for a single user prompt with a generic
// system instruction ("你是简洁准确的软件信息助手").
func (cl *Client) Complete(ctx context.Context, prompt string) (string, error) {
	return cl.Chat(ctx, "你是简洁准确的软件信息助手，用简体中文回答，不要多余废话。", prompt)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
