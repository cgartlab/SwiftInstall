package tui

import (
	"context"
	"strings"

	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// mode describes what the interface is currently doing.
type mode int

const (
	// modeBrowse is the normal package-selection view.
	modeBrowse mode = iota
	// modeRunning streams an install/uninstall batch.
	modeRunning
	// modeDone shows the final summary of a batch.
	modeDone
)

// pkgRow is the mutable per-package view state layered on top of the
// immutable manifest.Package.
type pkgRow struct {
	pkg       manifest.Package
	installed bool
	known     bool // whether the installed status has been probed yet
	selected  bool
}

// rowKind distinguishes the three kinds of line in the grouped list.
type rowKind int

const (
	rowHeader rowKind = iota
	rowSpacer
	rowPkg
)

// row is one display line of the grouped package list.
type row struct {
	kind     rowKind
	category string
	pkgIndex int
}

// model is the Bubble Tea model for the SwiftInstall TUI.
type model struct {
	opts   Options
	styles styles

	spinner spinner.Model

	mode   mode
	width  int
	height int

	packages []*pkgRow
	rows     []row
	cursor   int // index into rows; always points at a rowPkg when >= 0
	offset   int // first visible row (scroll position)

	// status probing
	loading    bool
	probed     int
	probeTotal int
	probeCh    chan probeMsg

	// batch execution
	action     string // "install" or "uninstall"
	runResults []engine.Result
	runTotal   int
	runDone    int
	summary    *engine.Summary
	runErr     error
	progressCh chan progressMsg
	doneCh     chan runOutcome
	cancel     context.CancelFunc

	// transient inline notice (e.g. "nothing selected")
	notice string
}

// newModel builds a model from Options. No I/O happens here, which keeps the
// whole type trivially testable without a terminal.
func newModel(opts Options) *model {
	m := &model{
		opts:   opts,
		styles: newStyles(opts.NoColor),
		width:  80,
		height: 24,
	}
	m.spinner = spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(m.styles.spinner),
	)

	if opts.Manifest != nil {
		for _, p := range opts.Manifest.Packages {
			m.packages = append(m.packages, &pkgRow{pkg: p, selected: true})
		}
	}
	m.probeTotal = len(m.packages)
	m.loading = opts.Backend != nil && len(m.packages) > 0

	m.buildRows()
	m.cursor = m.firstPkgRow()
	m.ensureVisible()
	return m
}

// Init starts the spinner and, when a backend is available, the async probe.
func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}
	if m.loading {
		cmds = append(cmds, m.beginProbe())
	}
	return tea.Batch(cmds...)
}

// busy reports whether a spinner should keep animating.
func (m *model) busy() bool {
	return m.loading || m.mode == modeRunning
}

// backendName returns a display name for the active backend.
func (m *model) backendName() string {
	if m.opts.Backend == nil {
		return "—"
	}
	return m.opts.Backend.Name()
}

// ---------------------------------------------------------------------------
// Grouping & cursor
// ---------------------------------------------------------------------------

// buildRows rebuilds the grouped display rows, preserving the cursor on the
// same package when possible.
func (m *model) buildRows() {
	prevID := ""
	if m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowPkg {
		prevID = m.packages[m.rows[m.cursor].pkgIndex].pkg.ID
	}

	// Bucket package indices by category, preserving first-appearance order of
	// both categories and packages within each category.
	var order []string
	buckets := make(map[string][]int)
	for i, p := range m.packages {
		cat := p.pkg.Category
		if cat == "" {
			cat = "未分类"
		}
		if _, ok := buckets[cat]; !ok {
			order = append(order, cat)
		}
		buckets[cat] = append(buckets[cat], i)
	}

	m.rows = m.rows[:0]
	for gi, cat := range order {
		if gi > 0 {
			m.rows = append(m.rows, row{kind: rowSpacer})
		}
		m.rows = append(m.rows, row{kind: rowHeader, category: cat})
		for _, idx := range buckets[cat] {
			m.rows = append(m.rows, row{kind: rowPkg, pkgIndex: idx})
		}
	}

	// Restore cursor to the previously selected package.
	m.cursor = m.firstPkgRow()
	if prevID != "" {
		for i, r := range m.rows {
			if r.kind == rowPkg && m.packages[r.pkgIndex].pkg.ID == prevID {
				m.cursor = i
				break
			}
		}
	}
	m.clampOffset()
}

