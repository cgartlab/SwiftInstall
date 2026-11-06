# 测试说明 — SwiftInstall

> 目标：核心功能"坚如磐石、稳定可靠"。本文件记录测试范围、运行方式与当前状态。
> 最后更新：2026-11

## 如何运行

```bash
go build ./...            # 编译
go vet ./...              # 静态检查
go test ./...             # 单元测试
go test -race ./...       # 竞态检测（推荐）
gofmt -l internal cmd     # 格式检查（应为空）
```

CI（`.github/workflows/ci.yml`）在上面的 `build + vet + test` 基础上运行。

## 测试矩阵

| 包 | 覆盖的核心行为 | 覆盖率 |
|---|---|---|
| `internal/audit` | `.app` 目录扫描（含嵌套、跳过包内 helper）、名称归一（含 `.app` 后缀、CJK）、替换/保留/补充判定、目标已装抑制、brew token 命中、分类与授权透传、空清单/空目录 | 87.3% |
| `internal/toollist` | 内置软件清单加载、文件加载、缺 id / 坏 YAML 报错、内置数据完整性（类别/授权/来源/日期、条目数 ≥20、平替表结构） | 85.7% |
| `internal/manifest` | YAML/TXT 内容嗅探、去重、校验、可选包、BOM、kind 字段、错误路径 | 88.7% |
| `internal/engine` | 跳过已装、dry-run、重试/耗尽、可选失败转跳过、kind/proxy 透传、上下文取消、进度回调、去重、卸载 | 85.1% |
| `internal/backend` | Homebrew 命令参数、cask/formula 回退、代理 env、IsInstalled 真/假/错误、卸载 | 15.1%（大量 exec 分支为集成路径） |
| `internal/preset` | 保存/加载/列表/删除、非法名、产物为合法 manifest | 75.0% |
| `internal/mirror` | apply/restore 幂等、env 内容正确 | 63.3% |
| `internal/gui` | `/api/audit`、`/api/state`、`/api/status`、`/api/run` 拒绝坏输入、audit 页面内嵌 | 10.6%（HTTP/SSE 服务层，另以真机冒烟覆盖） |
| `internal/tui` | 选择/移动/分组渲染/进度/退出等 | 已覆盖主要状态机 |

## 真机冒烟（CLI）

对全部子命令做了退出码冒烟，含错误路径：

- 成功：`version` / `--help` / `list` / `status --format json` / `audit`（table+json）/ `search --help` / `preset list` / `mirror show` / `llm show` / `gui --help` / `tui --help` / `install --help` / `uninstall --help`
- 预期失败（退出码 1，信息清晰）：无清单时的 `list`、未知命令、`audit --catalog` 指向不存在文件

## 本次修复

- **`.app` 目录识别**：macOS 的 `.app` 是**目录**，原逻辑只在"文件"分支匹配 `.app`，导致 `/Applications` 扫描读不到任何应用（实际只统计到 `brew list` 的 89 项）。改为按目录识别并跳过包内 helper 后，真机覆盖 **89 → 233**，"已装够用"从 1 → 4。已补测试 `TestSystemLister_TempDir` 防回归。

## 待加强（非核心阻塞）

- `internal/gui`、`internal/backend` 的覆盖率偏低，主要是 HTTP/SSE 服务层与 `brew` 集成路径；如需更高把握，可补 `httptest` 覆盖更多 handler 与 fake 命令。
- `sis tui` / `sis gui` 的交互层以真机冒烟为准，未做 UI 自动化。

## 迁移 Playbook（`sis guide`）

- 数据：平替条目可带 `playbook`（summary / data_fidelity / verified_at / steps[title,detail,source_url]）。
- 覆盖：`toollist` 校验内置 Playbook 完整性（≥3 条、每步 title/detail/source_url 齐全）；`audit` 校验 `FindEquivalence` 命中与 Playbook 透传到推荐项。
- CLI：`sis guide <软件>`、`sis guide --list`（只读，不安装/卸载、不搬数据）。
- GUI：体检报告「可替换」项可展开迁移指导抽屉。

## 软件清单导出 / 分享（`sis toollist`）

- API：`toollist.Marshal` / `SaveFile` / `EmbeddedNames` / `LoadBuiltinNamed`。
- 覆盖：`TestMarshalRoundTrip`（导出→解析，条目/平替/迁移指导不丢）、`TestEmbeddedNames`、`TestLoadBuiltinNamed`；`TestHandleToollistExport`（GUI `/api/toollist/export` 200 + 附件头）。
- 冒烟：`sis toollist export -o f.yaml` → `sis audit --catalog f.yaml` 往返成功。

## 多软件清单（`--toollist` / `/api/toollists`）

- 内置软件清单：`mac-designers`、`developers`。`sis audit` / `sis guide` 用 `--toollist <id>` 选择。
- 覆盖：`TestAllBuiltinsIntegrity`（每个内置软件清单的类别/授权/来源/平替完整性）、`TestHandleToollists`、`TestHandleAudit_ToollistParam`。
- GUI：`/api/toollists` 列表，`/api/audit?toollist=`、`/api/toollist/export?toollist=` 按软件清单切换。

