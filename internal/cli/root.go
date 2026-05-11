package cli

import (
	"github.com/spf13/cobra"
)

// Execute is the main entry point for the CLI.
func Execute(version, commit, date string) error {
	rootCmd := newRootCmd(version, commit, date)
	return rootCmd.Execute()
}

func newRootCmd(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sis",
		Short: "SwiftInstall — 跨平台批量软件安装工具",
		Long: `SwiftInstall (sis) 帮助你在新系统上一键安装所有常用软件。

支持从 YAML 或 TXT 清单文件批量安装/卸载软件包，
内置国内镜像加速和代理支持，开箱即用。

常用命令:
  sis install          从清单批量安装软件
  sis uninstall        从清单批量卸载软件
  sis list             查看清单中的软件列表
  sis status           检查各软件的安装状态

使用 "sis <command> --help" 查看各命令的详细用法。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	cmd.PersistentFlags().StringP("file", "f", "", "清单文件路径 (默认自动查找 sis.yaml / software_list.txt)")
	cmd.PersistentFlags().String("format", "table", "输出格式: table, json, silent")
	cmd.PersistentFlags().Bool("no-color", false, "禁用彩色输出")

	// Register subcommands
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newVersionCmd(version, commit, date))

	return cmd
}
