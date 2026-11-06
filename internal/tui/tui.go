// Package tui implements the interactive, keyboard-driven terminal interface
// for SwiftInstall (`sis tui`).
//
// The design goal is a calm, single-screen product: one framed list of
// packages grouped by category, a quiet near-monochrome palette with a single
// soft accent, hairline framing and a persistent keybinding bar. Nothing on
// screen competes for attention; the terminal width is respected and status
// probing happens entirely off the UI thread.
package tui

import (
	"fmt"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	tea "github.com/charmbracelet/bubbletea"
)

// Options is the complete input for the TUI. The CLI resolves the manifest,
// backend and effective settings before calling Run.
type Options struct {
	Manifest   *manifest.Manifest
	Source     string
	Backend    backend.Backend
	Proxy      string
	RetryCount int
	RetryDelay int
	NoColor    bool
}

// Run starts the interactive program and blocks until the user quits.
func Run(opts Options) error {
	if opts.Manifest == nil {
		return fmt.Errorf("未提供清单（manifest 为 nil）")
	}

	m := newModel(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("交互界面启动失败: %w", err)
	}
	return nil
}
