package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cgartlab/SwiftInstall/internal/config"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/spf13/cobra"
)

func stringsEqualFoldContains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List packages from a manifest file",
		Long: `Display the contents of a manifest file in table or JSON format.
Optionally filter by category.

Examples:
  sis list
  sis list -f software_list.txt
  sis list --category dev
  sis list --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			manifestPath, _ := cmd.Flags().GetString("file")
			category, _ := cmd.Flags().GetString("category")
			format, _ := cmd.Flags().GetString("format")

			if manifestPath == "" {
				manifestPath = resolveManifest(cfg.DefaultManifest)
			}
			if manifestPath == "" {
				return fmt.Errorf("no manifest file found; use --file or create software_list.txt or sis.yaml")
			}

			manifest, err := engine.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parse manifest: %w", err)
			}

			var filtered []engine.Package
			for _, p := range manifest.Packages {
				if category != "" && !stringsEqualFoldContains(p.Category, category) {
					continue
				}
				filtered = append(filtered, p)
			}

			switch format {
			case "json":
				out, err := json.MarshalIndent(filtered, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal JSON: %w", err)
				}
				fmt.Println(string(out))
			case "table":
				fmt.Fprintf(os.Stderr, "Packages from %s\n\n", manifestPath)
				if len(filtered) == 0 {
					fmt.Fprintln(os.Stderr, "  (no packages)")
				} else {
					for _, p := range filtered {
						if p.Category != "" {
							fmt.Fprintf(os.Stderr, "  %-30s  %s\n", p.ID, p.Category)
						} else {
							fmt.Fprintf(os.Stderr, "  %s\n", p.ID)
						}
					}
				}
				fmt.Fprintf(os.Stderr, "\nTotal: %d packages\n", len(filtered))
			default:
				return fmt.Errorf("unsupported format %q; use 'table' or 'json'", format)
			}

			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "path to manifest file")
	cmd.Flags().StringP("category", "c", "", "filter by category")
	cmd.Flags().String("format", "table", "output format: table or json")
	return cmd
}
