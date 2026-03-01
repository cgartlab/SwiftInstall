package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"swiftinstall/internal/appinfo"
	"swiftinstall/internal/config"
	"swiftinstall/internal/i18n"
	"swiftinstall/internal/installer"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ConfigModel struct {
	mode        string
	table       table.Model
	inputs      []textinput.Model
	focusIndex  int
	packages    []config.Software
	selected    int
	quitting    bool
	width       int
	height      int
	message     string
	messageType string
}

func NewConfigModel() ConfigModel {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Name", Width: 24},
		{Title: "ID", Width: 32},
		{Title: "Category", Width: 16},
	}

	cfg := config.Get()
	packages := cfg.GetSoftwareList()

	var rows []table.Row
	for i, pkg := range packages {
		id := pkg.ID
		if id == "" {
			id = pkg.Package
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			pkg.Name,
			id,
			pkg.Category,
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

	inputs := make([]textinput.Model, 3)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Name"
	inputs[0].Focus()
	inputs[0].CharLimit = 50
	inputs[0].Width = 32

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Package ID"
	inputs[1].CharLimit = 50
	inputs[1].Width = 32

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Category"
	inputs[2].CharLimit = 30
	inputs[2].Width = 32
	inputs[2].SetValue("Other")

	return ConfigModel{
		mode:     "list",
		table:    t,
		inputs:   inputs,
		packages: packages,
	}
}

func (m ConfigModel) Init() tea.Cmd {
	return nil
}

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.mode == "list" {
				m.quitting = true
				return m, tea.Quit
			} else {
				m.mode = "list"
				m.message = ""
				return m, nil
			}
		case "a":
			if m.mode == "list" {
				m.mode = "add"
				m.focusIndex = 0
				for i := range m.inputs {
					m.inputs[i].SetValue("")
					m.inputs[i].Blur()
				}
				m.inputs[0].Focus()
				return m, textinput.Blink
			}
		case "e":
			if m.mode == "list" && len(m.packages) > 0 {
				m.mode = "edit"
				m.selected = m.table.Cursor()
				if m.selected < len(m.packages) {
					pkg := m.packages[m.selected]
					m.inputs[0].SetValue(pkg.Name)
					id := pkg.ID
					if id == "" {
						id = pkg.Package
					}
					m.inputs[1].SetValue(id)
					m.inputs[2].SetValue(pkg.Category)
				}
				m.focusIndex = 0
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				m.inputs[0].Focus()
				return m, textinput.Blink
			}
		case "d", "r":
			if m.mode == "list" && len(m.packages) > 0 {
				m.mode = "remove"
				m.selected = m.table.Cursor()
				return m, nil
			}
		case "enter":
			if m.mode == "list" && len(m.packages) > 0 {
				m.mode = "edit"
				m.selected = m.table.Cursor()
				if m.selected < len(m.packages) {
					pkg := m.packages[m.selected]
					m.inputs[0].SetValue(pkg.Name)
					id := pkg.ID
					if id == "" {
						id = pkg.Package
					}
					m.inputs[1].SetValue(id)
					m.inputs[2].SetValue(pkg.Category)
				}
				m.focusIndex = 0
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				m.inputs[0].Focus()
				return m, textinput.Blink
			}

			switch m.mode {
			case "add":
				if m.focusIndex < len(m.inputs)-1 {
					m.focusIndex++
					for i := range m.inputs {
						if i == m.focusIndex {
							m.inputs[i].Focus()
						} else {
							m.inputs[i].Blur()
						}
					}
					return m, textinput.Blink
				}
				m.savePackage()
				return m, nil
			case "edit":
				if m.focusIndex < len(m.inputs)-1 {
					m.focusIndex++
					for i := range m.inputs {
						if i == m.focusIndex {
							m.inputs[i].Focus()
						} else {
							m.inputs[i].Blur()
						}
					}
					return m, textinput.Blink
				}
				m.updatePackage()
				return m, nil
			case "remove":
				m.deletePackage()
				return m, nil
			}
		case "y":
			if m.mode == "remove" {
				m.deletePackage()
				return m, nil
			}
		case "n":
			if m.mode == "remove" {
				m.mode = "list"
				m.message = ""
				return m, nil
			}
		case "tab", "shift+tab":
			if m.mode == "add" || m.mode == "edit" {
				if msg.String() == "tab" {
					m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
				} else {
					m.focusIndex--
					if m.focusIndex < 0 {
						m.focusIndex = len(m.inputs) - 1
					}
				}
				for i := range m.inputs {
					if i == m.focusIndex {
						m.inputs[i].Focus()
					} else {
						m.inputs[i].Blur()
					}
				}
				return m, textinput.Blink
			}
		}
	}

	if m.mode == "list" {
		m.table, cmd = m.table.Update(msg)
	} else if m.mode == "add" || m.mode == "edit" {
		for i := range m.inputs {
			m.inputs[i], cmd = m.inputs[i].Update(msg)
		}
	}

	return m, cmd
}

