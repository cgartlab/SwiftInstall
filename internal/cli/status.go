package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/cgartlab/SwiftInstall/internal/ui"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installation status of packages",
		Long: `Compare packages in a manifest with what is currently installed
and show which are installed, missing, or outdated.

Examples:
  sis status
  sis status -f software_list.txt
  sis status --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestPath, _ := cmd.Flags().GetString("file")
			format, _ := cmd.Flags().GetString("format")

			if manifestPath == "" {
				manifestPath = resolveManifest("")
			}
			if manifestPath == "" {
				return fmt.Errorf("no manifest file found")
			}

			m, err := manifest.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parse manifest: %w", err)
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

			renderer.Start(len(m.Packages), manifestPath, "Checking status of")

			results := make([]engine.InstallResult, 0, len(m.Packages))
			for _, pkg := range m.Packages {
				isInstalled, err := winget.IsInstalled(context.Background(), pkg.ID)
				result := engine.InstallResult{Package: pkg}
				if err != nil {
					result.Status = engine.StatusFailed
					result.Errors = []string{err.Error()}
					renderer.Progress(pkg, string(engine.StatusFailed), err)
				} else if isInstalled {
					result.Status = engine.StatusSuccess
					renderer.Progress(pkg, string(engine.StatusSuccess), nil)
				} else {
					result.Status = engine.StatusSkipped
					renderer.Progress(pkg, string(engine.StatusSkipped), nil)
				}
				results = append(results, result)
			}

			now := time.Now()
			summary := &engine.Summary{
				Total:     len(results),
				Results:   results,
				StartTime: now,
				EndTime:   now,
			}
			for _, r := range results {
				switch r.Status {
				case engine.StatusSuccess:
					summary.Succeeded++
				case engine.StatusSkipped:
					summary.Skipped++
				case engine.StatusFailed:
					summary.Failed++
				}
			}

			renderer.Done(summary)
			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "path to manifest file")
	cmd.Flags().String("format", "table", "output format: table, json, silent")
	return cmd
}
