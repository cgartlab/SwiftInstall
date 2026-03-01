package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swiftinstall/internal/config"
	"swiftinstall/internal/i18n"
	"swiftinstall/internal/installer"
)

type InstallModel struct {
	packages  []config.Software
	results   []*installer.InstallResult
	progress  progress.Model
	table     table.Model
	status    string
	quitting  bool
	done      bool
	parallel  bool
	width     int
	height    int
	mu        sync.Mutex
	showAbout bool
}

type tickMsg struct{}

func NewInstallModel(packages []config.Software, parallel bool) InstallModel {
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	columns := []table.Column{
		{Title: "Package", Width: 24},
		{Title: "ID", Width: 28},
		{Title: "Status", Width: 12},
	}

	var rows []table.Row
	for _, pkg := range packages {
		id := pkg.ID
		if id == "" {
			id = pkg.Package
		}
		rows = append(rows, table.Row{
			pkg.Name,
			id,
			"○",
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(lipgloss.Color(ColorPrimaryBright)).
		Bold(true).
		Padding(0, 1)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ColorPrimaryBright))
	t.SetStyles(s)

	return InstallModel{
		packages: packages,
		results:  make([]*installer.InstallResult, len(packages)),
		progress: p,
		table:    t,
		parallel: parallel,
		status:   i18n.T("install_progress"),
	}
}

func (m *InstallModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.runInstall(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *InstallModel) runInstall() tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup

		if m.parallel {
			semaphore := make(chan struct{}, 4)
			for i := range m.packages {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							m.mu.Lock()
							m.results[index] = &installer.InstallResult{
								Status: installer.StatusFailed,
								Error:  fmt.Errorf("panic during installation: %v", r),
							}
							m.mu.Unlock()
						}
					}()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					m.installPackage(index)
				}(i)
			}
		} else {
			for i := range m.packages {
				m.installPackage(i)
			}
		}

		wg.Wait()
		return installDoneMsg{}
	}
}

func (m *InstallModel) installPackage(index int) {
	inst := installer.NewInstaller()
	if inst == nil {
		m.mu.Lock()
		m.results[index] = &installer.InstallResult{
			Status: installer.StatusFailed,
			Error:  fmt.Errorf("unsupported platform"),
		}
		m.mu.Unlock()
		return
	}

	pkg := m.packages[index]
	packageID := pkg.ID
	if packageID == "" {
		packageID = pkg.Package
	}

	result, err := inst.Install(packageID)
	if err != nil && result == nil {
		result = &installer.InstallResult{
			Package: installer.PackageInfo{ID: packageID},
			Status:  installer.StatusFailed,
			Error:   err,
		}
	}
	if result == nil {
		result = &installer.InstallResult{
			Package: installer.PackageInfo{ID: packageID},
			Status:  installer.StatusFailed,
			Error:   fmt.Errorf("install failed with empty result"),
		}
	}

	m.mu.Lock()
	m.results[index] = result

	status := "○"
	if result.Status == installer.StatusSuccess {
		status = SuccessStyle.Render("✓")
	} else if result.Status == installer.StatusFailed {
		status = ErrorStyle.Render("✗")
	} else if result.Status == installer.StatusSkipped {
		status = WarningStyle.Render("⊘")
	}

	rows := m.table.Rows()
	if index < len(rows) {
		rows[index][2] = status
		m.table.SetRows(rows)
	}
	m.mu.Unlock()
}

type installDoneMsg struct{}

func (m *InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "a":
			m.showAbout = !m.showAbout
			return m, nil
		case "esc":
			if m.showAbout {
				m.showAbout = false
				return m, nil
			}
		case "enter":
			if m.done {
				return m, tea.Quit
			}
		}

	case tickMsg:
		completed := 0
		for _, r := range m.results {
			if r != nil {
				completed++
			}
		}

		if completed < len(m.packages) {
			percent := float64(completed) / float64(len(m.packages))
			m.progress.SetPercent(percent)
			return m, tickCmd()
		}

	case installDoneMsg:
		m.done = true
		m.progress.SetPercent(1.0)
		m.status = i18n.T("common_done")
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *InstallModel) View() string {
	if m.quitting {
		return "\n  " + i18n.T("common_cancel") + "\n"
	}

	var b strings.Builder

	if m.showAbout {
		b.WriteString(GetAboutText())
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render(i18n.T("common_back") + " Esc | " + i18n.T("common_cancel") + " q"))
		return b.String()
	}

	b.WriteString(TitleStyle.Render(i18n.T("install_title")))
	b.WriteString("\n\n")

	b.WriteString(m.progress.View())
	b.WriteString("\n\n")

	if m.done {
		b.WriteString(SuccessStyle.Render("✓ " + m.status))
	} else {
		b.WriteString(HighlightStyle.Render("◉ " + m.status))
	}
	b.WriteString("\n\n")

	b.WriteString(m.table.View())
	b.WriteString("\n")

	if m.done {
		success, failed, skipped := 0, 0, 0
		for _, r := range m.results {
			if r != nil {
				switch r.Status {
				case installer.StatusSuccess:
					success++
				case installer.StatusFailed:
					failed++
				case installer.StatusSkipped:
					skipped++
				}
			}
		}

		b.WriteString("\n")
		b.WriteString(SuccessStyle.Render(fmt.Sprintf("✓ %d", success)))
		if failed > 0 {
			b.WriteString("  ")
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("✗ %d", failed)))
		}
		if skipped > 0 {
			b.WriteString("  ")
			b.WriteString(WarningStyle.Render(fmt.Sprintf("⊘ %d", skipped)))
		}
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Enter confirm | q quit"))
	} else {
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("q quit"))
	}

	return b.String()
}

func RunInstall(packages []config.Software, parallel bool) {
	if len(packages) == 0 {
		fmt.Println(WarningStyle.Render(i18n.T("warn_no_packages")))
		return
	}

	model := NewInstallModel(packages, parallel)
	p := tea.NewProgram(&model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func RunInstallByName(packageNames []string, parallel bool) {
	packages := make([]config.Software, len(packageNames))
	for i, name := range packageNames {
		packages[i] = config.Software{
			Name: name,
			ID:   name,
		}
	}
	RunInstall(packages, parallel)
}

func RunUninstall(packages []config.Software) {
	inst := installer.NewInstaller()
	if inst == nil {
		fmt.Println(ErrorStyle.Render("Unsupported platform"))
		return
	}

	fmt.Println(TitleStyle.Render(i18n.T("menu_uninstall")))
	fmt.Println()

	for _, pkg := range packages {
		packageID := pkg.ID
		if packageID == "" {
			packageID = pkg.Package
		}

		fmt.Printf("  %s... ", pkg.Name)
		result, err := inst.Uninstall(packageID)
		if err != nil || result.Status == installer.StatusFailed {
			fmt.Println(ErrorStyle.Render("✗"))
		} else if result.Status == installer.StatusSkipped {
			fmt.Println(WarningStyle.Render("⊘"))
		} else {
			fmt.Println(SuccessStyle.Render("✓"))
		}
	}
}

func RunUninstallByName(packageNames []string) {
	packages := make([]config.Software, len(packageNames))
	for i, name := range packageNames {
		packages[i] = config.Software{
			Name: name,
			ID:   name,
		}
	}
	RunUninstall(packages)
}
