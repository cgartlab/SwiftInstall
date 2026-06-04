# AGENTS.md — SwiftInstall

**分层**: 产品 (Products) — 个人产品线

Cross-platform batch software installer. 作者开发的五款个人产品之一。 **Legacy scripts** use `winget` (Windows `.bat`) and `brew` (macOS `.sh`). **New Go CLI** (`sis`) replaces them incrementally — Windows first.

## Legacy scripts (still present, being replaced)

| Platform | Script | Manifest |
|----------|--------|----------|
| Windows | `Windows/software_install.bat` | `Windows/software_list.txt` |
| Windows (proxy) | `Windows/software_install_proxy.bat` — requires v2rayN on `127.0.0.1:10809` | same |
| Windows (mirror) | `Windows/switch_winget_to_USTCsource.bat` | — |
| macOS | `macOS/install_packages.sh` | `macOS/packages.txt` |

## New Go CLI (`sis`)

### Build

```powershell
go build -o sis.exe ./cmd/sis/
go vet ./...
```

### CI/CD

- **GitHub Actions**: `.github/workflows/` — `release.yml` (发布), `ci.yml` (push 触发)
- **Push 触发的 CI**: lint + vet + build 校验
- **发布触发**: tag push → 构建 + 发布 binary

### Entrypoint

`cmd/sis/main.go` — Cobra CLI, ldflags-injected version/commit/date.

### Architecture

```
cmd/sis/main.go
internal/
  cli/          Cobra commands (thin orchestration layer)
                 install | list | uninstall | status | version
  engine/       Install engine + manifest parser + pre-flight checks
                 major refactor (+294 lines), supports list/uninstall
  backend/      Backend interfaces + winget implementation
  ui/           Output renderers: terminal (+147 lines), json, silent
  config/       Two-level JSON config (~/.sis/config.json + .sis.json)
  mirror/       USTC mirror source switching
  proxy/        v2rayN proxy detection
  log/          Leveled logger (slog, terminal + JSON file)
```

### Development status

| Wave | Feature | Status |
|------|---------|--------|
| 1 | Foundation (CLI skeleton, config, logger, types) | Done |
| 2 | Engine + winget backend | Done |
| 3 | Mirror, proxy, preflight | Done |
| 4 | CLI commands wiring (install, list, uninstall, status) | Done |
| 5 | Polish (progress, colors, UI renderers, CI) | Done |

### Config precedence

CLI flags > local `.sis.json` > global `~/.sis/config.json` > built-in defaults.
Boolean fields use `*bool` tri-state pointers for proper overriding.

### Config validation

`Validate()` checks LogLevel, Color, RetryCount, RetryDelaySec, Mirror, Proxy.
Called on `Save()` (rejects invalid) and sanitized on `Load()` (resets to defaults).

### Windows admin detection

Uses `golang.org/x/sys/windows` token elevation check (`TokenElevationTypeFull`).
Fallback: `SIS_ADMIN_CHECK` env var for testing.

## Gotchas

- **`src/` directory** — 13 empty Go-pattern subdirectories, **untracked** by Git. Dead scaffolding. Ignore.
- **`bin/*.exe`** — precompiled Windows binaries from an earlier attempt. No source in this repo. Ignore.
- **Docs are Chinese** — scripts and README in Simplified Chinese. USTC mirror is a core China-user feature.
- **Production landing page** — `index.html` (自定义 GitHub Pages 入口)，已移除 Jekyll 主题改用纯 HTML
- **DESIGN_ISSUES.md** — 架构评审文档，位于根目录
