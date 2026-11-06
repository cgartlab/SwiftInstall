# 共享基线架构（方案 C）

> 2026-11-07 定稿。用户选定「抽公共基线，两线各自增强」。本文定义 `core/shared` 基线与两条平台线的分工与同步规则。

## 1. 三条分支

| 分支 | 角色 | 允许放什么 | 不允许放什么 |
|---|---|---|---|
| `core/shared` | 共享基线 | 平台无关代码、公共测试、架构文档、公共 CI 定义 | 任何 `winget` / `brew` 专属行为、平台特有清单、平台特有发布配置 |
| `main` | Windows 线 | winget 后端增强、Windows CI（含 `go test`）、Windows 多架构发布、Windows 清单/脚本、管理员权限与源镜像 | macOS 特有功能、Homebrew 深度能力 |
| `feat/product-v0.3`（长期） | macOS 线 | Homebrew 深度能力、macOS 清单与 `install.sh`、macOS 发布配置 | Windows 特有功能 |

公共更新**单向**从 `core/shared` 流向两条平台线（merge / rebase），不反向合回基线；两条平台线之间不互相合并，也不合回 `main`。

## 2. 代码分层

- `internal/backend/backend.go` — `Backend` 接口与可选能力接口（公共）
- `internal/backend/registry.go` — 可插拔平台注册表：`RegisterPlatform` / `PlatformFactories` / `ForGOOS` / `ForCurrentOS` / `ExtraBackends`（公共，不含平台分支判断）
- `internal/backend/platform_defaults.go` — 默认注册：`windows→winget`、`darwin→homebrew`（放在基线，保证任何主机都能取到任一后端）
- `internal/backend/{winget,homebrew,npm,github}.go` — 具体实现与其测试（跨平台编译、跨平台测试，不加 build tags）
- 其余 `internal/*` — CLI、engine、manifest、ui、gui、tui、audit、toollist、advice、llm、mirror、preset 均为平台无关公共代码

### 为什么不加 build tags

Windows 线上要能跑**全部**测试（包括 Homebrew 后端测试），macOS 线同理。所有后端都是纯 Go + `exec.Command` 字符串，不依赖平台 API，跨平台编译无成本。加 build tags 反而会让 Windows CI 少测一半代码，与「做到更好」相悖。

### 平台线如何注入增强后端

```go
// Windows 线自己的包（例如 internal/wingetext），init() 中覆盖默认注册
func init() {
    backend.RegisterPlatform("windows", func() backend.Backend {
        return NewExtendedWingetBackend()
    })
}
```

公共基线代码完全不改动；`ForGOOS("darwin")` 在 Windows 主机上仍可解析，便于单测覆盖双平台映射。

## 3. 同步规则

1. 公共改动先提交到 `core/shared`，跑 `go build ./... && go vet ./... && go test ./...`（Windows 与 macOS 各跑一次）。
2. 两条平台线各自 merge / rebase 基线更新；冲突只在各自平台特有文件里解决，不要把平台特有代码带回基线。
3. 平台特有改动只落在各自的线，不进基线、不进对方线。
4. 版本号、CHANGELOG 由各线自行维护；基线不打产品版本 tag。

## 4. 验证基线（Windows 实测 2026-11-07）

- `go build ./...` rc=0
- `go vet ./...` rc=0
- `go test ./internal/backend/ -v`：新增 4 个注册表测试全部 PASS，原有 Homebrew / registry 测试保持通过

## 5. 后续（Windows 线「做到更好」清单）

- Windows CI job 也跑 `go test ./...`（当前 macOS 分支的 windows job 只跑 build+smoke，这是现成可超越点）
- 多架构发布 windows amd64 + arm64、SHA256SUMS、`workflow_dispatch`（issue #29）
- winget 增强：源镜像 / USTC、导出导入清单、升级、管理员权限检测
- `ForGOOS` 驱动的平台映射测试覆盖更多 GOOS

## 6. 环境约束

- 远程写入（推送 / 开 PR / 合并）需要恢复会话审批；PR #34 的 `review` fail check 需在合并前处理。
- git 全局代理端口反复漂移，必要时应改走直连或固定端口。