// firstPkgRow returns the row index of the first package, or -1.
func (m *model) firstPkgRow() int {
	for i, r := range m.rows {
		if r.kind == rowPkg {
			return i
		}
	}
	return -1
}

// lastPkgRow returns the row index of the last package, or -1.
func (m *model) lastPkgRow() int {
	for i := len(m.rows) - 1; i >= 0; i-- {
		if m.rows[i].kind == rowPkg {
			return i
		}
	}
	return -1
}

// moveCursor moves the cursor delta package rows, skipping headers/spacers.
func (m *model) moveCursor(delta int) {
	if m.cursor < 0 {
		m.cursor = m.firstPkgRow()
		m.ensureVisible()
		return
	}
	i := m.cursor
	for {
		i += delta
		if i < 0 || i >= len(m.rows) {
			return
		}
		if m.rows[i].kind == rowPkg {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

// current returns the currently focused package row, or nil.
func (m *model) current() *pkgRow {
	if m.cursor < 0 || m.cursor >= len(m.rows) || m.rows[m.cursor].kind != rowPkg {
		return nil
	}
	return m.packages[m.rows[m.cursor].pkgIndex]
}

// ---------------------------------------------------------------------------
// Selection & counting
// ---------------------------------------------------------------------------

// toggleCurrent flips selection on the focused package.
func (m *model) toggleCurrent() {
	if p := m.current(); p != nil {
		p.selected = !p.selected
	}
}

// toggleAll selects every package, or clears them all when already all selected.
func (m *model) toggleAll() {
	all := m.selectedCount() == len(m.packages)
	for _, p := range m.packages {
		p.selected = !all
	}
}

func (m *model) selectedCount() int {
	n := 0
	for _, p := range m.packages {
		if p.selected {
			n++
		}
	}
	return n
}

func (m *model) installedCount() int {
	n := 0
	for _, p := range m.packages {
		if p.known && p.installed {
			n++
		}
	}
	return n
}

// selectedPackages returns the manifest packages currently selected.
func (m *model) selectedPackages() []manifest.Package {
	var out []manifest.Package
	for _, p := range m.packages {
		if p.selected {
			out = append(out, p.pkg)
		}
	}
	return out
}

// selectAllForTest is unused in production but kept trivial.

// ---------------------------------------------------------------------------
// Layout helpers
// ---------------------------------------------------------------------------

// cardOuter is the total rendered width of the framed card.
func (m *model) cardOuter() int {
	w := m.width
	if w <= 0 {
		w = 80
	}
	outer := w - 4 // two columns of breathing room on each side
	if outer > 88 {
		outer = 88
	}
	if outer < 12 {
		outer = 12
	}
	return outer
}

// textWidth is the usable width inside the card (borders + padding removed).
func (m *model) textWidth() int {
	tw := m.cardOuter() - 4
	if tw < 6 {
		tw = 6
	}
	return tw
}

// listHeight is how many list rows fit inside the card.
func (m *model) listHeight() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	// header(2) + blank(1) + card borders(2) + blank(1) + keybar(1) = 7,
	// minus separator + footer inside the card (2).
	lh := h - 7 - 2
	if lh < 3 {
		lh = 3
	}
	return lh
}

// ensureVisible scrolls so the cursor stays inside the visible window.
func (m *model) ensureVisible() {
	lh := m.listHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+lh {
		m.offset = m.cursor - lh + 1
	}
	m.clampOffset()
}

func (m *model) clampOffset() {
	lh := m.listHeight()
	maxOff := len(m.rows) - lh
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// truncateWidth trims s to at most max display columns, appending an ellipsis.
func truncateWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipglossWidth(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	r := []rune(s)
	for len(r) > 0 && lipglossWidth(string(r)) > max-1 {
		r = r[:len(r)-1]
	}
	return strings.TrimRight(string(r), " ") + "…"
}
