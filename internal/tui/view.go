package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/charmbracelet/lipgloss"
)

// lipglossWidth is a tiny indirection so model.go can measure display width
// without importing lipgloss directly.
func lipglossWidth(s string) int { return lipgloss.Width(s) }

// hint is one keybinding shown in the bottom bar.
type hint struct {
	key  string
	desc string
}

// View implements tea.Model.
func (m *model) View() string {
	if m.width <= 0 {
		m.width = 80
	}
	if m.height <= 0 {
		m.height = 24
	}

	blocks := []string{
		m.renderHeader(),
		"",
		m.renderCard(),
		"",
	}
	if m.notice != "" {
		blocks = append(blocks, m.styles.skipMark.Render(m.notice))
	}
	blocks = append(blocks, m.renderKeybar(m.hints()))

	view := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return lipgloss.NewStyle().PaddingLeft(2).Render(view)
}

// renderHeader draws the calm two-line masthead.
func (m *model) renderHeader() string {
	line1 := m.styles.appName.Render("sis") + m.styles.appTag.Render("  SwiftInstall")

	src := m.opts.Source
	if src == "" {
		src = "(未指定清单)"
	}
	sub := fmt.Sprintf("%s  ·  %s", m.backendName(), src)
	sub = truncateWidth(sub, m.cardOuter()+2)
	line2 := m.styles.subtitle.Render(sub)

	return line1 + "\n" + line2
}

// renderCard draws the framed list / progress area.
func (m *model) renderCard() string {
	tw := m.textWidth()
	lh := m.listHeight()

	sep := m.styles.separator.Render(strings.Repeat("─", tw))
	body := m.renderBody(lh, tw) + "\n" + sep + "\n" + m.renderFooter()

	return m.styles.card.Width(m.cardOuter() - 2).Render(body)
}

