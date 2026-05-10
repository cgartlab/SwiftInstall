# Tasks

## Phase 1: 删除无用包和命令

- [x] Task 1: 删除 `internal/config/` 包及其所有引用
  - [x] SubTask 1.1: 删除 `internal/config/config.go`
  - [x] SubTask 1.2: 从 `cli/install.go`、`cli/status.go`、`cli/uninstall.go`、`cli/list.go`、`cli/mirror.go`、`cli/config.go` 中移除 `config` 包的 import 和调用
  - [x] SubTask 1.3: 删除 `cli/config.go`（整个 config 子命令）
  - [x] SubTask 1.4: 验证 `go build` 通过

- [x] Task 2: 删除 `internal/mirror/` 包及其所有引用
  - [x] SubTask 2.1: 删除 `internal/mirror/mirror.go`
  - [x] SubTask 2.2: 从 `cli/install.go`、`cli/mirror.go` 中移除 mirror 引用
  - [x] SubTask 2.3: 删除 `cli/mirror.go`（整个 mirror 子命令）
  - [x] SubTask 2.4: 验证 `go build` 通过

- [x] Task 3: 删除 `internal/proxy/` 包及其所有引用
  - [x] SubTask 3.1: 删除 `internal/proxy/proxy.go`
  - [x] SubTask 3.2: 从 `cli/install.go` 中移除 proxy 引用和自动检测逻辑
  - [x] SubTask 3.3: 验证 `go build` 通过

- [x] Task 4: 删除 `internal/log/` 包
  - [x] SubTask 4.1: 删除 `internal/log/logger.go`
  - [x] SubTask 4.2: 验证 `go build` 通过

- [x] Task 5: 删除 `cli/uninstall.go` 和 `cli/list.go`
  - [x] SubTask 5.1: 删除 `cli/uninstall.go`
  - [x] SubTask 5.2: 删除 `cli/list.go`
  - [x] SubTask 5.3: 从 `cli/root.go` 中移除 `newUninstallCmd` 和 `newListCmd` 的注册
  - [x] SubTask 5.4: 验证 `go build` 通过

## Phase 2: 重构 Backend 接口

- [x] Task 6: 简化 `internal/backend/backend.go`
  - [x] SubTask 6.1: 移除 `Uninstall`、`ListInstalled`、`IsPermanent` 方法
  - [x] SubTask 6.2: 简化 `InstallOptions`，移除 `DryRun`
  - [x] SubTask 6.3: 移除 `FormatWingetError` 和 `ErrAlreadyInstalled`
  - [x] SubTask 6.4: 验证 `go build` 通过

- [x] Task 7: 重构 `internal/backend/winget.go`
  - [x] SubTask 7.1: 移除 `Uninstall` 方法
  - [x] SubTask 7.2: 移除 `ListInstalled` 方法
  - [x] SubTask 7.3: 移除 `IsPermanent` 方法
  - [x] SubTask 7.4: 简化 `Install` 方法，移除 DryRun 处理
  - [x] SubTask 7.5: 移除 `parseWingetListOutput` 函数
  - [x] SubTask 7.6: 验证 `go build` 通过

- [x] Task 8: 删除 `internal/backend/factory.go`
  - [x] SubTask 8.1: 删除 `factory.go`
  - [x] SubTask 8.2: 在 `cli/install.go` 和 `cli/status.go` 中直接使用 `backend.NewWingetBackend()`
  - [x] SubTask 8.3: 验证 `go build` 通过

## Phase 3: 重构 Engine 和 Manifest

- [x] Task 9: 创建 `internal/manifest/` 包
  - [x] SubTask 9.1: 创建 `internal/manifest/manifest.go`，定义 `Manifest`、`Package`、`Settings` 结构体
  - [x] SubTask 9.2: 从 `engine/manifest.go` 迁移解析逻辑到 `manifest/parser.go`
  - [x] SubTask 9.3: 在 `manifest/parser.go` 中支持 `settings` 节点解析
  - [x] SubTask 9.4: 创建 `manifest/validate.go`，添加清单校验逻辑
  - [x] SubTask 9.5: 验证 `go build` 通过

