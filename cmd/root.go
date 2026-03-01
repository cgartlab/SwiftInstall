package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"swiftinstall/internal/appinfo"
	"swiftinstall/internal/config"
	"swiftinstall/internal/i18n"
	"swiftinstall/internal/installer"
	"swiftinstall/internal/ui"
)

var (
	version  = "dev"
	commit   = "unknown"
	date     = "unknown"
	cfgFile  string
	language string
)

var rootCmd = &cobra.Command{
	Use:   "sis",
	Short: i18n.T("app_short_desc"),
	Long:  ui.GetLogo() + "\n" + i18n.T("app_long_desc"),
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintWelcomeScreen(version)
		runStartupChecks()
		fmt.Print(ui.InfoStyle.Render("Launch interactive menu? [Y/n]: "))
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			log.Printf("Warning: failed to read user input: %v", err)
			fmt.Println()
			fmt.Println(ui.InfoStyle.Render("Launching interactive menu..."))
			fmt.Println()
			runInteractiveTUI()
			return
		}
		if response == "" || response == "y" || response == "Y" {
			runInteractiveTUI()
		}
	},
}

func hasHelpArg(args []string) bool {
	return len(args) == 1 && strings.EqualFold(args[0], "help")
}

func showCommandHelpIfRequested(cmd *cobra.Command, args []string) bool {
	if !hasHelpArg(args) {
		return false
	}
	if err := cmd.Help(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	return true
}

func printComprehensiveHelp() {
	fmt.Println(ui.GetCompactLogo())
	fmt.Println()

	fmt.Println(ui.SectionStyle.Render("Usage"))
	fmt.Println()
	fmt.Println(ui.HelpStyle.Render("  sis <command> [arguments] [flags]"))
	fmt.Println()

	fmt.Println(ui.SectionStyle.Render("Commands"))
	fmt.Println()

	commands := []struct {
		cmd  string
		args string
		desc string
	}{
		{"install", "[package...]", "Install packages from config or by ID"},
		{"uninstall", "[package...]", "Uninstall packages from config or by ID"},
		{"uninstall-all", "", "Uninstall all configured packages"},
		{"search", "<query>", "Search for packages"},
		{"list", "", "List configured packages"},
		{"config", "", "Open configuration manager"},
		{"edit-list", "", "Edit package list in default editor"},
		{"wizard", "", "Launch interactive setup wizard"},
		{"batch", "[file]", "Batch install from file or config"},
		{"status", "", "Show system status and installed packages"},
		{"update", "", "Check for SwiftInstall updates"},
		{"clean", "", "Clean package manager cache"},
		{"about", "", "Show project and author information"},
		{"version", "", "Show version information"},
		{"help", "", "Show this help"},
	}

	for _, c := range commands {
		cmdPart := ui.KeyStyle.Render(fmt.Sprintf("  %-14s", c.cmd))
		argsPart := ""
		if c.args != "" {
			argsPart = ui.InfoStyle.Render(c.args)
		}
		descPart := ui.HelpStyle.Render("  " + c.desc)
		fmt.Printf("%s %s%s\n", cmdPart, argsPart, descPart)
	}

	fmt.Println()
	fmt.Println(ui.SectionStyle.Render("Flags"))
	fmt.Println()
	fmt.Println(ui.HelpStyle.Render("  --config <path>   Use specified config file"))
	fmt.Println(ui.HelpStyle.Render("  -l, --lang <en|zh> Set interface language"))
	fmt.Println()

	fmt.Println(ui.SectionStyle.Render("Examples"))
	fmt.Println()
	fmt.Println(ui.HelpStyle.Render("  sis install                    # Install from config"))
	fmt.Println(ui.HelpStyle.Render("  sis install Git.Git            # Install specific package"))
	fmt.Println(ui.HelpStyle.Render("  sis search vscode              # Search for VS Code"))
	fmt.Println(ui.HelpStyle.Render("  sis config                     # Open config manager"))
	fmt.Println(ui.HelpStyle.Render("  sis wizard                     # Launch setup wizard"))
	fmt.Println(ui.HelpStyle.Render("  sis edit-list                  # Edit package list"))
	fmt.Println()

	fmt.Println(ui.SectionStyle.Render("Keyboard Shortcuts"))
	fmt.Println()
	fmt.Println(ui.HelpStyle.Render("  ↑/↓ or j/k    Navigate menu"))
	fmt.Println(ui.HelpStyle.Render("  Enter         Select/Confirm"))
	fmt.Println(ui.HelpStyle.Render("  Space         Toggle selection (in wizard)"))
	fmt.Println(ui.HelpStyle.Render("  d             View details (in search)"))
	fmt.Println(ui.HelpStyle.Render("  i             Quick install (in search)"))
	fmt.Println(ui.HelpStyle.Render("  a             Add package (in config)"))
	fmt.Println(ui.HelpStyle.Render("  e             Edit package (in config)"))
	fmt.Println(ui.HelpStyle.Render("  r/d           Remove package (in config)"))
	fmt.Println(ui.HelpStyle.Render("  q/Esc         Back or quit"))
	fmt.Println()

	fmt.Println(ui.HelpStyle.Render(appinfo.Copyright))
}

func runStartupChecks() {
	handleAutoUpdatePreference()
	if config.GetBool("auto_update_check") {
		runAutomaticUpdateCheck()
	}
}

func handleAutoUpdatePreference() {
	if config.GetBool("auto_update_prompted") {
		return
	}
	fmt.Print(ui.InfoStyle.Render("Enable automatic update check on startup? [Y/n]: "))
	var ans string
	if _, err := fmt.Scanln(&ans); err != nil {
		ans = ""
	}
	enabled := ans == "" || strings.EqualFold(ans, "y") || strings.EqualFold(ans, "yes")
	if err := config.SetAndSave("auto_update_check", enabled); err != nil {
		log.Printf("Warning: failed to persist auto_update_check: %v", err)
	}
	if err := config.SetAndSave("auto_update_prompted", true); err != nil {
		log.Printf("Warning: failed to persist auto_update_prompted: %v", err)
	}
}

func runAutomaticUpdateCheck() {
	fmt.Println(ui.InfoStyle.Render("Auto update check..."))
	inst := installer.NewInstaller()
	if inst == nil {
		fmt.Println(ui.WarningStyle.Render("Skipped: unsupported platform for package manager update check"))
		return
	}
	if err := inst.Update(); err != nil {
		fmt.Println(ui.WarningStyle.Render("Update check finished with warnings: " + err.Error()))
		return
	}
	fmt.Println(ui.SuccessStyle.Render("✓ Package manager metadata is up to date"))
	fmt.Println(ui.HelpStyle.Render("Checked at: " + time.Now().Format(time.RFC3339)))
}

func ensureEnvironmentReady() bool {
	report := installer.CheckEnvironment()
	if report.Ready {
		if !config.GetBool("env_checked") {
			_ = config.SetAndSave("env_checked", true)
		}
		return true
	}
	fmt.Println(ui.ErrorStyle.Render("Environment check failed:"))
	for _, d := range report.Details {
		fmt.Println("  -", d)
	}
	fmt.Println(ui.WarningStyle.Render("Please fix environment issues before installation/search."))
	return false
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", i18n.T("flag_config"))
	rootCmd.PersistentFlags().StringVarP(&language, "lang", "l", "", i18n.T("flag_language"))

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(wizardCmd)
	rootCmd.AddCommand(batchCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(aboutCmd)
	rootCmd.AddCommand(helpDocCmd)
	rootCmd.AddCommand(uninstallAllCmd)
	rootCmd.AddCommand(editListCmd)

	exportCmd.Flags().StringP("format", "f", "json", i18n.T("flag_export_format"))
	exportCmd.Flags().StringP("output", "o", "", i18n.T("flag_export_output"))
	batchCmd.Flags().BoolP("parallel", "p", true, i18n.T("flag_parallel"))
}

func initConfig() {
	if cfgFile != "" {
		config.SetConfigFile(cfgFile)
	}
	config.Init()

	if language != "" {
		i18n.SetLanguage(language)
	} else if lang := config.GetString("language"); lang != "" {
		i18n.SetLanguage(lang)
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T("cmd_version_short"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ui.ColorPrimary)).
			Bold(true)

		fmt.Println(ui.GetLogo())
		fmt.Println()
		fmt.Println(style.Render(fmt.Sprintf("Version: %s", version)))
		fmt.Println(style.Render(fmt.Sprintf("Commit:  %s", commit)))
		fmt.Println(style.Render(fmt.Sprintf("Date:    %s", date)))
		fmt.Println(style.Render(fmt.Sprintf("Go:      %s", runtime.Version())))
		fmt.Println(style.Render(fmt.Sprintf("OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)))
	},
}

var installCmd = &cobra.Command{
	Use:   "install [package...]",
	Short: i18n.T("cmd_install_short"),
	Long:  i18n.T("cmd_install_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		if len(args) == 0 {
			runInstallFromConfig()
		} else {
			runInstallPackages(args)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [package...]",
	Short: i18n.T("cmd_uninstall_short"),
	Long:  i18n.T("cmd_uninstall_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		if len(args) == 0 {
			runUninstallFromConfig()
		} else {
			runUninstallPackages(args)
		}
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: i18n.T("cmd_search_short"),
	Long:  i18n.T("cmd_search_long"),
	Args: func(cmd *cobra.Command, args []string) error {
		if hasHelpArg(args) {
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runSearch(args[0])
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T("cmd_list_short"),
	Long:  i18n.T("cmd_list_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runList()
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.T("cmd_config_short"),
	Long:  i18n.T("cmd_config_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runConfig()
	},
}

var wizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: i18n.T("cmd_wizard_short"),
	Long:  i18n.T("cmd_wizard_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runWizard()
	},
}

var batchCmd = &cobra.Command{
	Use:   "batch [file]",
	Short: i18n.T("cmd_batch_short"),
	Long:  i18n.T("cmd_batch_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		if len(args) > 0 {
			runBatchFromFile(args[0])
		} else {
			runBatchFromConfig()
		}
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: i18n.T("cmd_export_short"),
	Long:  i18n.T("cmd_export_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		runExport(format, output)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: i18n.T("cmd_update_short"),
	Long:  i18n.T("cmd_update_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runUpdate()
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: i18n.T("cmd_clean_short"),
	Long:  i18n.T("cmd_clean_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runClean()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: i18n.T("cmd_status_short"),
	Long:  i18n.T("cmd_status_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runStatus()
	},
}

var aboutCmd = &cobra.Command{
	Use:   "about",
	Short: i18n.T("cmd_about_short"),
	Long:  i18n.T("cmd_about_long"),
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		ui.RunAbout()
	},
}

var uninstallAllCmd = &cobra.Command{
	Use:   "uninstall-all",
	Short: "一键卸载配置内所有软件",
	Long:  "一键卸载配置内所有软件（跨平台）",
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runUninstallFromConfig()
	},
}

var editListCmd = &cobra.Command{
	Use:   "edit-list",
	Short: "自由编辑软件安装列表",
	Long:  "在默认编辑器中直接编辑配置文件的软件安装列表",
	Run: func(cmd *cobra.Command, args []string) {
		if showCommandHelpIfRequested(cmd, args) {
			return
		}
		runEditSoftwareList()
	},
}

var helpDocCmd = &cobra.Command{
	Use:   "help",
	Short: i18n.T("cmd_help_short"),
	Long:  i18n.T("cmd_help_long"),
	Run: func(cmd *cobra.Command, args []string) {
		printComprehensiveHelp()
	},
}

func runInteractiveTUI() {
	ui.RunMainMenu()
}

func runInstallFromConfig() {
	if !ensureEnvironmentReady() {
		os.Exit(1)
	}
	cfg := config.Get()
	packages := cfg.GetSoftwareList()
	if len(packages) == 0 {
		fmt.Println(ui.WarningStyle.Render(i18n.T("warn_no_packages")))
		os.Exit(1)
	}
	ui.RunInstall(packages, false)
}

func runInstallPackages(packages []string) {
	if !ensureEnvironmentReady() {
		os.Exit(1)
	}
	ui.RunInstallByName(packages, false)
}

func runUninstallFromConfig() {
	cfg := config.Get()
	packages := cfg.GetSoftwareList()
	if len(packages) == 0 {
		fmt.Println(ui.WarningStyle.Render(i18n.T("warn_no_packages")))
		os.Exit(1)
	}
	ui.RunUninstall(packages)
}

func runUninstallPackages(packages []string) {
	ui.RunUninstallByName(packages)
}

func runSearch(query string) {
	if !ensureEnvironmentReady() {
		os.Exit(1)
	}
	ui.RunSearch(query)
}

func runList() {
	cfg := config.Get()
	packages := cfg.GetSoftwareList()
	ui.ShowPackageList(packages)
}

func runConfig() {
	ui.RunConfigManager()
}

func runWizard() {
	ui.RunWizard()
}

func runBatchFromFile(file string) {
	ui.RunBatchFromFile(file)
}

func runBatchFromConfig() {
	cfg := config.Get()
	packages := cfg.GetSoftwareList()
	ui.RunBatch(packages, true)
}

func runExport(format, output string) {
	cfg := config.Get()
	packages := cfg.GetSoftwareList()
	ui.RunExport(packages, format, output)
}

func runEditSoftwareList() {
	cfg := config.Get()
	path := cfg.GetConfigPath()
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(ui.ErrorStyle.Render("Failed to open editor: " + err.Error()))
		return
	}
	config.Reload()
	fmt.Println(ui.SuccessStyle.Render("✓ software list updated"))
}

func runUpdate() {
	ui.RunUpdateCheck()
}

func runClean() {
	ui.RunClean()
}

func runStatus() {
	ui.RunStatus()
}
