package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/config"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall packages listed in a manifest",
		Long: `Uninstall all packages listed in a manifest file.
Requires --yes to confirm batch uninstall.

Examples:
  sis uninstall -f software_list.txt --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				return fmt.Errorf("use --yes to confirm batch uninstall")
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			manifestPath, _ := cmd.Flags().GetString("file")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			if manifestPath == "" {
				manifestPath = resolveManifest(cfg.DefaultManifest)
			}
			if manifestPath == "" {
				return fmt.Errorf("no manifest file found")
			}

			winget := backend.NewWingetBackend()
			if err := winget.Detect(); err != nil {
				return fmt.Errorf("winget: %w", err)
			}

			manifest, err := engine.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parse manifest: %w", err)
			}

			eng := engine.New(winget, &engine.Config{
				DryRun:     dryRun,
				RetryCount: config.IntVal(cfg.RetryCount, 2),
				RetryDelay: time.Duration(config.IntVal(cfg.RetryDelaySec, 3)) * time.Second,
			})

			fmt.Fprintf(os.Stderr, "Uninstalling %d packages from %s\n", len(manifest.Packages), manifestPath)
			if dryRun {
				fmt.Fprintf(os.Stderr, "DRY-RUN — no changes will be made\n")
			}

			summary, err := eng.Uninstall(context.Background(), manifest)
			if err != nil {
				return err
			}

			printSummary(summary)
			if summary.Failed > 0 {
				return fmt.Errorf("%d package(s) failed to uninstall", summary.Failed)
			}
			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "path to manifest file")
	cmd.Flags().Bool("yes", false, "confirm batch uninstall")
	cmd.Flags().BoolP("dry-run", "n", false, "preview uninstallation without making changes")
	return cmd
}
