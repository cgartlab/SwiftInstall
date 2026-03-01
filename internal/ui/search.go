package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swiftinstall/internal/config"
	"swiftinstall/internal/i18n"
	"swiftinstall/internal/installer"
)

type SearchModel struct {
	input       textinput.Model
	results     []installer.PackageInfo
	table       table.Model
	query       string
	searching   bool
	quitting    bool
	width       int
	height      int
	selected    []installer.PackageInfo
	message     string
	messageType string
	showDetail  bool
	detailIndex int
	mode        string
}

func NewSearchModel(initialQuery string) SearchModel {
	ti := textinput.New()
	ti.Placeholder = i18n.T("search_placeholder")
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50
	ti.SetValue(initialQuery)

	columns := []table.Column{
		{Title: "Name", Width: 26},
		{Title: "ID", Width: 34},
		{Title: "Version", Width: 10},
		{Title: "Source", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(12),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(lipgloss.Color(ColorPrimaryBright)).
		Bold(true).
		Padding(0, 1)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ColorPrimaryBright)).
		Background(lipgloss.Color(ColorSecondary))
	t.SetStyles(s)

	return SearchModel{
		input:    ti,
		query:    initialQuery,
		table:    t,
		results:  []installer.PackageInfo{},
		selected: []installer.PackageInfo{},
		mode:     "input",
	}
}

func (m SearchModel) Init() tea.Cmd {
	if m.query != "" {
		return m.search(m.query)
	}
	return textinput.Blink
}

func (m SearchModel) search(query string) tea.Cmd {
	return func() tea.Msg {
		inst := installer.NewInstaller()
		if inst == nil {
			return searchResultMsg{err: fmt.Errorf("unsupported platform")}
		}

		results, err := inst.Search(query)
		return searchResultMsg{results: results, err: err}
	}
}