func (m *ConfigModel) savePackage() {
	name := m.inputs[0].Value()
	id := m.inputs[1].Value()
	category := m.inputs[2].Value()

	if name == "" || id == "" {
		m.message = "Name and ID are required"
		m.messageType = "error"
		return
	}

	cfg := config.Get()
	cfg.AddSoftware(config.Software{
		Name:     name,
		ID:       id,
		Category: category,
	})
	if err := config.Save(); err != nil {
		log.Printf("Error: failed to save config after adding package: %v", err)
		m.message = fmt.Sprintf("Error: %v", err)
		m.messageType = "error"
		return
	}

	m.packages = cfg.GetSoftwareList()
	m.refreshTable()

	m.mode = "list"
	m.message = fmt.Sprintf("Added: %s", name)
	m.messageType = "success"
}

func (m *ConfigModel) updatePackage() {
	if m.selected < 0 || m.selected >= len(m.packages) {
		m.message = "Invalid selection"
		m.messageType = "error"
		m.mode = "list"
		return
	}

	name := m.inputs[0].Value()
	id := m.inputs[1].Value()
	category := m.inputs[2].Value()

	if name == "" || id == "" {
		m.message = "Name and ID are required"
		m.messageType = "error"
		return
	}

	cfg := config.Get()
	cfg.UpdateSoftware(m.selected, config.Software{
		Name:     name,
		ID:       id,
		Category: category,
	})
	if err := config.Save(); err != nil {
		log.Printf("Error: failed to save config after updating package: %v", err)
		m.message = fmt.Sprintf("Error: %v", err)
		m.messageType = "error"
		return
	}

	m.packages = cfg.GetSoftwareList()
	m.refreshTable()

	m.mode = "list"
	m.message = fmt.Sprintf("Updated: %s", name)
	m.messageType = "success"
}

func (m *ConfigModel) deletePackage() {
	if m.selected >= len(m.packages) {
		m.mode = "list"
		return
	}

	name := m.packages[m.selected].Name
	cfg := config.Get()
	cfg.RemoveSoftware(m.selected)
	if err := config.Save(); err != nil {
		log.Printf("Error: failed to save config after removing package: %v", err)
		m.message = fmt.Sprintf("Error: %v", err)
		m.messageType = "error"
		return
	}

	m.packages = cfg.GetSoftwareList()
	m.refreshTable()

	m.mode = "list"
	m.message = fmt.Sprintf("Removed: %s", name)
	m.messageType = "success"
}

func (m *ConfigModel) refreshTable() {
	var rows []table.Row
	for i, pkg := range m.packages {
		id := pkg.ID
		if id == "" {
			id = pkg.Package
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			pkg.Name,
			id,
			pkg.Category,
		})
	}
	m.table.SetRows(rows)
	if len(rows) == 0 {
		m.table.SetCursor(0)
		return
	}
	if m.table.Cursor() >= len(rows) {
		m.table.SetCursor(len(rows) - 1)
	}
}

