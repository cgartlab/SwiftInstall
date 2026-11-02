package tui

import (
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampOffset()
		m.ensureVisible()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.busy() {
			return m, cmd
		}
		return m, nil

	case probeMsg:
		return m.handleProbe(msg)

	case probeDoneMsg:
		m.loading = false
		return m, nil

	case progressMsg:
		return m.handleProgress(msg)

	case runFinishedMsg:
		return m.handleRunFinished(runOutcome(msg))
	}

	return m, nil
}

// handleProbe applies one probe result and waits for the next.
func (m *model) handleProbe(msg probeMsg) (tea.Model, tea.Cmd) {
	if msg.index >= 0 && msg.index < len(m.packages) {
		p := m.packages[msg.index]
		p.known = true
		p.installed = msg.installed
		m.probed++
	}
	return m, m.waitProbe()
}

// handleProgress records one batch result and keeps the stream flowing.
func (m *model) handleProgress(msg progressMsg) (tea.Model, tea.Cmd) {
	m.runResults = append(m.runResults, msg.result)
	m.runDone = len(m.runResults)
	m.applyResultToStatus(msg.result)
	return m, m.waitProgress()
}

// applyResultToStatus keeps the checkbox-side install glyphs consistent with
// what the batch just did.
func (m *model) applyResultToStatus(r engine.Result) {
	for _, p := range m.packages {
		if p.pkg.ID != r.Package.ID {
			continue
		}
		p.known = true
		switch r.Status {
		case engine.StatusSuccess:
			p.installed = m.action != "uninstall"
		case engine.StatusSkipped, engine.StatusOptionalFailed:
			// A skipped install means it was already present; a skipped
			// uninstall means it was already absent.
			if m.action == "uninstall" {
				p.installed = false
			} else {
				p.installed = true
			}
		case engine.StatusFailed:
			// Leave the previous status untouched on failure.
		}
		return
	}
}

// handleRunFinished moves to the summary screen.
func (m *model) handleRunFinished(out runOutcome) (tea.Model, tea.Cmd) {
	m.mode = modeDone
	m.summary = out.summary
	m.runErr = out.err
	m.cancel = nil
	m.progressCh = nil
	m.doneCh = nil
	if out.summary != nil {
		m.runDone = len(out.summary.Results)
		if out.summary.Total > 0 {
			m.runTotal = out.summary.Total
		}
	}
	return m, nil
}

// handleKey dispatches a key press according to the current mode.
func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Quit is available everywhere; during a run it also cancels the batch.
	if key == "ctrl+c" {
		return m, m.quit()
	}

	switch m.mode {
	case modeRunning:
		if key == "q" {
			return m, m.quit()
		}
		return m, nil

	case modeDone:
		switch key {
		case "q":
			return m, tea.Quit
		case "enter", "esc", "b":
			m.mode = modeBrowse
			m.summary = nil
			m.runResults = nil
			m.runErr = nil
			m.buildRows()
			return m, nil
		}
		return m, nil
	}

	// modeBrowse
	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "g", "home":
		if c := m.firstPkgRow(); c >= 0 {
			m.cursor = c
			m.ensureVisible()
		}
	case "G", "end":
		if c := m.lastPkgRow(); c >= 0 {
			m.cursor = c
			m.ensureVisible()
		}
	case " ":
		m.toggleCurrent()
	case "a":
		m.toggleAll()
	case "i":
		if cmd := m.startRun("install"); cmd != nil {
			return m, cmd
		}
	case "u":
		if cmd := m.startRun("uninstall"); cmd != nil {
			return m, cmd
		}
	case "r":
		if !m.loading {
			m.notice = ""
			return m, m.beginProbe()
		}
	}
	return m, nil
}

// quit cancels any in-flight batch and asks Bubble Tea to exit.
func (m *model) quit() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	return tea.Quit
}