type searchResultMsg struct {
	results []installer.PackageInfo
	err     error
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if m.mode == "input" {
				m.query = m.input.Value()
				if m.query != "" {
					m.searching = true
					m.message = ""
					m.mode = "results"
					return m, m.search(m.query)
				}
			} else if m.mode == "results" && len(m.results) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.results) {
					pkg := m.results[selectedRow]
					cfg := config.Get()
					cfg.AddSoftware(config.Software{
						Name:     pkg.Name,
						ID:       pkg.ID,
						Category: "Other",
					})
					if err := config.Save(); err != nil {
						m.message = fmt.Sprintf("Error: %v", err)
						m.messageType = "error"
						return m, nil
					}
					m.selected = append(m.selected, pkg)
					m.message = fmt.Sprintf("Added: %s", pkg.Name)
					m.messageType = "success"
				}
			}
		case "esc":
			if m.showDetail {
				m.showDetail = false
				return m, nil
			}
			if m.mode == "results" {
				m.mode = "input"
				m.input.Focus()
				m.results = []installer.PackageInfo{}
				m.table.SetRows([]table.Row{})
				return m, textinput.Blink
			}
		case "/":
			if m.mode == "results" {
				m.mode = "input"
				m.input.Focus()
				m.input.SetValue("")
				return m, textinput.Blink
			}
		case "d":
			if m.mode == "results" && len(m.results) > 0 {
				m.showDetail = !m.showDetail
				m.detailIndex = m.table.Cursor()
				return m, nil
			}
		case "i":
			if m.mode == "results" && len(m.results) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.results) {
					pkg := m.results[selectedRow]
					return m, m.installPackage(pkg)
				}
			}
		}

	case searchResultMsg:
		m.searching = false
		if msg.err != nil {
			m.results = []installer.PackageInfo{}
			m.message = fmt.Sprintf("Error: %v", msg.err)
			m.messageType = "error"
		} else {
			m.message = ""
			m.messageType = ""
			m.results = msg.results
			var rows []table.Row
			for _, pkg := range m.results {
				id := pkg.ID
				if id == "" {
					id = pkg.Name
				}
				source := "winget"
				if pkg.Publisher != "" {
					source = pkg.Publisher
				}
				if len(source) > 8 {
					source = source[:8]
				}
				rows = append(rows, table.Row{
					truncate(pkg.Name, 24),
					truncate(id, 32),
					truncate(pkg.Version, 8),
					source,
				})
			}
			m.table.SetRows(rows)
			if len(rows) > 0 {
				m.table.SetCursor(0)
			}
		}
		return m, nil
	}

	if m.mode == "input" {
		m.input, cmd = m.input.Update(msg)
	} else if m.mode == "results" && !m.showDetail {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

func (m SearchModel) installPackage(pkg installer.PackageInfo) tea.Cmd {
	return func() tea.Msg {
		inst := installer.NewInstaller()
		if inst == nil {
			return installPkgResultMsg{err: fmt.Errorf("unsupported platform")}
		}
		result, err := inst.Install(pkg.ID)
		return installPkgResultMsg{result: result, err: err}
	}
}

type installPkgResultMsg struct {
	result *installer.InstallResult
	err    error
}

func (m SearchModel) View() string {
	if m.quitting {
		return "\n  " + i18n.T("common_cancel") + "\n"
	}

	var b strings.Builder

	b.WriteString(TitleStyle.Render(i18n.T("search_title")))
	b.WriteString("\n\n")

	if m.mode == "input" || m.searching {
		b.WriteString(HelpStyle.Render("Search query: "))
		b.WriteString(m.input.View())
		b.WriteString("\n\n")

		if m.searching {
			b.WriteString(HighlightStyle.Render("◉ Searching..."))
		} else {
			b.WriteString(HelpStyle.Render("Type software name and press Enter to search"))
		}
	} else if m.mode == "results" {
		b.WriteString(HelpStyle.Render("Query: "))
		b.WriteString(HighlightStyle.Render(m.query))
		b.WriteString("\n\n")

		if m.showDetail && m.detailIndex < len(m.results) {
			pkg := m.results[m.detailIndex]
			b.WriteString(m.renderDetail(pkg))
		} else {
			if len(m.results) > 0 {
				b.WriteString(InfoStyle.Render(fmt.Sprintf("Found %d results", len(m.results))))
				b.WriteString("\n")
				b.WriteString(m.table.View())
				b.WriteString("\n")
				b.WriteString(HelpStyle.Render("↑/↓ navigate | Enter add to config | i install now | d detail | / new search | Esc back | q quit"))
			} else {
				b.WriteString(WarningStyle.Render("No results found for: " + m.query))
				b.WriteString("\n")
				b.WriteString(HelpStyle.Render("Press / to search again or Esc to go back"))
			}
		}
	}

	if m.message != "" {
		b.WriteString("\n")
		if m.messageType == "error" {
			b.WriteString(ErrorStyle.Render("✗ " + m.message))
		} else {
			b.WriteString(SuccessStyle.Render("✓ " + m.message))
		}
	}

	if len(m.selected) > 0 {
		b.WriteString("\n")
		b.WriteString(InfoStyle.Render(fmt.Sprintf("Added to config: %d package(s)", len(m.selected))))
	}

	return b.String()
}

func (m SearchModel) renderDetail(pkg installer.PackageInfo) string {
	var b strings.Builder

	b.WriteString(BoxStyle.Render(func() string {
		var inner strings.Builder
		inner.WriteString(HighlightStyle.Render("Package Details"))
		inner.WriteString("\n\n")
		inner.WriteString(fmt.Sprintf("  %-12s %s\n", "Name:", pkg.Name))
		inner.WriteString(fmt.Sprintf("  %-12s %s\n", "ID:", pkg.ID))
		if pkg.Version != "" {
			inner.WriteString(fmt.Sprintf("  %-12s %s\n", "Version:", pkg.Version))
		}
		if pkg.Publisher != "" {
			inner.WriteString(fmt.Sprintf("  %-12s %s\n", "Publisher:", pkg.Publisher))
		}
		if pkg.Description != "" {
			inner.WriteString(fmt.Sprintf("  %-12s %s\n", "Description:", truncate(pkg.Description, 50)))
		}
		inner.WriteString("\n")
		inner.WriteString(HelpStyle.Render("Press d to close | Enter add | i install | Esc back"))
		return inner.String()
	}()))

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func RunSearch(query string) {
	p := tea.NewProgram(NewSearchModel(query), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func ShowPackageList(packages []config.Software) {
	if len(packages) == 0 {
		fmt.Println(WarningStyle.Render(i18n.T("warn_no_packages")))
		return
	}

	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Name", Width: 24},
		{Title: "ID", Width: 32},
		{Title: "Category", Width: 16},
	}

	var rows []table.Row
	for i, pkg := range packages {
		id := pkg.ID
		if id == "" {
			id = pkg.Package
		}
		category := pkg.Category
		if category == "" {
			category = "Other"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			pkg.Name,
			id,
			category,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(packages)+2),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(lipgloss.Color(ColorPrimaryBright)).
		Bold(true).
		Padding(0, 1)
	t.SetStyles(s)

	fmt.Println(TitleStyle.Render(i18n.T("cmd_list_short")))
	fmt.Println()
	fmt.Println(t.View())
	fmt.Println()
	fmt.Println(HelpStyle.Render(fmt.Sprintf("Total: %d packages", len(packages))))
}
