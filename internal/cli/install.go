package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/cgartlab/SwiftInstall/internal/ui"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install packages from a manifest file",
		Long: `Install all packages listed in a YAML or TXT manifest file.
Uses winget on Windows and Homebrew on macOS.

Failing one package does not stop the batch — errors are collected
and reported in a summary at the end.

Examples:
  sis install
  sis install -f software_list.txt
  sis install --dry-run
  sis install --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestPath, _ := cmd.Flags().GetString("file")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			format, _ := cmd.Flags().GetString("format")

			if manifestPath == "" {
				manifestPath = resolveManifest("")
			}
			if manifestPath == "" {
				return fmt.Errorf("no manifest file found; use --file or create software_list.txt or sis.yaml")
			}

			m, err := manifest.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parse manifest: %w", err)
			}

			if err := manifest.Validate(m); err != nil {
				return fmt.Errorf("validate manifest: %w", err)
			}

			winget := backend.NewWingetBackend()
			if err := winget.Detect(); err != nil {
				return fmt.Errorf("winget: %w", err)
			}

			var renderer ui.Renderer
			switch format {
			case "json":
				renderer = ui.NewJSONRenderer()
			case "silent":
				renderer = ui.NewSilentRenderer()
			case "table":
				renderer = ui.NewTerminalRenderer()
			default:
				return fmt.Errorf("invalid format %q: supported values are table, json, silent", format)
			}

			opts := engine.InstallOptions{
				DryRun:       dryRun,
				SkipExisting: m.Settings.SkipExisting,
				Proxy:        m.Settings.Proxy,
			}

			eng := engine.New(winget, opts)

			eng.SetProgressHook(func(r engine.InstallResult) {
				var err error
				if len(r.Errors) > 0 {
					err = fmt.Errorf("%s", r.Errors[0])
				}
				renderer.Progress(r.Package, string(r.Status), err)
			})

			renderer.Start(len(m.Packages), manifestPath, "Installing")
			if dryRun {
				fmt.Fprintf(os.Stderr, "DRY-RUN — no changes will be made\n")
			}

			summary, err := eng.Install(context.Background(), m)
			if err != nil {
				return err
			}

			renderer.Done(summary)

			if summary != nil && summary.Failed > 0 {
				return fmt.Errorf("%d package(s) failed", summary.Failed)
			}
			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "path to manifest file")
	cmd.Flags().BoolP("dry-run", "n", false, "preview installation without making changes")
	cmd.Flags().String("format", "table", "output format: table, json, silent")
	return cmd
}

func resolveManifest(defaultPath string) string {
	if defaultPath != "" {
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	for _, name := range []string{"sis.yaml", "software_list.txt", "packages.txt"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}