// renderBody renders the scrollable list in browse mode, or the batch log in
// running/done mode, always padded to exactly lh lines.
func (m *model) renderBody(lh, tw int) string {
	if m.mode == modeRunning || m.mode == modeDone {
		return m.renderProgressBody(lh, tw)
	}

	m.ensureVisible()
	lines := make([]string, 0, lh)
	for i := m.offset; i < len(m.rows) && len(lines) < lh; i++ {
		lines = append(lines, m.renderRowLine(i))
	}
	for len(lines) < lh {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// renderRowLine renders a single list row (header, spacer or package).
func (m *model) renderRowLine(rowIdx int) string {
	tw := m.textWidth()
	r := m.rows[rowIdx]

	switch r.kind {
	case rowSpacer:
		return ""
	case rowHeader:
		label := truncateWidth(strings.ToUpper(r.category), maxInt(tw-2, 1))
		return m.styles.faint.Render("· ") + m.styles.category.Render(label)
	case rowPkg:
		active := rowIdx == m.cursor && m.mode == modeBrowse
		return m.renderPackageRow(r.pkgIndex, active)
	}
	return ""
}

// renderPackageRow renders one package: cursor bar, checkbox, status glyph,
// ID and a dim category tag pinned to the right edge.
func (m *model) renderPackageRow(idx int, active bool) string {
	tw := m.textWidth()
	p := m.packages[idx]

	marker := "  "
	if active {
		marker = m.styles.marker.Render("▌") + " "
	}

	checkbox := m.styles.checkboxOff.Render("[ ]")
	if p.selected {
		checkbox = m.styles.checkboxOn.Render("[x]")
	}

	glyph, gStyle := m.statusGlyph(p)
	glyphStr := gStyle.Render(glyph)

	// The dim category tag is the first thing to go on narrow terminals.
	tagText := ""
	if p.pkg.Category != "" && tw >= 30 {
		tagText = truncateWidth(p.pkg.Category, 20)
	}
	tagW := lipglossWidth(tagText)

	// marker(2) + checkbox(3) + space + glyph(1) + space = 8 fixed columns.
	fixed := 8
	idMax := tw - fixed - tagW
	if tagW > 0 {
		idMax-- // gap before the right-aligned tag
	}
	if idMax < 4 {
		idMax = 4
	}
	idText := truncateWidth(p.pkg.ID, idMax)

	idStyle := m.styles.pkgID
	if active {
		idStyle = m.styles.pkgIDActive
	}

	leftPlain := fixed + lipglossWidth(idText)
	left := marker + checkbox + " " + glyphStr + " " + idStyle.Render(idText)

	gap := tw - leftPlain - tagW
	if gap < 0 {
		gap = 0
	}
	line := left + strings.Repeat(" ", gap)
	if tagW > 0 {
		line += m.styles.tag.Render(tagText)
	}
	return line
}

// statusGlyph returns the installed-status glyph and its style.
func (m *model) statusGlyph(p *pkgRow) (string, lipgloss.Style) {
	switch {
	case !p.known:
		return "·", m.styles.statusUnknown
	case p.installed:
		return "✓", m.styles.statusOK
	default:
		return "○", m.styles.statusMissing
	}
}

// renderFooter draws the counts / probing indicator line.
func (m *model) renderFooter() string {
	tw := m.textWidth()
	total := len(m.packages)
	sel := m.selectedCount()
	inst := m.installedCount()

	var leftPlain, left string
	if tw < 38 {
		// Compact footer for narrow terminals.
		leftPlain = fmt.Sprintf("%d pkgs · %d sel · %d inst", total, sel, inst)
		left = m.styles.lblValue.Render(fmt.Sprintf("%d", total)) + m.styles.lbl.Render(" pkgs") +
			m.styles.faint.Render(" · ") +
			m.styles.lblAccent.Render(fmt.Sprintf("%d", sel)) + m.styles.lbl.Render(" sel") +
			m.styles.faint.Render(" · ") +
			m.styles.lblValue.Render(fmt.Sprintf("%d", inst)) + m.styles.lbl.Render(" inst")
	} else {
		leftPlain = fmt.Sprintf("packages %d  ·  selected %d  ·  installed %d", total, sel, inst)
		left = m.styles.lbl.Render("packages ") + m.styles.lblValue.Render(fmt.Sprintf("%d", total)) +
			m.styles.faint.Render("  ·  ") +
			m.styles.lbl.Render("selected ") + m.styles.lblAccent.Render(fmt.Sprintf("%d", sel)) +
			m.styles.faint.Render("  ·  ") +
			m.styles.lbl.Render("installed ") + m.styles.lblValue.Render(fmt.Sprintf("%d", inst))
	}

	// The probing indicator only belongs on the browse screen during the
	// initial status probe — never while a batch runs or on the summary.
	if m.mode != modeBrowse || !m.loading {
		return left
	}

	rightPlain := fmt.Sprintf("probing %d/%d", m.probed, m.probeTotal)
	right := m.spinner.View() + m.styles.dim.Render(" "+rightPlain)
	rightWidth := 2 + lipglossWidth(rightPlain) // spinner(1) + space(1) + text

	gap := tw - lipglossWidth(leftPlain) - rightWidth
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// renderProgressBody renders the batch log and, when finished, the summary.
func (m *model) renderProgressBody(lh, tw int) string {
	lines := make([]string, 0, lh)

	if m.mode == modeRunning {
		title := "Installing"
		if m.action == "uninstall" {
			title = "Uninstalling"
		}
		head := m.spinner.View() + " " + m.styles.lblValue.Render(title) +
			m.styles.dim.Render(fmt.Sprintf("  %d/%d", m.runDone, m.runTotal))
		lines = append(lines, head)
	} else {
		lines = append(lines, m.renderSummaryHeadline())
	}

	room := lh - 1
	if room < 0 {
		room = 0
	}
	results := m.runResults
	if len(results) > room {
		results = results[len(results)-room:]
	}
	for _, r := range results {
		lines = append(lines, m.renderResultLine(r, tw))
	}
	for len(lines) < lh {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// renderSummaryHeadline renders the one-line outcome.
func (m *model) renderSummaryHeadline() string {
	s := m.summary
	if s == nil {
		if m.runErr != nil {
			return m.styles.badMark.Render("✗ ") + m.styles.lblValue.Render(truncateWidth(m.runErr.Error(), m.textWidth()-2))
		}
		return m.styles.okMark.Render("✔ ") + m.styles.lblValue.Render("Done")
	}

	head := m.styles.okMark.Render("✔ ") + m.styles.lblValue.Render("Done")
	head += m.styles.faint.Render("  ·  ")
	head += m.styles.lblValue.Render(fmt.Sprintf("%d", s.Succeeded)) + m.styles.lbl.Render(" succeeded")
	if s.Skipped > 0 {
		head += m.styles.faint.Render("  ·  ") + m.styles.skipMark.Render(fmt.Sprintf("%d", s.Skipped)) + m.styles.lbl.Render(" skipped")
	}
	if s.Failed > 0 {
		head += m.styles.faint.Render("  ·  ") + m.styles.badMark.Render(fmt.Sprintf("%d", s.Failed)) + m.styles.lbl.Render(" failed")
	}
	head += m.styles.faint.Render("  (" + formatDuration(s.Duration) + ")")
	return head
}

// renderResultLine renders one finished package in the batch log.
func (m *model) renderResultLine(r engine.Result, tw int) string {
	glyph, gStyle := resultGlyph(m.styles, r.Status)
	line := gStyle.Render(glyph) + " " + m.styles.pkgID.Render(truncateWidth(r.Package.ID, tw-4))

	if r.Error != "" {
		msg := firstLine(r.Error)
		msg = truncateWidth(msg, maxInt(tw-lipglossWidth(r.Package.ID)-8, 6))
		line += m.styles.faint.Render("  " + msg)
	}
	return line
}

// resultGlyph maps an engine status to a glyph and style.
func resultGlyph(st styles, status engine.Status) (string, lipgloss.Style) {
	switch status {
	case engine.StatusSuccess:
		return "✓", st.okMark
	case engine.StatusSkipped:
		return "○", st.skipMark
	case engine.StatusOptionalFailed:
		return "⚠", st.skipMark
	case engine.StatusFailed:
		return "✗", st.badMark
	default:
		return "·", st.dim
	}
}

// renderKeybar renders the bottom keybinding hint bar, trimming hints that do
// not fit the available width.
func (m *model) renderKeybar(hints []hint) string {
	budget := m.cardOuter()
	sep := m.styles.faint.Render(" · ")

	var parts []string
	curW := 0
	for _, h := range hints {
		segPlain := h.key + " " + h.desc
		addW := lipglossWidth(segPlain)
		sepW := 0
		if len(parts) > 0 {
			sepW = lipglossWidth(" · ")
		}
		if curW+sepW+addW > budget {
			break
		}
		parts = append(parts, m.styles.key.Render(h.key)+" "+m.styles.keyDesc.Render(h.desc))
		curW += sepW + addW
	}
	return strings.Join(parts, sep)
}

// hints returns the keybinding set for the current mode.
func (m *model) hints() []hint {
	switch m.mode {
	case modeRunning:
		return []hint{{"ctrl+c", "cancel"}}
	case modeDone:
		return []hint{{"enter", "back"}, {"q", "quit"}}
	default:
		return []hint{
			{"↑↓", "move"},
			{"space", "select"},
			{"a", "all"},
			{"i", "install"},
			{"u", "uninstall"},
			{"r", "refresh"},
			{"q", "quit"},
		}
	}
}

// formatDuration formats a duration compactly.
func formatDuration(d time.Duration) string {
	secs := d.Seconds()
	if secs < 1 {
		return "<1s"
	}
	if secs < 60 {
		return fmt.Sprintf("%.1fs", secs)
	}
	return fmt.Sprintf("%dm%ds", int(secs)/60, int(secs)%60)
}

// firstLine returns the first line of a possibly multi-line error.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
