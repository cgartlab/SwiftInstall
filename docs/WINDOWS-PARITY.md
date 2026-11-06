# Windows 线：看齐 macOS 进度并做得更好

> 2026-11-07。分支 `windows/parity-v0.3`，基于共享基线 `core/shared`（提交 `20f9daf`）。

## 1. 看齐（从基线同步）

Windows 线已包含原 macOS 进度线（`feat/product-v0.3`）的全部跨平台能力：

- 16 个 CLI 命令、GUI / TUI、5 份内置软件清单、AI 补全、环境体检、迁移指导、系统自检
- 可插拔平台后端注册表（`RegisterPlatform` / `ForGOOS`）+ Homebrew / winget / npm / github 后端
- 12 个包的测试、设计系统文档、原型与截图
- `VERSION = 0.3.0`

## 2. 做得更好（Windows 线独有增强）

| 项 | macOS 进度线 | Windows 线 |
|---|---|---|
| Windows CI | 只跑 build + smoke（不跑测试） | `go build` + `go vet` + **`go test ./...`** + smoke + 上传产物 |
| 发布触发 | 仅 tag push | tag push **+ `workflow_dispatch`**（可指定 tag） |
| Windows 发布产物 | amd64 | amd64 **+ arm64** |
| 发布完整性 | 无校验和 | **`SHA256SUMS`** 随 Release 发布 |
| 发布矩阵 | 4 个目标 | **5 个目标** |
| Windows 测试隔离 | 只设 `HOME` / `XDG_CONFIG_HOME`，测试写真实 `%APPDATA%` | 4 个包补 `APPDATA` 隔离，测试不再污染用户目录 |

## 3. 为什么这些算「更好」

- 原 macOS 线的 Windows job 只保证「winget 路径仍可编译」，测试只在 Linux 跑。Windows 线让全量测试（含 Homebrew 后端测试）在 Windows 上也通过，覆盖翻倍。所有后端是纯 Go + `exec.Command`，不加 build tags 即可跨平台编译与测试。
- `workflow_dispatch` + `SHA256SUMS` + Windows arm64 直接回应 issue #29 的三项缺口。
- 测试隔离修复了真实缺陷：`os.UserConfigDir()` 在 Windows 读 `%APPDATA%`，此前测试把 `dev` / `work` / `toollist-x` 等预设写进用户真实漫游目录，既污染数据又导致 `go test ./...` 红灯。

## 4. 验证（Windows，2026-11-07）

- `go build ./...` rc=0
- `go vet ./...` rc=0
- `go test ./...` rc=0（12 个包 ok）
- `gofmt -l`（改动文件）为空；`git diff --check` rc=0
- workflow YAML 解析通过（见提交记录）

## 5. 仍待处理

- 远程写入（推送分支 / 开 PR / 合并）需恢复会话审批。
- `%APPDATA%\sis\presets` 里历史测试残留 `dev.yaml` / `work.yaml` / `toollist-x.yaml` 未删除（用户目录数据，待确认）。
- winget 深度增强（源镜像 / USTC、导出导入清单、升级、管理员权限检测）留待后续轮次。
- PR #34 的 `review` fail check（Argus 约 30 分钟超时）需在合并前处理。
