package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/config"
	"github.com/cgartlab/SwiftInstall/internal/engine"
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
  sis status -f software_list.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			manifestPath, _ := cmd.Flags().GetString("file")
			if manifestPath == "" {
				manifestPath = resolveManifest(cfg.DefaultManifest)
			}
			if manifestPath == "" {
				return fmt.Errorf("no manifest file found")
			}

			manifest, err := engine.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parse manifest: %w", err)
			}

			winget := backend.NewWingetBackend()
			if err := winget.Detect(); err != nil {
				return fmt.Errorf("winget: %w", err)
			}

			eng := engine.New(winget, &engine.Config{})
			statuses, err := eng.CheckStatus(context.Background(), manifest)
			if err != nil {
				return fmt.Errorf("check status: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Status for %d packages from %s\n\n", len(statuses), manifestPath)

			installed := 0
			missing := 0
			for _, s := range statuses {
				icon := "✓"
				if !s.Installed {
					icon = "✗"
					missing++
				} else {
					installed++
				}
				fmt.Fprintf(os.Stderr, "  %s %s\n", icon, s.Package.ID)
			}

			fmt.Fprintf(os.Stderr, "\n%d installed, %d missing\n", installed, missing)
			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "path to manifest file")
	return cmd
}
