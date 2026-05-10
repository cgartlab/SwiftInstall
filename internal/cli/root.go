package cli

import (
	"github.com/spf13/cobra"
)

func Execute(version, commit, date string) error {
	rootCmd := newRootCmd(version, commit, date)
	return rootCmd.Execute()
}

func newRootCmd(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sis",
		Short: "SwiftInstall - Cross-platform batch software installer",
		Long: `SwiftInstall installs and manages software packages in bulk.
Uses winget on Windows and Homebrew on macOS.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newVersionCmd(version, commit, date))
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}
