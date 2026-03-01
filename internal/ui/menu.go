package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swiftinstall/internal/config"
	"swiftinstall/internal/i18n"
)

type MenuItem struct {
	Title       string
	Description string
	Icon        string
	Action      func()
}

func (i MenuItem) FilterValue() string { return i.Title }

type MainMenuModel struct {
	list     list.Model
	quitting bool
	width    int
	height   int
}

func NewMainMenu() MainMenuModel {
	items := []list.Item{
		MenuItem{
			Title:       i18n.T("menu_install"),
			Description: i18n.T("cmd_install_long"),
			Icon:        "⚡",
			Action:      func() { RunInstall(config.Get().GetSoftwareList(), false) },
		},
		MenuItem{
			Title:       i18n.T("menu_search"),
			Description: i18n.T("cmd_search_long"),
			Icon:        "🔍",
			Action:      func() { RunSearch("") },
		},
		MenuItem{
			Title:       i18n.T("menu_uninstall"),
			Description: i18n.T("cmd_uninstall_long"),
			Icon:        "🗑",
			Action:      func() { RunUninstall(config.Get().GetSoftwareList()) },
		},
		MenuItem{
			Title:       i18n.T("menu_config"),
			Description: i18n.T("cmd_config_long"),
			Icon:        "⚙",
			Action:      func() { RunConfigManager() },
		},
		MenuItem{
			Title:       i18n.T("menu_status"),
			Description: i18n.T("cmd_status_long"),
			Icon:        "📊",
			Action:      func() { RunStatus() },
		},
		MenuItem{
			Title:       i18n.T("menu_about"),
			Description: i18n.T("menu_about_desc"),
			Icon:        "ℹ",
			Action:      func() { RunAbout() },
		},
		MenuItem{
			Title:       i18n.T("menu_exit"),
			Description: i18n.T("menu_exit_desc"),
			Icon:        "✕",
			Action:      func() { os.Exit(0) },
		},
	}

	l := list.New(items, menuItemDelegate{}, 50, 16)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = TitleStyle
	l.Styles.PaginationStyle = HelpStyle
	l.Styles.HelpStyle = HelpStyle

	return MainMenuModel{list: l}
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.list.CursorUp()
			return m, nil
		case "down", "j":
			m.list.CursorDown()
			return m, nil
		case "enter":
			if item, ok := m.list.SelectedItem().(MenuItem); ok {
				item.Action()
			}
		case "i":
			RunInstall(config.Get().GetSoftwareList(), false)
		case "s":
			RunSearch("")
		case "c":
			RunConfigManager()
		case "a":
			RunAbout()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m MainMenuModel) View() string {
	if m.quitting {
		return "\n  " + i18n.T("menu_exit") + "\n"
	}

	logo := GetCompactLogo()
	menu := m.list.View()

	helpItems := []string{
		KeyStyle.Render("↑/↓") + HelpStyle.Render(" navigate"),
		KeyStyle.Render("Enter") + HelpStyle.Render(" select"),
		KeyStyle.Render("q") + HelpStyle.Render(" quit"),
	}
	helpText := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted)).Render(
		"  " + helpItems[0] + "  " + helpItems[1] + "  " + helpItems[2],
	)

	shortcuts := []string{
		KeyStyle.Render("i") + HelpStyle.Render(" install"),
		KeyStyle.Render("s") + HelpStyle.Render(" search"),
		KeyStyle.Render("c") + HelpStyle.Render(" config"),
		KeyStyle.Render("a") + HelpStyle.Render(" about"),
	}
	shortcutText := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted)).Render(
		"  " + shortcuts[0] + "  " + shortcuts[1] + "  " + shortcuts[2] + "  " + shortcuts[3],
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		logo,
		"",
		menu,
		"",
		helpText,
		shortcutText,
	)
}

type menuItemDelegate struct{}

func (d menuItemDelegate) Height() int                             { return 2 }
func (d menuItemDelegate) Spacing() int                            { return 0 }
func (d menuItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d menuItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(MenuItem)
	if !ok {
		return
	}

	selected := index == m.Index()

	var title, desc string
	if selected {
		title = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimaryBright)).
			Bold(true).
			Render(fmt.Sprintf("  → %s  %s", item.Icon, item.Title))
		desc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted)).
			Render("     " + item.Description)
	} else {
		title = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			Render(fmt.Sprintf("    %s  %s", item.Icon, item.Title))
		desc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted)).
			Render("     " + item.Description)
	}

	fmt.Fprint(w, lipgloss.JoinVertical(lipgloss.Left, title, desc))
}

func RunMainMenu() {
	p := tea.NewProgram(NewMainMenu(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
}

func NewSpinner(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary))
	return SpinnerModel{spinner: s, message: message}
}

func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m SpinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.spinner.View(),
		" ",
		m.message,
	)
}

func ShowSpinner(message string, action func()) {
	p := tea.NewProgram(NewSpinner(message))
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Action panicked: %v\n", r)
			}
			p.Quit()
		}()
		action()
	}()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
