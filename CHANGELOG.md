# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Added

- CI 门禁接入 `go vet ./...`（修 #27）。
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

- 文档中「macOS 开发中 / not implemented」等过时描述全部修正，与真实 CLI（含 `tui`、`kind`）对齐。

## [0.2.3] - 2026-02-20

### 说明

早期版本（v0.1.x ~ v0.2.2）发布前未维护 CHANGELOG，历史条目自 v0.2.3 起补记；
更早的变更记录见 [GitHub Releases](https://github.com/cgartlab/SwiftInstall/releases) 与 git tag。
