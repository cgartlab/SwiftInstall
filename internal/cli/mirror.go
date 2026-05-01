package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/cgartlab/SwiftInstall/internal/mirror"
	"github.com/spf13/cobra"
)

func newMirrorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Manage package mirror sources",
		Long: `Switch between package mirror sources for faster downloads.
Supports 'ustc' (USTC mirror for China) and 'official'.

Examples:
  sis mirror ustc
  sis mirror official
  sis mirror --status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			showStatus, _ := cmd.Flags().GetBool("status")
			reset, _ := cmd.Flags().GetBool("reset")

			if showStatus {
				current := mirror.Current()
				fmt.Fprintf(os.Stderr, "Current mirror: %s\n", current)
				fmt.Fprintf(os.Stderr, "Supported mirrors: %s\n", strings.Join(mirror.Supported(), ", "))
				return nil
			}

			if reset {
				if err := mirror.Reset(); err != nil {
					return fmt.Errorf("reset mirror: %w", err)
				}
				fmt.Fprintln(os.Stderr, "Mirror reset to official source")
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("no mirror specified; use one of: %s", strings.Join(mirror.Supported(), ", "))
			}

			name := args[0]
			if err := mirror.Set(name); err != nil {
				return fmt.Errorf("set mirror: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Mirror switched to: %s\n", name)
			return nil
		},
	}

	cmd.Flags().Bool("status", false, "show current mirror")
	cmd.Flags().Bool("reset", false, "reset to official source")
	return cmd
}