## 软件清单模块（`/toollists`）

- 页面：`web/`... 实际为 `internal/gui/assets/toollists.html`（嵌入式），数据来自 `/api/toollists` 与 `/api/toollist?toollist=`。
- 覆盖：`TestHandleToollist`（单软件清单完整目录 JSON）、`TestToollistsPageEmbedded`（页面已内嵌），`toollists.html` 通过 `node --check`。
- 交互：左栏切换软件清单、条目分类渲染、平替 `<details>` 展开迁移指导（只读，不安装/卸载）。

## GUI 导入软件清单 + 措辞/新手文案

- 接口：`POST /api/toollist/import`（YAML，限量 1MB），导入清单会话内保存（`server.imported`），`/api/toollists` 会列出（标记 `imported`），`/api/audit?toollist=<id>` 可使用。
- 覆盖：`TestHandleToollistImport`（导入→列表→审计→坏 YAML 400）、`TestParse`（导出解析）。
- 文案：面向用户统一使用「软件清单」；清单页/体检页/主界面/README 增加新手引导文案。

## 资料新鲜度（`sis toollist verify` / `Report.stale_count`）

- 计算：`booklist`→`toollist.Staleness(verified_at, now)`，阈值 180/365 天 → 新鲜/较久/需复核；未知日期为 unknown。
- 覆盖：`TestStaleness`（各阈值与未知日期）、`toollist` 包/`audit` 包全绿。
- 出口：CLI `sis toollist verify`；体检报告与 GUI 对 `stale` 项标注「需复核」。

## 术语统一

- `booklist` → `toollist`：包 `internal/toollist`、命令 `sis toollist`、接口 `/api/toollist(s)`、页面 `/toollists`、以及全部文档。

## AI 补全长尾（`sis explain`）

- 包 `internal/advice`：`Explain(ctx, Chatter, app)` 组装 prompt 并返回模型文本；`Chatter` 接口便于注入 fake。
- 覆盖：`TestExplain`（裁剪空白、prompt 含应用名）、`TestExplain_EmptyApp`、`TestExplain_PropagatesError`。
- 未配置 LLM 时 `sis explain` 友好报错并提示 `sis llm set`（不联网、不影响其他功能）。

## GUI 接入 AI（`/api/explain`）

- 处理器：`handleExplain` 读取 `?app=`，走 `internal/advice` + `internal/llm`；缺参 400，未配置 LLM 返回 400 + 配置指引。
- 覆盖：`TestHandleExplain_NoApp`；`toollists.html` 新增 AI 输入/结果区并通过 `node --check`。

## 交互层自动化测试（Round 11）

- GUI handler（httptest）：`handlePresets`、`handlePreset`（缺 name→400）、`handleOutdated`（后端无能力→400）、`handleLLM`（GET），连同此前的 state/status/run/audit/toollist(s)/import/export/explain。
- CLI：`freshnessLabel`/`ageStr`/`joinURLs` 单测；`TestRootRegistersCommands` 校验 root 注册了 audit/guide/toollist/explain/llm 等全部子命令。
- 覆盖率（本轮实测）：gui **24.4%**（↑ from 10.6%）、cli **16.1%**、toollist 84.3%、audit 86.2%、advice 90.0%；`go test -race ./...` 全绿。

## LLM 端到端验证（mock，无需外部 key）

- `internal/llm`：`TestChat_ParsesResponse`（解析+鉴权头）、`TestChat_HTTPError`、`TestChat_EmptyChoices`、`TestComplete`、`TestConfig_ConfiguredAndMask`。
- `internal/advice`：`TestExplain_EndToEnd` 经真实 `llm.Client` 打本地 mock OpenAI 端点。
- `internal/gui`：`TestHandleExplain_Configured` 覆盖 GUI `/api/explain` → advice → llm → mock 全链路（临时 HOME 写入配置）。
- 覆盖率：llm 46.7%（↑ from 0）、advice 90.0%、gui 25.7%。

## 自检命令 `sis doctor`

- 输出四段：包管理器检测、已装扫描数量、3 份清单共 N 项的新鲜度汇总、LLM 配置状态。
- 覆盖：`TestSummarizeFreshness`（固定时间点 → 全新鲜 / 全 stale 两种边界），纯函数可测。

## 体检覆盖「未收录已安装」

- `Report.unmatched_count` / `unmatched_sample`（最多 20 个）：已安装但不在清单中的应用；CLI 与体检页提示可用 `sis explain`。
- 覆盖：`TestRun_Unmatched`（命中清单项不计入、其余计入并取样）。

## 全产品回归冒烟（Round 16）

- CLI：25 条（version/--help/doctor/audit 各种/toollist list·show·verify·export/guide/list/status/preset/mirror/llm + 预期失败路径）全部符合预期。
- GUI：`/audit`、`/toollists`、`/`、`/api/audit`、`/api/toollists`、`/api/toollist`、`/api/toollist/export` 均 200；`/api/explain`（缺参）400。
- JSON：`replaces/adds/keeps` 空集输出为 `[]`（非 null）。