- [x] Task 10: 重构 `internal/engine/engine.go`
  - [x] SubTask 10.1: 移除 `Uninstall` 方法
  - [x] SubTask 10.2: 移除 `CheckStatus` 方法（移到 cli/status.go 直接用 backend）
  - [x] SubTask 10.3: 简化 `Install` 方法，移除复杂的重试逻辑，改为简单的循环
  - [x] SubTask 10.4: 移除 `installWithRetry`、`uninstallWithRetry`、`retry`、`stop`、`isStop` 等辅助函数
  - [x] SubTask 10.5: 引入 `InstallOptions` 结构体替代 `Config`
  - [x] SubTask 10.6: 验证 `go build` 通过

- [x] Task 11: 删除 `internal/engine/manifest.go` 和 `internal/engine/check.go`
  - [x] SubTask 11.1: 删除 `engine/manifest.go`（逻辑已迁移到 manifest 包）
  - [x] SubTask 11.2: 删除 `engine/check.go`（预检逻辑简化，合并到 install 命令）
  - [x] SubTask 11.3: 验证 `go build` 通过

## Phase 4: 重构 CLI 层

- [x] Task 12: 重构 `cli/install.go`
  - [x] SubTask 12.1: 使用 `manifest` 包替代 `engine.ParseManifest`
  - [x] SubTask 12.2: 从 manifest 的 `settings` 读取配置（proxy、retry、skip_existing 等）
  - [x] SubTask 12.3: 简化 preamble 逻辑：加载 manifest → 检测 backend → 执行安装
  - [x] SubTask 12.4: 移除 `--proxy`、`--mirror`、`--skip-checks` 等 flag（由 manifest settings 替代）
  - [x] SubTask 12.5: 保留 `--dry-run`、`--file`、`--format` flag
  - [x] SubTask 12.6: 验证 `go build` 通过

- [x] Task 13: 重构 `cli/status.go`
  - [x] SubTask 13.1: 使用 `manifest` 包替代 `engine.ParseManifest`
  - [x] SubTask 13.2: 直接使用 `backend.IsInstalled` 检查每个包状态
  - [x] SubTask 13.3: 移除对 `engine.New` 和 `engine.CheckStatus` 的依赖
  - [x] SubTask 13.4: 验证 `go build` 通过

- [x] Task 14: 创建 `internal/ui/` 包
  - [x] SubTask 14.1: 创建 `ui/renderer.go`，定义 `Renderer` 接口
  - [x] SubTask 14.2: 创建 `ui/terminal.go`，实现终端表格输出
  - [x] SubTask 14.3: 创建 `ui/json.go`，实现 JSON 输出
  - [x] SubTask 14.4: 在 `cli/install.go` 和 `cli/status.go` 中使用 `ui` 包
  - [x] SubTask 14.5: 验证 `go build` 通过

- [x] Task 15: 清理 `cli/root.go`
  - [x] SubTask 15.1: 移除 `--verbose` flag（未使用）
  - [x] SubTask 15.2: 简化 `PersistentPreRunE`
  - [x] SubTask 15.3: 验证 `go build` 通过

## Phase 5: 清理和验证

- [x] Task 16: 清理遗留文件
  - [x] SubTask 16.1: 删除 `Windows/software_install_proxy.bat`（代理逻辑已移除）
  - [x] SubTask 16.2: 删除 `Windows/switch_winget_to_USTCsource.bat`（mirror 命令已移除）
  - [x] SubTask 16.3: 更新 `Windows/software_list.txt` 为新的 manifest 格式示例
  - [x] SubTask 16.4: 更新 `macOS/packages.txt` 为新的 manifest 格式示例

- [x] Task 17: 更新 `go.mod`
  - [x] SubTask 17.1: 运行 `go mod tidy` 清理未使用的依赖
  - [x] SubTask 17.2: 验证 `go build` 通过

- [x] Task 18: 运行 `go vet ./...`
  - [x] SubTask 18.1: 修复所有 vet 警告
  - [x] SubTask 18.2: 验证通过

# Task Dependencies

- Task 1-5 可并行执行（各自删除不同包）
- Task 6-8 依赖 Task 1-5 完成（backend 接口简化）
- Task 9 可独立执行（manifest 包创建）
- Task 10-11 依赖 Task 6-8 和 Task 9
- Task 12-15 依赖 Task 10-11
- Task 16-18 依赖 Task 12-15
