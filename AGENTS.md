# AGENTS.md — SwiftInstall

Cross-platform (Win/macOS) batch software installer. Uses `winget` (Windows) and `brew` (macOS) to install packages from manifest files.

## No build / test / lint / CI

There is zero build pipeline, test framework, linter, formatter, typechecker, or CI. Nothing to run.

## Entrypoints

| Platform | Script | Manifest |
|----------|--------|----------|
| Windows | `Windows/software_install.bat` | `Windows/software_list.txt` |
| Windows (proxy) | `Windows/software_install_proxy.bat` — requires v2rayN on `127.0.0.1:10809` | same |
| Windows (mirror) | `Windows/switch_winget_to_USTCsource.bat` — replaces winget source with USTC mirror for China | — |
| macOS | `macOS/install_packages.sh` | `macOS/packages.txt` |

Run any script directly — no deps to install.

## Gotchas

- **`src/` directory** — 13 empty Go-pattern subdirectories (`cli/`, `commands/`, `config/`, `core/`, etc.) exist on disk but are **untracked** by Git. No Go source code lives in the repo. Ignore.
- **`bin/*.exe`** — precompiled Windows binaries (`sis-windows-amd64.exe`). No source code in this repo. Ignore.
- **Docs are Chinese** — scripts and README are in Simplified Chinese. USTC mirror is a core China-user feature.
- **No error recovery** — scripts do not handle individual install failures gracefully.
- **No uninstall** — scripts only install, no rollback.