## 路由集成测试（Round 21）

- `gui`: 抽出 `server.routes(fs.FS)`，`TestRoutes` 用 `httptest` 命中 `/`、`/audit`、`/toollists`、`/api/state·status·audit·toollists·toollist·toollist/export·presets`（200），`/api/outdated`、`/api/explain`（400），并 POST `/api/toollist/import`（200）。防止路由改名/注册遗漏回归。
- gui 覆盖率 25.7%→31.6%。

## 清单 → 安装 闭环（Round 23）

- 清单页「设为安装清单」：把 toollist 的条目（brew 令牌 + 分类）保存为预设（`/api/preset?action=save`），主界面即可安装。
- 覆盖：`TestPresetSaveFromToollist`（temp HOME 保存，返回 ok + total）。

## GUI 新鲜度维护视图（Round 24）

- `/api/toollists` 每项含 `fresh/aging/stale/total`；`toollist.FreshnessCounts(c, now)` 纯函数。
- 覆盖：`TestFreshnessCounts`、`TestHandleToollists_Freshness`；gui 覆盖率 37.9%→38.0%。

## /swap 逐条决策页（Round 26）

- 路由 `/swap` 提供 `swap.html`，从 `/api/toollist` 的 `equivalences` 生成平替滑卡；结束可存为安装清单（复用 `/api/preset?action=save`）。
- 覆盖：`TestSwapRoute`（`/swap` 200）；`swap.html` / `toollists.html` 通过 `node --check`。

## 主界面 ⌘K 命令面板（Round 28）

- 内联脚本实现：`⌘K/Ctrl-K` 开合、输入过滤、↑↓ 选择、↵ 执行、Esc 关闭；命令覆盖体检/清单/逐条决策/过滤/更新/刷新。
- 覆盖：`TestCommandPaletteAsset`（index.html 内嵌 `__cmdkOpen`）；调色板脚本通过 `node --check`。

## 体检 Scan→Report 形变（Round 29）

- `/audit`：扫描结束 → `morphToResult()`（扫描器 opacity/scale 过渡后显示结果，统计卡错峰动画）；报告顶部「逐条决策这些平替 →」跳 `/swap?toollist=`。
- `audit.html` 通过 `node --check`；reduced-motion 下降级为直切。

## ⌘K 全站化（Round 30）

- 共享 `internal/gui/assets/cmdk.js`（自注入 CSS/DOM，绑定 ⌘K/Ctrl-K，过滤/↑↓/↵/Esc），由 `/audit`、`/toollists`、`/swap` 引入；`/cmdk.js` 经 FileServer 提供。
- 覆盖：`TestCommandPaletteScript`（内嵌 + `/cmdk.js` 200）；`cmdk.js` 通过 `node --check`。

## 浅色主题视觉 QA（Round 31）

- 静态亮色快照验收 `/swap` 与主界面（`docs/screenshots/gui-swap-light.png`、`gui-main-light.png`）；令牌与层级在亮/暗两套下成立。

## GUI 系统自检页（Round 34）

- `/api/health` 返回平台/包管理器/已装扫描/清单新鲜度/LLM；`/health` 页面渲染（Cabinet 风格），支持复制诊断 JSON；主界面侧栏与 ⌘K 均可进入。
- 覆盖：`TestHealthRoutes`（`/health`、`/api/health` 均 200）；`health.html`/`cmdk.js` 通过 `node --check`。

## 清单浏览页过滤（Round 35）

- `/toollists`：顶部过滤框按名称/用途/理由即时筛选（客户端 DOM 过滤，隐藏空分类与不匹配平替）；`toollists.html` 通过 `node --check`。

## 无障碍硬化（Round 37）

- `cmdk.js` 增加 `@media (prefers-reduced-transparency:reduce)` 实心兜底；主界面顶栏按钮补 `aria-label`；`audit` 的 `#sections`、`toollists` 的 `#body` 加 `aria-live="polite"`。
- `cmdk.js`/`audit.html`/`toollists.html` 通过 `node --check`。

## 动效机会点（Round 41）

- `/audit` 报告分类行错峰入场、`/health` 状态行轻淡入；两类均在 reduced-motion 下直显。`audit.html`/`health.html` 通过 `node --check`。

## 分享卡片页（Round 43）

- `/share?toollist=` 渲染清单海报（标题/受众/两列工具+授权标签/品牌页脚）+「复制文案」；清单页加入口。
- 覆盖：`TestShareRoute`（`/share` 200）；`share.html` 通过 `node --check`；含 Canvas「下载图片」导出（1080pt PNG）。

## 端到端回归脚本（Round 16）

- `bash scripts/regression.sh`：构建二进制 → 跑 CLI 退出码矩阵（version/doctor/audit/toollist/guide/status/mirror/llm/format 校验/参数校验）+ 跨平台构建（windows/linux/darwin）+ 起 GUI 跑路由/E2E（页面 200、未知清单 404、只读 POST 405、`/cmdk.js` 200）。全部通过返回 0。
