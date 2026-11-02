# CLI Usage — SwiftInstall (`sis`)

## Overview

SwiftInstall is a cross-platform batch software installer. The primary interface is the `sis` CLI (Go), with legacy `.bat` (Windows) and `.sh` (macOS) scripts as fallbacks.

Two backends ship today, selected automatically by operating system:

- **macOS** → Homebrew (formula + cask)
- **Windows** → winget

Linux builds are produced, but no package backend is available there yet (running package commands returns a clear "platform not supported" error).

## Installation

```bash
# macOS one-liner
curl -fsSL https://raw.githubusercontent.com/cgartlab/SwiftInstall/main/install.sh | sh

# From source
go install github.com/cgartlab/SwiftInstall/cmd/sis@latest
```

## Commands

| Command | Description |
|---|---|
| `sis tui` | Interactive terminal UI for browsing the manifest and installing |
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
# packages.txt (YAML content)
proxy: http://127.0.0.1:7890
skip_existing: true
retry_count: 2
retry_delay: 3

packages:
  - id: git
    category: Dev
  - id: python@3.12
    category: Dev
  - id: visual-studio-code
    category: Dev
    kind: cask
  - id: figma
    category: Design
    kind: cask
    optional: true
```

Both the flat layout above and a nested `settings:` block are accepted. The parsed format is decided by **content**, not by extension: a file containing a `packages:` or `settings:` key is YAML even if it is named `.txt`; anything else is read as a one-ID-per-line list. Duplicate IDs are collapsed.

### Package fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Package ID / Homebrew token (required) |
| `category` | string | Grouping for `sis list` |
| `kind` | string | `formula` or `cask` (Homebrew only; empty = auto-detect, formula first, then cask) |
| `optional` | bool | Failure does not count as `failed` and does not affect the exit code |

Plain-text list:

```text
# Dev
git
python@3.12
# Design
visual-studio-code
```

## Cross-Platform Support

| Platform | Primary | Legacy |
|---|---|---|
| macOS | `sis` CLI (Homebrew backend, formula + cask) | `macOS/install_packages.sh` (Homebrew) |
| Windows | `sis` CLI (winget backend) | `Windows/software_install.bat` |
| Linux | build artifacts only — no package backend yet | — |

The macOS sample manifest lives at [`macOS/packages.txt`](../macOS/packages.txt); it uses real lowercase Homebrew tokens and marks GUI apps with `kind: cask`.

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
│   ├── cli/          # Cobra commands + global flags + tui
│   ├── engine/       # batch install/uninstall orchestration
│   ├── manifest/     # manifest parsing, dedupe, validation
│   ├── backend/      # Backend interface + registry + homebrew + winget
│   └── ui/           # terminal / json / silent renderers
├── Windows/          # legacy .bat installer + example manifest
├── macOS/            # legacy .sh installer + example manifest
├── install.sh        # macOS one-line installer
└── install.ps1       # Windows one-line installer
```

## Troubleshooting

- Homebrew not found on macOS — install it from <https://brew.sh/> and ensure `brew` is on `PATH`.
- `winget is not installed or not in PATH` — install App Installer from the Microsoft Store.
- `未找到清单文件` — pass `-f <path>` or create one of the auto-detected manifest names.
- `清单文件校验失败` — the manifest has no packages or a package with an empty ID.
- `No available formula or cask` on macOS — check the token; GUI apps need `kind: cask`.
- Check `DESIGN_ISSUES.md` for historical design decisions and limitations.
