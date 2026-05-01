package cli

import (
	"fmt"

	"github.com/cgartlab/SwiftInstall/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Get and set configuration values.`,
	}

	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			val, err := config.Get(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if err := config.Set(cfg, args[0], args[1]); err != nil {
				return err
			}

			local, _ := cmd.Flags().GetBool("local")
			if err := config.Save(cfg, config.SaveOpts{Local: local}); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolP("local", "l", false, "save to local .sis.json instead of global config")
	return cmd
}