func (m ConfigModel) View() string {
	if m.quitting {
		return "\n  " + i18n.T("common_cancel") + "\n"
	}

	var b strings.Builder

	b.WriteString(TitleStyle.Render(i18n.T("config_title")))
	b.WriteString("\n\n")

	switch m.mode {
	case "list":
		if len(m.packages) > 0 {
			b.WriteString(m.table.View())
			b.WriteString("\n")
		} else {
			b.WriteString(WarningStyle.Render("No packages configured"))
			b.WriteString("\n")
		}

		if m.message != "" {
			b.WriteString("\n")
			if m.messageType == "success" {
				b.WriteString(SuccessStyle.Render("✓ " + m.message))
			} else if m.messageType == "error" {
				b.WriteString(ErrorStyle.Render("✗ " + m.message))
			} else {
				b.WriteString(InfoStyle.Render(m.message))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("a add | Enter/e edit | d remove | q quit"))

	case "add":
		b.WriteString(HighlightStyle.Render("Add Package"))
		b.WriteString("\n\n")

		labels := []string{"Name:", "ID:", "Category:"}
		for i, input := range m.inputs {
			if i == m.focusIndex {
				b.WriteString(KeyStyle.Render("→ " + labels[i]))
			} else {
				b.WriteString("  " + labels[i])
			}
			b.WriteString(" ")
			b.WriteString(input.View())
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Tab next | Enter save | q cancel"))

	case "edit":
		b.WriteString(HighlightStyle.Render("Edit Package"))
		b.WriteString("\n\n")

		labels := []string{"Name:", "ID:", "Category:"}
		for i, input := range m.inputs {
			if i == m.focusIndex {
				b.WriteString(KeyStyle.Render("→ " + labels[i]))
			} else {
				b.WriteString("  " + labels[i])
			}
			b.WriteString(" ")
			b.WriteString(input.View())
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Tab next | Enter save | q cancel"))

	case "remove":
		if m.selected < len(m.packages) {
			pkg := m.packages[m.selected]
			b.WriteString(WarningStyle.Render("Delete this package?"))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("  Name: %s\n", pkg.Name))
			id := pkg.ID
			if id == "" {
				id = pkg.Package
			}
			b.WriteString(fmt.Sprintf("  ID:   %s\n", id))
			b.WriteString("\n")
			b.WriteString(HelpStyle.Render("y confirm | n cancel"))
		}
	}

	b.WriteString("\n")
	return b.String()
}

func RunConfigManager() {
	p := tea.NewProgram(NewConfigModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

type WizardModel struct {
	step       int
	language   string
	categories []string
	selected   map[string]bool
	quitting   bool
	done       bool
	message    string
}

var wizardCategories = map[string][]config.Software{
	"Development": {
		{Name: "Git", ID: "Git.Git", Category: "Development"},
		{Name: "Visual Studio Code", ID: "Microsoft.VisualStudioCode", Category: "Development"},
		{Name: "Node.js", ID: "OpenJSFoundation.NodeJS.LTS", Category: "Development"},
		{Name: "Python", ID: "Python.Python.3.12", Category: "Development"},
	},
	"Browsers": {
		{Name: "Google Chrome", ID: "Google.Chrome", Category: "Browsers"},
		{Name: "Mozilla Firefox", ID: "Mozilla.Firefox", Category: "Browsers"},
		{Name: "Microsoft Edge", ID: "Microsoft.Edge", Category: "Browsers"},
	},
	"Utilities": {
		{Name: "7-Zip", ID: "7zip.7zip", Category: "Utilities"},
		{Name: "Notepad++", ID: "Notepad++.Notepad++", Category: "Utilities"},
		{Name: "Everything", ID: "voidtools.Everything", Category: "Utilities"},
	},
	"Media": {
		{Name: "VLC", ID: "VideoLAN.VLC", Category: "Media"},
		{Name: "Spotify", ID: "Spotify.Spotify", Category: "Media"},
	},
}

func NewWizardModel() WizardModel {
	categories := make([]string, 0, len(wizardCategories))
	for cat := range wizardCategories {
		categories = append(categories, cat)
	}
	return WizardModel{
		step:       0,
		language:   "zh",
		categories: categories,
		selected:   make(map[string]bool),
	}
}

func (m WizardModel) Init() tea.Cmd {
	return nil
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if m.step == 0 {
				m.step = 1
				return m, nil
			} else if m.step == 1 {
				m.step = 2
				return m, nil
			} else if m.step == 2 {
				m.saveSelections()
				m.done = true
				return m, tea.Quit
			}
		case "1":
			if m.step == 0 {
				m.language = "en"
				i18n.SetLanguage("en")
			}
		case "2":
			if m.step == 0 {
				m.language = "zh"
				i18n.SetLanguage("zh")
			}
		case " ":
			if m.step == 1 {
				for i, cat := range m.categories {
					if msg.String() == " " && i < len(m.categories) {
						m.selected[cat] = !m.selected[cat]
						break
					}
				}
			}
		}
	}

	return m, nil
}

func (m *WizardModel) saveSelections() {
	cfg := config.Get()
	cfg.ClearSoftware()

	for cat, selected := range m.selected {
		if selected {
			for _, sw := range wizardCategories[cat] {
				cfg.AddSoftware(sw)
			}
		}
	}

	if err := config.Save(); err != nil {
		m.message = fmt.Sprintf("Error saving: %v", err)
	}
}

func (m WizardModel) View() string {
	if m.quitting {
		return "\n  " + i18n.T("common_cancel") + "\n"
	}

	var b strings.Builder

	b.WriteString(GetCompactLogo())
	b.WriteString("\n\n")

	switch m.step {
	case 0:
		b.WriteString(TitleStyle.Render(i18n.T("wizard_welcome")))
		b.WriteString("\n\n")
		b.WriteString(i18n.T("wizard_desc"))
		b.WriteString("\n\n")

		b.WriteString(InfoStyle.Render("Select language / 选择语言:"))
		b.WriteString("\n\n")

		langZh := "  2. 中文"
		langEn := "  1. English"
		if m.language == "zh" {
			langZh = KeyStyle.Render("→ 2. 中文")
		} else {
			langEn = KeyStyle.Render("→ 1. English")
		}
		b.WriteString(langEn + "\n")
		b.WriteString(langZh + "\n")
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("1/2 select | Enter next | q quit"))

	case 1:
		b.WriteString(TitleStyle.Render(i18n.T("wizard_welcome")))
		b.WriteString(" - ")
		b.WriteString(HighlightStyle.Render(i18n.T("wizard_step") + " 1/2"))
		b.WriteString("\n\n")
		b.WriteString(i18n.T("wizard_select_categories"))
		b.WriteString("\n\n")

		for _, cat := range m.categories {
			prefix := "○"
			if m.selected[cat] {
				prefix = SuccessStyle.Render("●")
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", prefix, cat))
		}
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Space select | Enter next | q quit"))

	case 2:
		b.WriteString(TitleStyle.Render(i18n.T("wizard_welcome")))
		b.WriteString(" - ")
		b.WriteString(HighlightStyle.Render(i18n.T("wizard_step") + " 2/2"))
		b.WriteString("\n\n")

		b.WriteString(InfoStyle.Render(i18n.T("wizard_confirm")))
		b.WriteString("\n\n")

		selectedCount := 0
		for cat, selected := range m.selected {
			if selected {
				selectedCount += len(wizardCategories[cat])
				b.WriteString(fmt.Sprintf("  • %s (%d packages)\n", cat, len(wizardCategories[cat])))
			}
		}

		if selectedCount == 0 {
			b.WriteString(WarningStyle.Render("  No packages selected"))
		} else {
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  %s: %d", i18n.T("install_total"), selectedCount))
		}

		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Enter confirm | q quit"))
	}

	return b.String()
}

func RunWizard() {
	p := tea.NewProgram(NewWizardModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func RunBatch(packages []config.Software, parallel bool) {
	RunInstall(packages, parallel)
}

func RunBatchFromFile(file string) {
	cfg := config.Get()
	err := cfg.ImportFromFile(file)
	if err != nil {
		fmt.Println(ErrorStyle.Render(fmt.Sprintf("Failed to load file: %v", err)))
		return
	}

	packages := cfg.GetSoftwareList()
	RunInstall(packages, true)
}

func RunExport(packages []config.Software, format, output string) {
	if len(packages) == 0 {
		fmt.Println(WarningStyle.Render(i18n.T("warn_no_packages")))
		return
	}

	var content string
	var err error

	switch format {
	case "json":
		content, err = exportToJSON(packages)
	case "yaml", "yml":
		content, err = exportToYAML(packages)
	case "powershell", "ps1":
		content = exportToPowerShell(packages)
	case "bash", "sh":
		content = exportToBash(packages)
	default:
		fmt.Println(ErrorStyle.Render(fmt.Sprintf("Unsupported format: %s", format)))
		return
	}

	if err != nil {
		fmt.Println(ErrorStyle.Render(fmt.Sprintf("Export failed: %v", err)))
		return
	}

	if output != "" {
		err = os.WriteFile(output, []byte(content), 0644)
		if err != nil {
			fmt.Println(ErrorStyle.Render(fmt.Sprintf("Failed to write file: %v", err)))
			return
		}
		fmt.Println(SuccessStyle.Render(fmt.Sprintf("✓ Exported to: %s", output)))
	} else {
		fmt.Println(InfoStyle.Render(fmt.Sprintf("Export format: %s", format)))
		fmt.Println()
		fmt.Println(content)
	}
}

func exportToJSON(packages []config.Software) (string, error) {
	data, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func exportToYAML(packages []config.Software) (string, error) {
	data, err := yaml.Marshal(packages)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func exportToPowerShell(packages []config.Software) string {
	var b strings.Builder
	b.WriteString("# SwiftInstall PowerShell Installation Script\n")
	b.WriteString("# Generated by SwiftInstall\n\n")
	b.WriteString("$packages = @(\n")
	for _, pkg := range packages {
		id := pkg.ID
		if id == "" {
			id = pkg.Package
		}
		b.WriteString(fmt.Sprintf("    \"%s\",\n", id))
	}
	b.WriteString(")\n\n")
	b.WriteString("foreach ($package in $packages) {\n")
	b.WriteString("    Write-Host \"Installing $package...\" -ForegroundColor Cyan\n")
	b.WriteString("    winget install --id $package --silent --accept-package-agreements --accept-source-agreements\n")
	b.WriteString("}\n\n")
	b.WriteString("Write-Host \"Installation complete!\" -ForegroundColor Green\n")
	return b.String()
}

func exportToBash(packages []config.Software) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("# SwiftInstall Bash Installation Script\n")
	b.WriteString("# Generated by SwiftInstall\n\n")
	b.WriteString("packages=(\n")
	for _, pkg := range packages {
		id := pkg.ID
		if id == "" {
			id = pkg.Package
		}
		b.WriteString(fmt.Sprintf("    \"%s\"\n", id))
	}
	b.WriteString(")\n\n")
	b.WriteString("for package in \"${packages[@]}\"; do\n")
	b.WriteString("    echo \"Installing $package...\"\n")
	b.WriteString("    brew install \"$package\"\n")
	b.WriteString("done\n\n")
	b.WriteString("echo \"Installation complete!\"\n")
	return b.String()
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

func RunUpdateCheck() {
	fmt.Println(TitleStyle.Render(i18n.T("cmd_update_short")))
	fmt.Println()

	currentVersion := appinfo.Version
	if currentVersion == "" || currentVersion == "dev" {
		currentVersion = "v0.1.3"
	}

	fmt.Println(InfoStyle.Render(fmt.Sprintf("%s %s", i18n.T("update_current"), currentVersion)))
	fmt.Println()
	fmt.Println(InfoStyle.Render(i18n.T("update_checking")))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/cgartlab/SwiftInstall/releases/latest")
	if err != nil {
		fmt.Println()
		fmt.Println(WarningStyle.Render(fmt.Sprintf("%s: %v", i18n.T("update_failed"), err)))
		fmt.Println(InfoStyle.Render(i18n.T("update_manual")))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println()
		fmt.Println(WarningStyle.Render(fmt.Sprintf("%s (HTTP %d)", i18n.T("update_failed"), resp.StatusCode)))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println()
		fmt.Println(WarningStyle.Render(fmt.Sprintf("%s: %v", i18n.T("update_failed"), err)))
		return
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		fmt.Println()
		fmt.Println(WarningStyle.Render(fmt.Sprintf("%s: %v", i18n.T("update_parse_failed"), err)))
		return
	}

	latestVersion := release.TagName
	fmt.Println()
	fmt.Println(InfoStyle.Render(fmt.Sprintf("%s %s", i18n.T("update_latest"), latestVersion)))
	fmt.Println()

	if latestVersion == currentVersion {
		fmt.Println(SuccessStyle.Render("✓ " + i18n.T("update_uptodate")))
	} else {
		fmt.Println(HighlightStyle.Render("→ " + i18n.T("update_available")))
		fmt.Println()
		fmt.Println(InfoStyle.Render(fmt.Sprintf("%s: %s", i18n.T("update_download"), release.HTMLURL)))
		fmt.Println()
		fmt.Println(HelpStyle.Render(i18n.T("update_hint")))
	}
}

func RunClean() {
	fmt.Println(TitleStyle.Render(i18n.T("cmd_clean_short")))
	fmt.Println()

	var cleaned []string
	var errors []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
		return
	}

	cacheDirs := []struct {
		name string
		path string
	}{
		{"Winget Cache", filepath.Join(os.Getenv("LOCALAPPDATA"), "Temp", "WinGet")},
		{"Winget Packages", filepath.Join(os.Getenv("LOCALAPPDATA"), "Packages", "Microsoft.DesktopAppInstaller_8wekyb3d8bbwe", "TempState")},
		{"Temp Folder", filepath.Join(os.Getenv("TEMP"))},
		{"SI Cache", filepath.Join(homeDir, ".si", "cache")},
	}

	if runtime.GOOS == "darwin" {
		cacheDirs = []struct {
			name string
			path string
		}{
			{"Homebrew Cache", filepath.Join(homeDir, "Library", "Caches", "Homebrew")},
			{"Temp Folder", "/tmp"},
			{"SI Cache", filepath.Join(homeDir, ".si", "cache")},
		}
	} else if runtime.GOOS == "linux" {
		cacheDirs = []struct {
			name string
			path string
		}{
			{"APT Cache", "/var/cache/apt/archives"},
			{"Temp Folder", "/tmp"},
			{"SI Cache", filepath.Join(homeDir, ".si", "cache")},
		}
	}

	fmt.Println(InfoStyle.Render(i18n.T("clean_scanning")))
	fmt.Println()

	for _, dir := range cacheDirs {
		if dir.path == "" {
			continue
		}

		info, err := os.Stat(dir.path)
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", dir.name, err))
			continue
		}

		if !info.IsDir() {
			continue
		}

		size, fileCount, err := getDirSize(dir.path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", dir.name, err))
			continue
		}

		if fileCount > 0 {
			fmt.Printf("  %s: %s (%d files)\n", dir.name, formatSize(size), fileCount)
			cleaned = append(cleaned, dir.name)
		}
	}

	if len(cleaned) == 0 {
		fmt.Println(SuccessStyle.Render("✓ " + i18n.T("clean_no_cache")))
		return
	}

	fmt.Println()
	fmt.Print(InfoStyle.Render(i18n.T("clean_confirm")))

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" && response != i18n.T("common_yes") {
		fmt.Println(WarningStyle.Render(i18n.T("clean_cancelled")))
		return
	}

	fmt.Println()
	fmt.Println(InfoStyle.Render(i18n.T("clean_cleaning")))

	for _, dir := range cacheDirs {
		if dir.path == "" {
			continue
		}
		if _, err := os.Stat(dir.path); os.IsNotExist(err) {
			continue
		}
		if err := clearDirContents(dir.path); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", dir.name, err))
		}
	}

	fmt.Println()
	if len(errors) > 0 {
		fmt.Println(WarningStyle.Render(i18n.T("clean_partial")))
		for _, e := range errors {
			fmt.Printf("  • %s\n", e)
		}
	} else {
		fmt.Println(SuccessStyle.Render("✓ " + i18n.T("clean_done")))
	}
}

func getDirSize(path string) (int64, int, error) {
	var size int64
	var count int
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
			count++
		}
		return nil
	})
	return size, count, err
}

