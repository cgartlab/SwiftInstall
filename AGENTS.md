# AGENTS.md — SwiftInstall

> **会话约定**：开始任何工作前，先读根目录 `LESSONS.md`（本仓库从真实失误沉淀的「教训→规则」清单）并遵循其中所有规则；遇到新失误按其 §维护 追加。依据：Anthropic Claude Code memory/best practices + 社区 lessons.md 惯例。

**分层**: 产品 (Products) — 个人产品线

Cross-platform batch software installer. 作者开发的五款个人产品之一。**macOS + Homebrew 现在是一等公民**（formula + cask），Windows 用 `winget`；后端按操作系统自动选择。**Legacy scripts** 用 `winget` (Windows `.bat`) 与 `brew` (macOS `.sh`)，**Go CLI** `sis` 正在增量取代它们。

## Legacy scripts (still present, being replaced)

| Platform | Script | Manifest |
|----------|--------|----------|
| Windows | `Windows/software_install.bat` | `Windows/software_list.txt` |
| macOS | `macOS/install_packages.sh` | `macOS/packages.txt` |

Both scripts accept either a YAML manifest (`- id: xxx` lines) or a plain TXT list. The `.bat` files for proxy and USTC mirror switching were deleted when the corresponding `sis` commands were dropped; the standard `.bat` still offers the USTC source switch interactively.

## New Go CLI (`sis`)

### Build

```powershell
go build -o sis.exe ./cmd/sis/
go vet ./...
```

### CI/CD

- **GitHub Actions**: `.github/workflows/` — `release.yml` (发布), `ci.yml` (push/PR 触发), `argus-review.yml` (PR 评审)
- **Push 触发的 CI**（`ci.yml`）：`ubuntu-latest` 跑 `go build ./...` + `go vet ./...` + `go test ./...` + version/help 冒烟；`windows-latest` 仅验证 winget 路径仍可编译。不再 Windows-only。
- **发布触发**（`release.yml`）：tag push `v*` → 矩阵构建 darwin/arm64、darwin/amd64、windows/amd64、linux/amd64 → 每个二进制校验 `sis version` 输出含 tag → 汇总发布到同一个 GitHub Release（softprops/action-gh-release）。
- **GOPROXY 回退**：CI 与本地下载统一 `GOPROXY=https://goproxy.cn,direct`（proxy.golang.org 在本环境常超时）。
- **一键安装**：macOS `install.sh`（架构检测 + `SIS_VERSION` / `SIS_MIRROR`），Windows `install.ps1`。

### Entrypoint

`cmd/sis/main.go` — Cobra CLI, ldflags-injected version/commit/date.

### Architecture

```
cmd/sis/main.go   Cobra entrypoint; ldflags inject Version/Commit/Date
internal/
  cli/            Cobra commands (thin orchestration layer)
                  install | uninstall | list | status | version | gui | tui | search | preset | mirror
                  helpers.go — manifest auto-discovery + renderer selection
  engine/         Batch install/uninstall orchestration
                  dedupe, skip-existing, dry-run, retry loop, progress callback
  manifest/       Manifest schema, YAML/TXT parsing, validation
                  manifest.go | parser.go | validate.go
                  Package.kind = formula | cask (Homebrew; empty = auto-detect)
  backend/        Backend interface (Name/Detect/IsInstalled/Install/Uninstall)
                  + registry.go (ForCurrentOS + ExtraBackends: darwin→homebrew, npm, github)
                  + homebrew.go (formula+cask, bulk InstalledSet, Info) | npm.go | github.go | winget.go
  mirror/         China mirror config (brew/npm/pip/ghproxy) — apply/show/restore, idempotent
  gui/            Local web GUI: HTTP + SSE API (state/status/search/info/preset/run) + go:embed assets
  tui/            Interactive terminal UI (bubbletea/lipgloss)
  preset/         Named manifest presets (save/list/load/delete) — the only persistence sis keeps
  ui/             Renderer interface + terminal | json | silent implementations
```

Earlier waves also shipped `internal/config`, `internal/proxy`,
`internal/log` and the `config`/`mirror` subcommands. They were deleted as
dead weight — `sis` is flag-driven, with no config file governing CLI behaviour.
Two pieces of persistence are kept because the user explicitly asked for them:
`internal/preset` (named manifest files) and `internal/mirror` (China mirror
config: one managed env file + shell-rc marker block, `sis mirror restore`
removes it). Do not reintroduce config files, proxy state or other hidden
persistence without a concrete need.

### Development status

| Wave | Feature | Status |
|------|---------|--------|
| 1 | Foundation (CLI skeleton, types) | Done |
| 2 | Engine + winget backend | Done |
| 3 | ~~Mirror, proxy, preflight~~ | Removed (flag-driven instead) |
| 4 | CLI commands wiring (install, list, uninstall, status) | Done |
| 5 | Polish (progress, colors, UI renderers, CI) | Done |
| 6 | Homebrew backend for macOS (formula + cask) + 后端自动选择 | Done |
| 7 | 交互式 TUI（`sis tui`）+ macOS 发布产物 / `install.sh`（v0.2.3） | Done |

### Config precedence

There is no config file. Flags beat manifest settings, which beat built-in
defaults (`install.go` resolves this with `cmd.Flags().Changed`).

## Gotchas

- **`bin/` was deleted** — it held precompiled Windows binaries from an earlier attempt, with no source in this repo. `.gitignore` covers `*.exe` now.
- **Manifest format is decided by content, not extension** — `Windows/software_list.txt` and `macOS/packages.txt` contain YAML. `manifest.ParseManifest` sniffs for a `packages:`/`settings:` key. Do not "fix" these files by renaming without also updating the legacy scripts' expectations.
- **The legacy scripts must stay format-agnostic** — they extract `- id:` lines but still accept plain TXT lists. Either consumer breaking is a bug.
- **Docs are Chinese** — scripts and README in Simplified Chinese. USTC mirror switching lives only in `Windows/software_install.bat`.
- **Preflight/admin detection does not exist** — the only environment check is `backend.Detect()` choosing the package manager (`winget` / `brew`). DESIGN_ISSUES.md and TEST_REPORT.md still claim otherwise; treat both as historical.
- **macOS is supported now** — README/CLI docs no longer say "macOS 开发中 / not implemented". Keep them in sync with the real CLI, including `sis tui` and the manifest `kind: formula|cask` field.
- **macOS manifest tokens** — `macOS/packages.txt` must use real lowercase Homebrew tokens; GUI apps need `kind: cask`.
- **Production landing page** — `index.html` (自定义 GitHub Pages 入口)，已移除 Jekyll 主题改用纯 HTML
- **DESIGN_ISSUES.md** — 架构评审文档，位于根目录
- **Toollists are platform-tagged** — built-in catalogs in `internal/toollist/builtin/*.yaml` carry `platform: macos|windows`. All list/default entry points filter by `runtime.GOOS` (`toollist.AvailableNames` / `LoadBuiltinNamedFor` / `LoadBuiltinForPlatform`), so a Windows build never surfaces a macOS catalog, and a hand-written URL cannot bypass it. Imported catalogs without `platform` stay available everywhere.
- **Windows install IDs** — Windows catalogs use `winget:` (verified `winget search --exact` ids) instead of `brew:`; the GUI's "设为安装清单" resolves `brew || winget` via `Item.InstallID()`.
