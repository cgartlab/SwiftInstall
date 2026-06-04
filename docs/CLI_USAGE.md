# CLI Usage — SwiftInstall (`sis`)

## Overview

SwiftInstall is a cross-platform batch software installer. The primary interface is the `sis` CLI (Go), with legacy `.bat` (Windows) and `.sh` (macOS) scripts as fallbacks.

## Installation

```bash
# From source
go install github.com/cgartlab/SwiftInstall/cmd/sis@latest

# Or use pre-built binary
./sis --help
```

## Commands

| Command | Description |
|---|---|
| `sis install` | Install software from a manifest file |
| `sis list` | List available software packages |
| `sis search <query>` | Search for packages |
| `sis status` | Show installed packages |
| `sis version` | Show version |

## Manifest Format

Software definitions are YAML files. Example:

```yaml
# test-manifest.yaml
software:
  - name: "7zip"
    source: "winget"
    id: "7zip.7zip"
  - name: "Firefox"
    source: "winget"  
    id: "Mozilla.Firefox"
```

## Cross-Platform Support

| Platform | Primary | Legacy |
|---|---|---|
| Windows | `sis` CLI (Go) | `install.bat` (winget) |
| macOS | `sis` CLI (Go) | `install.sh` (Homebrew) |
| Linux | `sis` CLI (Go) | — |

## Test Manifests

Test fixtures are in the project root as `test_manifest_*.yaml` files:

| File | Purpose |
|---|---|
| `test_manifest_minimal.yaml` | Minimum valid manifest |
| `test_manifest_duplicates.yaml` | Duplicate entry handling |
| `test_manifest_txt.txt` | Invalid format test |
| Other `test_manifest_*.yaml` | Various edge cases |

## Project Layout

```
SwiftInstall/
├── cmd/          # CLI entry point
├── internal/     # Core logic (Go packages)
├── bin/          # Build output
├── macOS/        # macOS-specific scripts
├── Windows/      # Windows-specific scripts
├── install.ps1   # PowerShell installer
└── .sis.json     # sis runtime config
```

## Troubleshooting

- Run `sis --debug` for verbose output
- Check `DESIGN_ISSUES.md` for known design decisions and limitations
- Check `TEST_REPORT.md` for test coverage details