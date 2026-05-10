# SwiftInstall 精简重构 Spec

## Why

SwiftInstall 当前存在过度工程化问题：两级 JSON 配置系统、未使用的日志模块、硬编码的代理检测、仅支持一个镜像源却拥有独立命令等。这些功能增加了维护成本，却未提供相应价值。核心定位应该是 **winget 的增强型批量安装体验工具**，而非通用跨平台包管理器包装器。

## What Changes

- **BREAKING**: 移除 `sis uninstall`、`sis list`、`sis mirror`、`sis config` 四个子命令
- **BREAKING**: 移除两级 JSON 配置系统（`~/.sis/config.json` + `.sis.json`），所有配置内聚到 manifest YAML
- **BREAKING**: 移除 `internal/config/`、`internal/mirror/`、`internal/proxy/`、`internal/log/` 包
- 重构 `internal/engine/`：引入 Pipeline 安装流水线模型
- 重构 `internal/backend/`：简化 Backend 接口，移除 Uninstall/ListInstalled
- 新增 `internal/manifest/`：清单解析与校验（从 engine 中提取）
- 新增 `internal/ui/`：终端输出渲染器（替代分散的 fmt.Fprintf）
- 保留 `sis install`、`sis status`、`sis version` 三个核心命令
- `sis install` 支持从 manifest 读取 `settings`（proxy、retry、skip_existing 等）

## Impact

- Affected specs: CLI 命令体系、配置管理、镜像管理、安装引擎
- Affected code: `internal/cli/*`, `internal/engine/*`, `internal/backend/*`, `internal/config/*`, `internal/mirror/*`, `internal/proxy/*`, `internal/log/*`
- 用户迁移：原有 `.sis.json` 和 `~/.sis/config.json` 配置需手动迁移到 `sis.yaml` 的 `settings` 节点

## ADDED Requirements

### Requirement: Manifest 自包含配置

The system SHALL support a `settings` section in `sis.yaml` that replaces the old two-level JSON config system.

#### Scenario: Success case
- **WHEN** a `sis.yaml` contains `settings.skip_existing: true`
- **THEN** `sis install` SHALL honor that setting without reading any external config file

### Requirement: Pipeline 安装模型

The system SHALL model each package installation as a Pipeline with configurable steps (check installed → install → verify).

#### Scenario: Success case
- **WHEN** installing a package with `optional: true`
- **THEN** failure SHALL not abort the entire batch

### Requirement: 结构化输出渲染器

The system SHALL support pluggable output renderers (terminal table, JSON, silent).

#### Scenario: Success case
- **WHEN** user runs `sis install --format json`
- **THEN** output SHALL be valid JSON with install results

## MODIFIED Requirements

### Requirement: Backend 接口

The Backend interface SHALL be reduced to:
- `Name() string`
- `Detect() error`
- `Install(ctx, id, opts) error`
- `IsInstalled(ctx, id) (bool, error)`

Removed: `Uninstall`, `ListInstalled`, `IsPermanent` (error handling simplified).

### Requirement: CLI 命令集

保留命令：
- `sis install [manifest]` — 批量安装（支持 `--dry-run`, `--format`, `--file`）
- `sis status [manifest]` — 检查清单安装状态
- `sis version` — 版本信息

移除命令：
- `sis uninstall` — 危险低频，由 winget 直接处理
- `sis list` — cat manifest 即可替代
- `sis mirror` — 由 winget source 直接处理
- `sis config` — 配置内聚到 manifest

## REMOVED Requirements

### Requirement: 两级 JSON 配置系统
**Reason**: 过度复杂，一个批量安装工具不需要 git config 级别的配置管理。配置应内聚到 manifest。
**Migration**: 用户将 `.sis.json` / `~/.sis/config.json` 中的配置迁移到 `sis.yaml` 的 `settings:` 节点。

### Requirement: v2rayN 代理自动检测
**Reason**: 只检测单一软件且硬编码端口，不具备通用性。代理应在 manifest 或环境变量中显式声明。
**Migration**: 在 `sis.yaml` 中设置 `settings.proxy: http://127.0.0.1:10809`。

### Requirement: USTC 镜像源切换命令
**Reason**: 仅支持一个镜像源，却拥有完整命令和包。winget 本身的 source 管理已足够。
**Migration**: 使用 `winget source` 命令直接管理，或在 manifest 中通过 pre-install hook 未来支持。

### Requirement: 自定义 Logger 包
**Reason**: 全项目零引用，标准库 `log/slog` 已足够。
**Migration**: 直接使用 `log/slog`。
