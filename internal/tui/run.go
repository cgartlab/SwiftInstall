package tui

import (
	"context"

	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	tea "github.com/charmbracelet/bubbletea"
)

// probeMsg carries the installed status of a single package discovered by the
// background probe.
type probeMsg struct {
	index     int
	installed bool
	err       error
}

// probeDoneMsg signals that the background probe has finished.
type probeDoneMsg struct{}

// progressMsg carries one engine result from the background batch.
type progressMsg struct {
	index  int
	total  int
	result engine.Result
}

// runOutcome is the terminal result of a batch.
type runOutcome struct {
	summary *engine.Summary
	err     error
	action  string
}

// runFinishedMsg wraps a runOutcome as a Bubble Tea message.
type runFinishedMsg runOutcome

// beginProbe launches the asynchronous installed-status probe and returns the
// command that starts pumping results back into the UI. The probe runs in its
// own goroutine so typing and navigation never block.
func (m *model) beginProbe() tea.Cmd {
	be := m.opts.Backend
	if be == nil || len(m.packages) == 0 {
		m.loading = false
		return nil
	}

	pkgs := make([]manifest.Package, len(m.packages))
	for i, p := range m.packages {
		pkgs[i] = p.pkg
	}

	ch := make(chan probeMsg, len(pkgs))
	m.probeCh = ch
	m.loading = true
	m.probed = 0
	m.probeTotal = len(pkgs)

	launch := func() tea.Msg {
		go func() {
			// Sequential probing is gentle on the package manager and keeps
			// the output deterministic; it is still fully off the UI thread.
			for i, p := range pkgs {
				installed, err := be.IsInstalled(context.Background(), p.ID)
				ch <- probeMsg{index: i, installed: installed, err: err}
			}
			close(ch)
		}()
		return nil
	}

	return tea.Batch(launch, m.waitProbe())
}

// waitProbe returns a command that waits for the next probe result.
func (m *model) waitProbe() tea.Cmd {
	ch := m.probeCh
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return probeDoneMsg{}
		}
		return msg
	}
}

// startRun kicks off an install/uninstall batch for the selected packages.
// It returns nil when there is nothing to do (no selection or no backend).
func (m *model) startRun(action string) tea.Cmd {
	if m.opts.Backend == nil {
		m.notice = "未检测到可用的包管理器后端"
		return nil
	}
	selected := m.selectedPackages()
	if len(selected) == 0 {
		m.notice = "未选择任何软件包"
		return nil
	}

	man := &manifest.Manifest{Packages: selected}
	eng := engine.New(m.opts.Backend, engine.Options{
		Proxy:      m.opts.Proxy,
		RetryCount: m.opts.RetryCount,
		RetryDelay: m.opts.RetryDelay,
	})

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.action = action
	m.mode = modeRunning
	m.notice = ""
	m.runResults = nil
	m.runTotal = len(selected)
	m.runDone = 0
	m.summary = nil
	m.runErr = nil

	progressCh := make(chan progressMsg, len(selected))
	doneCh := make(chan runOutcome, 1)
	m.progressCh = progressCh
	m.doneCh = doneCh

	eng.OnProgress(func(index, total int, result engine.Result) {
		progressCh <- progressMsg{index: index, total: total, result: result}
	})

	launch := func() tea.Msg {
		go func() {
			var summary *engine.Summary
			var err error
			if action == "uninstall" {
				summary, err = eng.Uninstall(ctx, man)
			} else {
				summary, err = eng.Install(ctx, man)
			}
			close(progressCh)
			doneCh <- runOutcome{summary: summary, err: err, action: action}
		}()
		return nil
	}

	return tea.Batch(m.spinner.Tick, launch, m.waitProgress())
}

// waitProgress returns a command that waits for the next batch result or the
// terminal outcome.
func (m *model) waitProgress() tea.Cmd {
	ch := m.progressCh
	done := m.doneCh
	if ch == nil || done == nil {
		return nil
	}
	return func() tea.Msg {
		// Receiving on the closed channel only reports ok==false after every
		// buffered result has been drained, so no progress line is skipped.
		if msg, ok := <-ch; ok {
			return msg
		}
		return runFinishedMsg(<-done)
	}
}
