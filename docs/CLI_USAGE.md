# CLI Usage — SwiftInstall (`sis`)

## Overview

SwiftInstall is a cross-platform batch software installer. The primary interface is the `sis` CLI (Go), with legacy `.bat` (Windows) and `.sh` (macOS) scripts as fallbacks.

Only the **winget** backend is implemented today, so `sis` currently works on Windows. Homebrew support is still to come.

## Installation

```bash
# From source
go install github.com/cgartlab/SwiftInstall/cmd/sis@latest

# Or use a pre-built binary
./sis --help
```

## Commands

| Command | Description |
|---|---|
| `sis install` | Install software from a manifest file |
| `sis uninstall` | Uninstall software listed in a manifest (`--yes` required) |
| `sis list` | List the packages in a manifest |
| `sis status` | Check which manifest packages are installed |
| `sis version` | Show version |

Global flags (available on every subcommand):

| Flag | Description |
|---|---|
| `-f, --file string` | Manifest path (default: auto-detect `sis.yaml` / `sis.yml` / `software_list.txt` / `packages.txt`) |
| `--format string` | Output format: `table`, `json`, `silent` (default `table`) |
| `--no-color` | Disable ANSI colours |

## Manifest Format

A manifest is a YAML file, or a plain-text list with one package ID per line.

```yaml
# sis.yaml
proxy: http://127.0.0.1:10809
skip_existing: true
retry_count: 2
retry_delay: 3

packages:
  - id: 7zip.7zip
    category: utils
  - id: Mozilla.Firefox
    category: browser
    optional: true
```

Both the flat layout above and a nested `settings:` block are accepted. The parsed format is decided by **content**, not by extension: a file containing a `packages:` or `settings:` key is YAML even if it is named `.txt`; anything else is read as a one-ID-per-line list. Duplicate IDs are collapsed.

Plain-text list:

```text
# utils
7zip.7zip
# browser
Mozilla.Firefox
```

## Cross-Platform Support

| Platform | Primary | Legacy |
|---|---|---|
| Windows | `sis` CLI (winget backend) | `Windows/software_install.bat` |
| macOS | not implemented yet | `macOS/install_packages.sh` (Homebrew) |
| Linux | not implemented | — |

## Test Manifests

Test fixtures live in the project root as `test_manifest_*` files and are not used at runtime:

| File | Purpose |
|---|---|
| `test_manifest_valid.yaml` | Valid manifest, three packages |
| `test_manifest_dup.yaml` | Duplicate IDs — collapsed to two packages |
| `test_manifest_invalid.yaml` | Contains an empty ID — rejected by validation |
| `test_manifest_empty.yaml` | No packages — rejected by validation |
| `test_manifest_optional.yaml` | Exercises `optional: true` |
| `test_manifest_unknown_fields.yaml` | Unknown keys are ignored |
| `test_manifest_broken.yaml` | Malformed YAML — parse error |
| `test_manifest_txt.txt` | Plain-text list with category comments |

## Project Layout

```
SwiftInstall/
├── cmd/sis/          # CLI entry point
├── internal/
│   ├── cli/          # Cobra commands + global flags
│   ├── engine/       # batch install/uninstall orchestration
│   ├── manifest/     # manifest parsing, dedupe, validation
│   ├── backend/      # Backend interface + winget implementation
│   └── ui/           # terminal / json / silent renderers
├── Windows/          # legacy .bat installer + example manifest
├── macOS/            # legacy .sh installer + example manifest
└── install.ps1       # PowerShell one-line installer
```

## Troubleshooting

- `winget is not installed or not in PATH` — install App Installer from the Microsoft Store.
- `未找到清单文件` — pass `-f <path>` or create one of the auto-detected manifest names.
- `清单文件校验失败` — the manifest has no packages or a package with an empty ID.
- Check `DESIGN_ISSUES.md` for known design decisions and limitations.
- Check `TEST_REPORT.md` for the last recorded manual test run (predates the simplifications described there — parts of it are stale).
