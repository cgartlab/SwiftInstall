# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

- **空状态引导**：主界面区分「清单为空」（引导去软件清单挑选）与「过滤无匹配」，附 CTA 按钮，新手更易上手。

- **LLM 接口回归测试**：覆盖 `/api/llm` 保存后 GET 返回脱敏 key（绝不回显明文）、字段往返、非法 JSON→400。

- **列表行 hover 打磨**：hover 底色过渡 120ms；用 `@media (hover:hover)` 门控，触屏不出现粘滞 hover（保留 `:active` 按压反馈）。

- **⌘K 分组标题**：命令面板按「导航 / 动作」分组显示（最近使用仍置顶），信息层级更清晰。

- **端到端回归脚本**：新增 `scripts/regression.sh`（CLI 退出码矩阵 + GUI 路由/E2E，23 项，exit 0=全过）；本轮全线回归 `go test -race ./...` 全绿、脚本 23/23。

- **修正 + 交互**：修正上一轮误给 `/api/preset` 加 GET 守卫（POST 保存预设恢复可用，补全量测试通过）；`/swap` 松手未达阈值时回弹带轻微弹簧（cubic-bezier 过冲，reduced-motion 下不位移）。

- **只读 API 方法校验**：state/status/audit/explain/health/toollists/toollist/export/info/search/presets/preset/outdated 仅接受 GET，其它方法返回 405；导入等 POST 接口不受影响。补测试。

- **修帮助浮层穿透 + gg/G**：帮助/命令浮层打开时不再响应 `j/k/空格/Enter`（修复上轮引入的穿透）；新增 `gg` 回顶、`G` 到底，并写入快捷键卡片。

- **键盘快捷键帮助浮层**：主界面顶栏加 `?` 按钮，按 `?` 弹出快捷键卡片（j/k、空格、Enter、⌘K、Esc），`Esc`/点遮罩关闭。

- **导入/导出安全加固**：`/api/toollist/import` 校验清单 id 字符集（防注入/异常 key）；`/export` 文件名净化（`safeFilename`，防 Content-Disposition 头注入）。补测试。

- **`install`/`uninstall` 参数校验**：加 `cobra.NoArgs`，多余位置参数报错（之前被静默忽略，`sis install foo` 会误装整份清单）；补测试。

- **`--format` 校验**：非法值（如 `--format xml`）现在明确报错（exit 1），不再静默回退表格；补 CLI 测试。

- **⌘K 最近使用 + 容错测试**：命令面板把最近执行的命令置顶（内存态，重启即失，无持久化）；新增 preset 损坏文件容错回归测试（List 不崩、Load 报错）。

- **边界回归测试 + 行按压反馈**：新增 CRLF/BOM 清单解析、`/api/run` 参数校验回归测试；列表行加 `:active` 按压底色。

- **列表键盘导航**：主界面支持 `j/k` 上下选择行、`空格` 勾选、`Enter` 打开详情（输入框/命令面板/按钮聚焦时自动让路），选中行有绿底+左标高亮。

- **安全回归测试 + 顶栏现代化**：补 preset 路径穿越拒绝、清单导入→回读 回归测试；顶栏加镜面高光边并补 `prefers-reduced-transparency` 实心兜底。

- **更多命令尊重 `--format`**：`sis doctor`、`sis guide --list`、`sis llm show`、`sis mirror show` 支持 `json|silent`（之前忽略输出开关）。

- **修复/优化**：`sis toollist list/show/verify` 尊重 `--format`（json/silent，之前恒打印表格）；`sis status` 改用批量 `InstalledSet`（43 包 >20s → ~3s）。

- **UI 成功案例 + ⌘K 模糊排序**：新增 `docs/UI-CASES.md`（Raycast/Linear/Apple Liquid Glass 提炼）；`cmdk.js` 过滤升级为模糊排序（标题子串优先、子序列次之），命令面板加镜面高光边。

- **前端评审报告 + 二轮色值收敛**：新增 `docs/REVIEW-2026-11.md`（人工按 Argus 口径补做）；第二遍把 styles.css/audit 的 `#fff`、npm 品牌红、遮罩黑、品牌渐变绿收敛为 token；设计系统补充「暗色用 prefers-color-scheme」说明。

