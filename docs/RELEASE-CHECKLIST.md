# 发布就绪清单（Release Readiness）

> 目的：在提交/发布前，用可核验的证据确认"这个本地版本可以发"。最后更新：2026-11（版本 0.3.0）。

## 一、当前状态

- 版本：`VERSION = 0.3.0`，与 `CHANGELOG.md` 的 `## [0.3.0] - 2026-11-05` 一致；`Unreleased` 为空。
- **尚未提交**：改动仍在本地工作区（这是唯一未完成项，需作者授权提交/开 PR）。
- 安装脚本：`install.sh` / `install.ps1` 仓库指向 `cgartlab/SwiftInstall`，支持 `SIS_VERSION` / `SIS_MIRROR`，macOS 自动检测架构、可回退 `~/bin`。

## 二、回归结果（本轮实测）

| 项目 | 结果 |
|---|---|
| `go build ./...` / `go vet ./...` / `gofmt -l` | ✅ 通过 / 干净 |
| `go test -race ./...` | ✅ 全部包通过 |
| CLI 全命令冒烟 | **23 / 23**（version/help/doctor/audit×3/toollist×4/guide×2/list/status/preset/mirror/llm/search + 4 条预期失败路径） |
| GUI 路由 | **12 / 12**（`/`、`/audit`、`/toollists`、`/swap`、`/health`、`/api/health`、`/api/audit`、`/api/toollists`、`/api/toollist`、`/api/toollist/export`、`/api/explain`(400)、`/cmdk.js`） |
| `sis version` | `sis 0.3.0` |
| `sis doctor` | Homebrew ✓ · 扫描 233 · 5 份清单 100 项「新鲜」· LLM 未配置（可选） |

> 注：`/audit.html` 直接访问返回 200 属正常（静态资源可直连），不是缺陷。

## 三、发布步骤（待授权后执行）

1. **提交 & PR**：
   ```bash
   git checkout -b feat/product-v0.3
   git add -A && git commit -m "feat: 产品化升级 —— 体检/软件清单/迁移指导/AI/自检 + 交互与设计系统 (v0.3.0)"
   git push -u origin feat/product-v0.3
   # 在 GitHub 开 PR；CI（ci.yml）跑 build/vet/test
   ```
2. **打 tag 触发发布**（合并后）：
   ```bash
   git tag v0.3.0 && git push origin v0.3.0
   ```
   `release.yml` 矩阵构建 `darwin/arm64·amd64`、`windows/amd64`、`linux/amd64`，并校验 `sis version` 含 tag。
3. **发布后验证**：下载各平台产物运行 `sis version`；`install.sh` / `install.ps1` 一键安装可用。

## 四、已知边界（发布说明应提及）

- **LLM 功能可选**：`sis explain` / GUI 的 AI 查询需 `sis llm set` 配置；未配置时有友好提示，不影响其它功能。
- **AI 端到端**：本机以本地 mock 覆盖了 llm→advice→GUI 全链路；需在有 key 的环境做一次真机验收。
- **交互层自动化**：TUI/GUI 以真机冒烟为主，未做 UI 自动化；核心逻辑与 handler 有单测。
- **目录规模**：5 份精选清单共 100 项，主打"深度与可信"，非"全量目录"。
