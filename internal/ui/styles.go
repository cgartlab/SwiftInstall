package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"swiftinstall/internal/appinfo"
)

const (
	ColorPrimary       = "#c7894e"
	ColorPrimaryBright = "#d9a56f"
	ColorSecondary     = "#2f3338"
	ColorAccent        = "#6da874"
	ColorWarning       = "#c99b67"
	ColorError         = "#ef4444"
	ColorInfo          = "#7f9ab5"
	ColorMuted         = "#6b7280"
	ColorText          = "#f8fafc"
	ColorBackground    = "#1e293b"
)

var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText))

	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			Padding(0, 1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted)).
			Italic(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent)).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning)).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError)).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo))

	HighlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimaryBright)).
			Bold(true)

	MenuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			Padding(0, 1)

	MenuDescriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorMuted)).
				PaddingLeft(4)

	MenuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPrimaryBright)).
				Bold(true).
				Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPrimary)).
			Padding(1, 2)

	BoxActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorAccent)).
			Padding(1, 2)

	StatusSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent))

	StatusFailed = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError))

	StatusPending = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	StatusInstalling = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorInfo))

	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPrimary))

	ProgressCompleteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorAccent))

	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPrimaryBright)).
				Bold(true)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText))

	InputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(ColorPrimary))

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	LogoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	DividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	KeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	CmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent))

	SectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimaryBright)).
			Bold(true).
			Underline(true)
)

func PrintWelcomeScreen(version string) {
	fmt.Println(GetCompactLogo())
	fmt.Println()

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorMuted))
	fmt.Println(infoStyle.Render(fmt.Sprintf("v%s | %s", version, appinfo.Author)))
	fmt.Println()

	commands := []struct {
		cmd  string
		desc string
	}{
		{"install", "Install software packages"},
		{"uninstall", "Uninstall software packages"},
		{"search", "Search for packages"},
		{"list", "List configured packages"},
		{"config", "Manage configuration"},
		{"status", "Show system status"},
		{"about", "About SwiftInstall"},
	}

	fmt.Println(SectionStyle.Render("Commands"))
	fmt.Println()

	maxCmdLen := 12
	for _, c := range commands {
		cmdStr := KeyStyle.Render(fmt.Sprintf("  %-"+fmt.Sprintf("%d", maxCmdLen)+"s", c.cmd))
		descStr := HelpStyle.Render(c.desc)
		fmt.Printf("%s %s\n", cmdStr, descStr)
	}

	fmt.Println()
	fmt.Println(HelpStyle.Render("Run 'sis help' for more information"))
	fmt.Println()
}

// GetStatusStyle 根据状态获取样式
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "success", "completed", "installed":
		return StatusSuccess
	case "failed", "error":
		return StatusFailed
	case "pending", "waiting":
		return StatusPending
	case "installing", "running", "downloading":
		return StatusInstalling
	default:
		return StatusPending
	}
}

// GetStatusIcon 根据状态获取图标
func GetStatusIcon(status string) string {
	switch status {
	case "success", "completed", "installed":
		return "✓"
	case "failed", "error":
		return "✗"
	case "pending", "waiting":
		return "○"
	case "installing", "running":
		return "◉"
	case "downloading":
		return "↓"
	case "skipped":
		return "⊘"
	default:
		return "○"
	}
}