暂无。

## [0.3.0] - 2026-11-05

### Added

- CI 门禁接入 `go vet ./...`（修 #27）。
- **体检与软件清单（`sis audit`）**：新增「首次启动体检」——扫描本机已装软件（`/Applications` + `brew list`），按可核验维度（授权、免费版范围、维护活跃度、隐私、可导出性）给出评价，并结合内置「只收免费/开源、免费版够用」的软件清单与平替表，输出「可替换 / 建议补充 / 已装且够用」；全程本地扫描、清单不外传。新增 `internal/toollist`（软件清单 + 平替目录，含内置 `mac-designers` 软件清单）与 `internal/audit`（扫描 + 名称归一匹配 + 报告）。
- **迁移指导（`sis guide` / Playbook）**：为平替条目新增「带版本日期与来源的迁移步骤」（数据保真度 + 分步 + 来源 URL）；`sis guide <软件>` 查看、`sis guide --list` 列出；GUI 体检报告「可替换」项可展开查看迁移指导。只给指导，不安装/卸载、不搬数据。
- **可分享的动态软件清单（`sis toollist`）**：`list` / `show` / `export`；`sis toollist export` 把软件清单导出为 YAML，别人 `sis audit --catalog <文件>` 一加载即按其机器呈现（软件清单是「活」的）。GUI 体检页提供「导出软件清单」下载（`/api/toollist/export`）。
- **新增内置软件清单「开发者」+ 多软件清单选择**：内置软件清单增至两份——`developers`（18 条，含 Postman→Bruno、TablePlus→DBeaver 迁移指导）与 `mac-designers`；`sis audit` / `sis guide` 支持 `--toollist <id>`；GUI 体检页可切换软件清单（`/api/toollists` + `?toollist=`），导出也随所选软件清单。
- **主 GUI 统一到《Cabinet of Equivalents》编辑风**：`styles.css` 调色板改为骨纸 / 铁墨 / 氧化绿 / 焦琥珀，标题改用中文衬线，主界面与体检页视觉一致（亮/暗两套）。
- **GUI 新增「软件清单」模块（`/toollists`）**：左栏为软件清单索引，右栏为图鉴式阅读——按分类列条目（含免费/开源标注、免费版够用说明与来源），平替条目可展开查看迁移指导；入口在主界面侧栏与体检页。接口 `/api/toollist?toollist=<id>`。
- **GUI 支持导入他人分享的软件清单**：`POST /api/toollist/import` 解析上传的 YAML，会话内保存并出现在清单列表，可直接浏览与体检（`/api/audit?toollist=<导入 id>`）。同时**面向用户的「书单」统一改为「软件清单」**，并为新手补充了浅显的引导文案（清单页/体检页/主界面/README）。
- **资料新鲜度**：按 `verified_at` 计算新鲜 / 较久 / 需复核（180 / 365 天阈值）；`sis toollist verify <id>` 逐项列出，体检报告与 GUI 对超期项标注「需复核」，`Report.stale_count` 汇总。
- **术语统一**：`booklist` 全部改为 `toollist`（包 `internal/toollist`、命令 `sis toollist`、接口 `/api/toollist(s)`、页面 `/toollists` 等）。
- **AI 补全长尾（`sis explain <软件>`）**：对不在人工清单里的软件，用已配置的 LLM 给出「它是什么 / 有无成熟免费或开源平替 / 迁移注意」，输出明确标注为 AI 生成、仅供参考。新增 `internal/advice`（可注入 `Chatter` 便于测试）。不配置 LLM 时友好提示 `sis llm set`。
- **新增第三份软件清单「内容创作者」**：`creators`（16 条：DaVinci Resolve / OBS / GIMP / Inkscape / Krita / darktable / Blender…），含 Premiere→DaVinci Resolve、Photoshop→GIMP、Illustrator→Inkscape 三条带日期与来源的迁移指导。
- **GUI 接入 AI**：新增 `/api/explain`，清单页（`/toollists`）提供「用 AI 查清单外软件 / 找免费平替」输入框与结果区；未配置 LLM 时返回友好提示，输出标注「AI 生成，仅供参考」。
- **交互层测试加固**：为 GUI 多个 handler（presets/preset/outdated/llm）补 httptest；为 CLI helper（`freshnessLabel`/`ageStr`/`joinURLs`）与 root 命令注册补单测；gui 覆盖率 10.6%→24.4%。
- **LLM 端到端测试（mock）**：新增本地 mock OpenAI 端点测试，覆盖 llm.Chat/Complete、advice.Explain、以及 GUI `/api/explain` 全链路（llm 覆盖率 0→46.7%）。
- **新增 `sis doctor` 自检命令**：一键检查包管理器、已装扫描、3 份软件清单的新鲜度汇总、LLM 配置，帮助用户自行诊断环境。
- **体检覆盖未收录应用**：`sis audit` 新增 `unmatched_count` / `unmatched_sample`——列出「已安装但不在软件清单中」的应用（最多 20 个示例），并在 CLI/GUI 提示可用 `sis explain` 走 AI 了解与找免费平替。
- **全产品回归 + JSON 清洁度**：全命令冒烟 25/25、GUI 路由 8/8 通过；`sis audit` 的 `replaces/adds/keeps` 空集输出由 `null` 改为 `[]`，方便脚本消费。
- **补充迁移指导**：`mac-designers` 新增 `Alfred → Raycast`、`Notion → Obsidian` 两条带数据保真与来源的迁移指导；该清单平替现 7 组且全部含指导。
- **新增第 4 份清单「Mac 新手上手」（`essentials`）**：14 条常用免费/开源工具，面向刚用 Mac 的新手，每条用大白话写明“为什么值得装”。内置清单增至 4 份、共 85 项。
- **官网落地页（index.html）刷新**：Hero、数据条、6 张特性卡改为「环境体检 / 精选软件清单 / 迁移指导 / AI 平替 / 一键安装 / 自检」，并**修正 macOS「即将推出」为真实 Homebrew 安装命令**。
- **落地页 CLI 段落同步**：命令卡从 5 条增至 14 条（含 gui/audit/toollist/guide/explain/doctor/search/preset/tui），底部 CTA 增加 macOS 安装命令。
- **路由集成测试**：把 GUI 路由抽出为 `server.routes()`，用 `httptest` 覆盖全部页面与接口（含 `/api/toollist/import` POST），gui 覆盖率 25.7%→31.6%。
- **清单页「设为安装清单」**：把一份软件清单保存为预设（`POST /api/preset?action=save`），随后到主界面一键安装——打通“浏览 → 安装”闭环（不自动执行）。gui 覆盖率 31.6%→37.9%。
- **GUI 维护视图（新鲜度）**：`/api/toollists` 返回每个清单的新鲜度计数（fresh/aging/stale），清单页侧栏显示“N 需复核”、顶部对超期项给出提示，强化“版本核验”的可视化。
- **交互设计方案 + 平替滑卡原型**：新增 `docs/INTERACTION-DESIGN.md`（2026 动效/渐进披露/两阶段漏斗调研 + 三个签名交互 Spec），并做出可点原型 `docs/prototype/swap-cards.html`（拖拽/键盘/翻面/撤销）与预览图。
- **`/swap` 逐条决策页（实际落地）**：从真实软件清单的平替生成“平替滑卡”（拖拽/键盘/翻面），结束可「把已换的存成安装清单」（复用预设）；清单页新增「逐条决策这些平替」入口。
- **交互设计 v2 + 对比原型**：`docs/INTERACTION-DESIGN.md` 补充竞品交互拆解（Setapp/CaskHub/Bundl/Ninite/Raycast Store）、Liquid Glass 与“三 Z 平面”规范、滑块无障碍规范；新增并排对比原型 `docs/prototype/compare.html`（文字维度默认并排，滑块仅用于截图前后对比）。
- **主界面 `⌘K` 命令面板**：键盘优先（Raycast 式）——体检 / 软件清单 / 逐条决策 / 过滤清单 / 检查更新 / 刷新，一个快捷键到位；顶栏加 ⌘K 入口。
- **体检 Scan→Report 形变**：扫描完成时扫描器轻缩淡出、报告统计卡错峰弹入（首次仪式，`prefers-reduced-motion` 降级）；体检报告顶部新增「逐条决策这些平替 →」直连 `/swap`。
- **⌘K 命令面板全站化**：抽出共享 `cmdk.js`（自注入样式与浮层），接入 `/audit`、`/toollists`、`/swap`，让键盘优先体验贯穿全站（导航/逐条决策/回顶/刷新）。
- **浅色主题视觉 QA**：对 `/swap` 平替滑卡做亮色快照验收（暖纸底/白卡/绿·琥珀标签、对比与层级成立），确认亮/暗两套一致。
- **亮色 QA 扩展到主界面**：主 GUI 外壳（侧栏/顶栏/列表/操作条）亮色快照验收通过（`docs/screenshots/gui-main-light.png`），令牌与层级在亮色下成立。
- **新增第 5 份清单「隐私与安全」（`privacy`）**：13 条（Firefox/LibreWolf/Signal/Thunderbird/OnionShare/KeePassXC/Cryptomator/VeraCrypt/LuLu 等），含 Chrome→Firefox、LastPass→KeePassXC 迁移指导。内置清单增至 5 份。
- **GUI 系统自检页 `/health`**：`/api/health` 汇集平台/包管理器/已装扫描/清单新鲜度/AI 配置，门面页一键查看（与 CLI `sis doctor` 对齐），支持“复制诊断 JSON”；主界面侧栏与 `⌘K` 均可进入。
- **清单浏览页过滤**：`/toollists` 顶部新增过滤框，按名称/用途/理由即时筛选条目与平替（纯客户端，空分类自动隐藏）。
- **设计系统文档**：新增 `docs/DESIGN-SYSTEM.md`（颜色/字体/间距/运动 token、按钮/标签/行/卡片/抽屉/命令面板/滑卡/对比/步进组件规范、无障碍门槛、Do/Don't、代码映射）。
- **无障碍硬化**：`⌘K` 面板加 `prefers-reduced-transparency` 实心兜底；顶栏按钮补 `aria-label`；体检/清单动态区加 `aria-live`（兑现设计系统门槛）。
- **发布就绪审计**：新增 `docs/RELEASE-CHECKLIST.md`——版本一致（0.3.0）、CLI 23/23、GUI 12/12、`go test -race` 全绿；记录发布步骤与已知边界（唯一未完成项：提交/开 PR 待授权）。
- **执行动效改进计划（001–004）**：滑卡松手改速度判定 + 飞出与换牌对齐（320ms）、reduced-motion 只去位移保留颜色/透明度、全站按钮加 `:active` 按压反馈并门控 hover、扫描进度条改用 `transform:scaleX`。计划见 `plans/`。
- **动效机会点落地**：体检报告分类行错峰入场（每项 +40ms，最多 8 档）、自检页状态行轻淡入（+45ms）；均在 `prefers-reduced-motion` 下退化为直显。
- **增长/获客调研（GTM）**：新增 `docs/GTM.md`——一手 Product Hunt 复盘结论（发布是脉冲、内容是长期获客、真实截图/具体描述/周二至四发布）+ 渠道矩阵、发布清单、漏斗指标；并说明本项目「清单/滑卡/体检」天然适合内容传播。
- **分享卡片页 `/share`**：一键把软件清单渲染成适合小红书/朋友圈的海报（标题/受众/两列工具+授权标签/品牌页脚），并「复制文案」生成可粘贴的推广文案；清单页加入口。落地 GTM 的“清单即传播物”。
- **分享卡片「下载图片」**：`/share` 用 Canvas 导出 1080pt 高清 PNG（标题/双列清单/授权标签/品牌页脚），无依赖、可直接发小红书/朋友圈。
- **国内镜像基础设施**：`sis mirror apply/show/restore` 一键配置 Homebrew（中科大）、npm（npmmirror）、pip（清华）、GitHub 下载代理，干净可回滚（受管理 env 文件 + shell rc 标记块，幂等不重复）。
- **多源安装**：manifest `Package.source` 支持 `brew`（默认）/`npm`/`github`；新增 `NpmBackend`（`npm install -g`）与 `GithubBackend`（下载 release 二进制到 `~/.local/bin`，支持 zip/tar.gz、`{os}/{arch}` 占位、`SIS_GH_PROXY` 加速）；engine 按源选后端。
- **Homebrew 后端**：macOS 上支持 formula 与 cask 两类包；`sis` 按操作系统自动选用 Homebrew（macOS）或 winget（Windows）。
- **图形界面（GUI）**：新增 `sis gui`，在本地浏览器窗口中浏览、勾选并一键安装/卸载，实时状态与逐条进度；前端资源 `go:embed` 内嵌，默认仅监听 127.0.0.1，`--lan`/`--port` 可开局域网预览。每个软件显示品牌图标（内嵌 SVG，缺失时以字母色块回退）。
- **交互式 TUI**：新增 `sis tui`，在终端内浏览清单、选择并安装。
- **快速状态探测**：Homebrew 后端新增一次性批量 `InstalledSet`，数十个包的状态探测从逐包调用降为一次调用。
- **仓库搜索**：新增 `sis search <词>` 命令与 GUI「搜索 Homebrew」面板，可搜索 Homebrew 全部 formula/cask，结果可加入清单或直接安装（`backend.Searcher` 可选能力）。
- **预设（Presets）**：可把软件清单按习惯保存为命名预设（`sis preset save/list/show/delete`，存于系统配置目录）；CLI 与 GUI 均支持 `--preset/-p` 加载。GUI 顶部预设栏可切换、另存为、删除。`internal/preset` 是 sis 唯一的持久化（用户显式保存的数据，非隐藏配置）。
- **macOS 发布产物**：release CI 交叉构建 `sis-darwin-arm64`、`sis-darwin-amd64`、`sis-windows-amd64.exe`、`sis-linux-amd64`，统一发布到同一个 GitHub Release。
- **一键安装脚本**：新增 `install.sh`，支持架构自动检测、`SIS_VERSION` 指定版本、`SIS_MIRROR` 镜像加速，安装到 `/usr/local/bin`（回退 `~/bin`）并验证版本。
- **Manifest `kind` 字段**：`Package.kind` 支持 `formula` / `cask`，留空则自动探测（formula 优先）。

