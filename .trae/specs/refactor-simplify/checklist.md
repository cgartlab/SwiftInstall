# Checklist

## Phase 1: 删除无用包和命令

- [x] `internal/config/` 包已完全删除，所有引用已清理
- [x] `internal/mirror/` 包已完全删除，所有引用已清理
- [x] `internal/proxy/` 包已完全删除，所有引用已清理
- [x] `internal/log/` 包已完全删除
- [x] `cli/uninstall.go` 已删除，`root.go` 中已移除注册
- [x] `cli/list.go` 已删除，`root.go` 中已移除注册
- [x] `cli/config.go` 已删除，`root.go` 中已移除注册
- [x] `cli/mirror.go` 已删除，`root.go` 中已移除注册
- [x] Phase 1 完成后 `go build` 通过

## Phase 2: 重构 Backend 接口

- [x] `backend.go` 接口只包含 `Name`、`Detect`、`Install`、`IsInstalled`
- [x] `InstallOptions` 已简化，移除 `DryRun`
- [x] `winget.go` 已移除 `Uninstall`、`ListInstalled`、`IsPermanent` 方法
- [x] `winget.go` 已移除 `parseWingetListOutput` 函数
- [x] `factory.go` 已删除
- [x] Phase 2 完成后 `go build` 通过

## Phase 3: 重构 Engine 和 Manifest

- [x] `internal/manifest/` 包已创建，包含 `manifest.go`、`parser.go`、`validate.go`
- [x] `manifest.go` 定义了 `Manifest`、`Package`、`Settings` 结构体
- [x] `parser.go` 支持 YAML 和 TXT 格式，支持 `settings` 节点
- [x] `engine.go` 已移除 `Uninstall`、`CheckStatus` 方法
- [x] `engine.go` 已简化 `Install` 方法，移除复杂重试逻辑
- [x] `engine/manifest.go` 已删除
- [x] `engine/check.go` 已删除
- [x] Phase 3 完成后 `go build` 通过

## Phase 4: 重构 CLI 层

- [x] `cli/install.go` 使用 `manifest` 包解析清单
- [x] `cli/install.go` 从 manifest `settings` 读取配置
- [x] `cli/install.go` 保留 `--dry-run`、`--file`、`--format` flag
- [x] `cli/install.go` 移除 `--proxy`、`--mirror`、`--skip-checks` flag
- [x] `cli/status.go` 直接使用 `backend.IsInstalled` 检查状态
- [x] `cli/status.go` 使用 `manifest` 包解析清单
- [x] `internal/ui/` 包已创建，包含 `renderer.go`、`terminal.go`、`json.go`
- [x] `cli/install.go` 和 `cli/status.go` 使用 `ui` 包输出
- [x] `cli/root.go` 已移除 `--verbose` flag
- [x] Phase 4 完成后 `go build` 通过

## Phase 5: 清理和验证

- [x] `Windows/software_install_proxy.bat` 已删除
- [x] `Windows/switch_winget_to_USTCsource.bat` 已删除
- [x] `Windows/software_list.txt` 已更新为新格式示例
- [x] `macOS/packages.txt` 已更新为新格式示例
- [x] `go mod tidy` 已运行，未使用依赖已清理
- [x] `go vet ./...` 无警告
- [x] `go build -o sis.exe ./cmd/sis/` 成功生成可执行文件
- [x] 最终 `sis install`、`sis status`、`sis version` 三个命令可用
