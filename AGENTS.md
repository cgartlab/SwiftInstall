# AGENTS.md — SwiftInstall

**分层**: 产品 (Products) — 个人产品线

Cross-platform batch software installer. 作者开发的五款个人产品之一。 **Legacy scripts** use `winget` (Windows `.bat`) and `brew` (macOS `.sh`). **New Go CLI** (`sis`) replaces them incrementally — Windows first.

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

- **GitHub Actions**: `.github/workflows/` — `release.yml` (发布), `ci.yml` (push 触发), `argus-review.yml` (PR 评审)
- **Push 触发的 CI**: `go build` + version/help smoke test（`windows-latest`）；**尚未接入 `go vet` 与测试**
- **发布触发**: tag push → 构建 + 校验版本一致 + 发布 binary（仅 Windows amd64）

### Entrypoint

`cmd/sis/main.go` — Cobra CLI, ldflags-injected version/commit/date.

### Architecture

```
cmd/sis/main.go   Cobra entrypoint; ldflags inject Version/Commit/Date
internal/
  cli/            Cobra commands (thin orchestration layer)
                  install | uninstall | list | status | version
                  helpers.go — manifest auto-discovery + renderer selection
  engine/         Batch install/uninstall orchestration
                  dedupe, skip-existing, dry-run, retry loop, progress callback
  manifest/       Manifest schema, YAML/TXT parsing, validation
                  manifest.go | parser.go | validate.go
  backend/        Backend interface (Name/Detect/IsInstalled/Install/Uninstall)
                  + the only implementation, winget
  ui/             Renderer interface + terminal | json | silent implementations
```

Earlier waves also shipped `internal/config`, `internal/mirror`, `internal/proxy`,
`internal/log` and the `config`/`mirror` subcommands. They were deleted as
dead weight — `sis` is now flag-driven only, with no config file and no
persistence. Do not reintroduce them without a concrete need.

### Development status

| Wave | Feature | Status |
|------|---------|--------|
| 1 | Foundation (CLI skeleton, types) | Done |
| 2 | Engine + winget backend | Done |
| 3 | ~~Mirror, proxy, preflight~~ | Removed (flag-driven instead) |
| 4 | CLI commands wiring (install, list, uninstall, status) | Done |
| 5 | Polish (progress, colors, UI renderers, CI) | Done |
| 6 | Homebrew backend for macOS | Not started |

### Config precedence

There is no config file. Flags beat manifest settings, which beat built-in
defaults (`install.go` resolves this with `cmd.Flags().Changed`).

## Gotchas

- **`bin/` was deleted** — it held precompiled Windows binaries from an earlier attempt, with no source in this repo. `.gitignore` covers `*.exe` now.
- **Manifest format is decided by content, not extension** — `Windows/software_list.txt` and `macOS/packages.txt` contain YAML. `manifest.ParseManifest` sniffs for a `packages:`/`settings:` key. Do not "fix" these files by renaming without also updating the legacy scripts' expectations.
- **The legacy scripts must stay format-agnostic** — they extract `- id:` lines but still accept plain TXT lists. Either consumer breaking is a bug.
- **Docs are Chinese** — scripts and README in Simplified Chinese. USTC mirror switching lives only in `Windows/software_install.bat`.
- **Preflight/admin detection does not exist** — the only environment check is `backend.Detect()` looking for `winget`. DESIGN_ISSUES.md and TEST_REPORT.md still claim otherwise; treat both as historical.
- **Production landing page** — `index.html` (自定义 GitHub Pages 入口)，已移除 Jekyll 主题改用纯 HTML
- **DESIGN_ISSUES.md** — 架构评审文档，位于根目录