### Changed

- README 与 `docs/CLI_USAGE.md` 改为 macOS / Homebrew 优先，Windows（winget）继续支持。
- CI 从 Windows-only 扩展为 Linux（build + vet + test + 冒烟）与 Windows（winget 路径可编译）双作业。

### Fixed

- 修复 `internal/audit` 的 `.app` 目录识别：macOS 的 `.app` 是目录而非文件，原逻辑仅在文件分支匹配 `.app`，导致 `/Applications` 扫描读不到任何应用（只统计到 `brew list` 的 89 项）；现按目录识别并跳过包内 helper，真机覆盖 89 → 233。补 `TestSystemLister_TempDir` 防回归。


- 文档中「macOS 开发中 / not implemented」等过时描述全部修正，与真实 CLI（含 `tui`、`kind`）对齐。

## [0.2.3] - 2026-02-20

### 说明

v0.2.0 ~ v0.2.2 的发布未维护 CHANGELOG 条目，相应变更见 [GitHub Releases](https://github.com/cgartlab/SwiftInstall/releases) 与 git tag。

## [v0.1.7] - 2026-02-19

### Fixed
- 修复 `install` 和 `search` 命令的 `help` 参数处理，现在正确显示帮助文档
- 修复 `search` 命令无参数时的行为，现在启动交互式搜索界面
- 修复搜索界面表格列标题不一致问题（Category → Publisher）
- 修复主菜单快捷键执行后状态混乱的问题
- 修复安装界面完成后只能按 Enter 退出的问题
- 修复 About 页面无交互的问题

### Added
- 搜索界面添加清晰的快捷键提示
- 安装界面改进帮助文本显示
- About 页面改为 TUI 交互模式，支持任意键返回
- 主菜单快捷键执行后正确退出并返回终端

### Changed
- 改进 `hasHelpArg()` 函数，支持在参数任意位置检测 help
- 优化搜索界面帮助文本，显示 "Add: Enter | Refine: / | Back: Esc | Quit: q"
- 优化安装界面完成后的退出提示 "Exit: Enter/Esc | About: a | Quit: q"
- 更新 README.md，添加完整的快捷键说明表格

## [v0.1.6] - 2026-02-19

### Added
- 新增自动更新检查功能，启动时可选择是否启用
- 新增环境预检功能，安装/搜索前自动检测包管理器状态
- 新增 Linux 平台支持（见 `internal/installer/linux.go`）
- 新增 `sis setup` 命令，一键自动完成环境检测、依赖准备与验证

### Changed
- 优化 setup 流程，支持 `--auto-install-deps` 和 `--dry-run` 参数
- 增强 release 日志记录模块
- 改进软件接口特性和用户体验

### Fixed
- 修复配置保存错误处理
- 修复安装模型指针接收器问题
- 修复并发安装中的 panic 恢复和错误处理问题

## [v0.1.5] - 2026-02-18

### Changed
- 合并多个特性分支，统一代码结构
- 优化软件接口特性

## [v0.1.4] - 2026-02-18

### Added
- 增强菜单导航系统
- 改进 help 系统，支持更完整的帮助文档

### Changed
- 更新项目文档和许可证信息

## [v0.1.3] - 2026-02-18

### Fixed
- 修复 `InstallModel` 并发安全问题，添加 `sync.Mutex` 保护共享数据
- 合并 `cmd/root.go` 中的重复 `init()` 函数
- 修复 `config.save()` 方法，保存所有配置项（language, theme, parallel_install 等）
- 修复 `getOSName()` 和 `getArch()` 使用 `runtime.GOOS` 和 `runtime.GOARCH` 动态获取系统信息
- 修复 Makefile 中 ldflags 包路径错误
- 添加 `validatePackageID()` 函数验证包 ID 格式，防止潜在命令注入
- 改进 `parseWingetLine()` 函数，使用正则表达式更好地解析含空格的软件名
- 修复 `IsInstalled()` 方法错误返回值，正确返回错误信息
- 修复 `Install()` 方法中 `IsInstalled` 错误被忽略的问题
- 修复 `BatchInstall()` 并发安装时缺少 panic 恢复机制
- 修复 `install.go` 中非并行模式实际仍是并发执行的问题
- 修复 `ShowSpinner()` goroutine 缺少 panic 恢复机制
- 修复 `installPackage()` 中 `Install` 错误被忽略的问题
- 添加数组越界检查防止 table 更新崩溃

### Added
- 添加 `internal/installer/installer_test.go` 测试文件
  - TestValidatePackageID: 包 ID 验证测试
  - TestParseWingetLine: winget 输出解析测试
  - TestCheckPackageManager: 包管理器检测测试
- 添加 `internal/config/config_test.go` 测试文件
  - TestConfigInit: 配置初始化测试
  - TestConfigSave: 配置保存测试
  - TestConfigPersistence: 配置持久化测试
  - TestConcurrentAccess: 并发访问测试
- 添加 `internal/i18n/i18n_test.go` 测试文件
  - TestTranslation: 翻译功能测试
  - TestLanguageSwitch: 语言切换测试
  - TestCompleteness: 翻译完整性测试
- 添加包级正则表达式缓存，优化性能

### Changed
- 优化并发安装时的线程安全性
- 改进 winget 输出解析的健壮性
- 统一错误处理模式，添加日志记录
- 改进并发安装的 semaphore 获取方式，避免主 goroutine 阻塞

## [0.1.2] - 2024-01-01

### Added
- 初始版本发布
- 支持 Windows (Winget) 和 macOS (Homebrew)
- 交互式 TUI 界面
- 多语言支持（中文/英文）
- 批量安装功能
- 配置文件管理