func clearDirContents(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			log.Printf("Warning: failed to remove %s: %v", entry.Name(), err)
		}
	}
	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func RunStatus() {
	fmt.Println(TitleStyle.Render(i18n.T("cmd_status_short")))
	fmt.Println()

	pm, available := installer.CheckPackageManager()

	fmt.Println(SectionStyle.Render(i18n.T("status_platform")))
	fmt.Printf("  OS:   %s\n", getOSName())
	fmt.Printf("  Arch: %s\n", getArch())
	fmt.Println()

	fmt.Println(SectionStyle.Render(i18n.T("status_package_mgr")))
	if available {
		fmt.Printf("  %s: %s\n", pm, SuccessStyle.Render("✓ "+i18n.T("status_available")))

		var pmVersion string
		switch runtime.GOOS {
		case "windows":
			cmd := exec.Command("winget", "--version")
			output, err := cmd.Output()
			if err == nil {
				pmVersion = strings.TrimSpace(string(output))
			}
		case "darwin":
			cmd := exec.Command("brew", "--version")
			output, err := cmd.Output()
			if err == nil {
				lines := strings.Split(string(output), "\n")
				if len(lines) > 0 {
					pmVersion = strings.TrimSpace(lines[0])
				}
			}
		}
		if pmVersion != "" {
			fmt.Printf("  %s: %s\n", i18n.T("status_version"), pmVersion)
		}
	} else {
		fmt.Printf("  %s: %s\n", pm, ErrorStyle.Render("✗ "+i18n.T("status_unavailable")))
		fmt.Println()
		fmt.Println(WarningStyle.Render(i18n.T("status_install_pm")))
		return
	}
	fmt.Println()

	inst := installer.NewInstaller()
	if inst != nil {
		fmt.Println(SectionStyle.Render(i18n.T("status_installed")))
		installed, err := inst.GetInstalled()
		if err != nil {
			fmt.Printf("  %s: %v\n", i18n.T("common_error"), err)
		} else {
			fmt.Printf("  %s: %d\n", i18n.T("status_total"), len(installed))
			if len(installed) > 0 && len(installed) <= 10 {
				fmt.Println()
				for i, pkg := range installed {
					if i >= 10 {
						break
					}
					name := pkg.Name
					if name == "" {
						name = pkg.ID
					}
					fmt.Printf("    • %s", name)
					if pkg.Version != "" {
						fmt.Printf(" (%s)", pkg.Version)
					}
					fmt.Println()
				}
				if len(installed) > 10 {
					fmt.Printf("    ... %s %d %s\n", i18n.T("status_more"), len(installed)-10, i18n.T("status_packages"))
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(SectionStyle.Render(i18n.T("status_config")))
	cfg := config.Get()
	configPath := cfg.GetConfigPath()
	fmt.Printf("  %s: %s\n", i18n.T("status_config_path"), configPath)

	packages := cfg.GetSoftwareList()
	fmt.Printf("  %s: %d\n", i18n.T("status_configured"), len(packages))
}

func getOSName() string {
	return runtime.GOOS
}

func getArch() string {
	return runtime.GOARCH
}
