package tui

import "github.com/charmbracelet/lipgloss"

// The palette is deliberately quiet: near-monochrome text with a single soft
// violet accent. Nothing shouts; hierarchy comes from weight, spacing and
// hairlines rather than colour.
var (
	colAccent = lipgloss.AdaptiveColor{Light: "#6f5bd6", Dark: "#b7a6ff"}
	colFg     = lipgloss.AdaptiveColor{Light: "#1f1f24", Dark: "#e9e9ee"}
	colMuted  = lipgloss.AdaptiveColor{Light: "#82828c", Dark: "#86868f"}
	colFaint  = lipgloss.AdaptiveColor{Light: "#b6b6c0", Dark: "#4c4c56"}
	colBorder = lipgloss.AdaptiveColor{Light: "#d8d8e0", Dark: "#3a3a44"}
	colOK     = lipgloss.AdaptiveColor{Light: "#2f9e6e", Dark: "#63d59d"}
	colWarn   = lipgloss.AdaptiveColor{Light: "#b07d16", Dark: "#e3c072"}
	colErr    = lipgloss.AdaptiveColor{Light: "#c0392b", Dark: "#e8847b"}
)

// styles holds every lipgloss style the UI needs, built once so colour can be
// disabled in a single place.
type styles struct {
	noColor bool

	appName  lipgloss.Style
	appTag   lipgloss.Style
	subtitle lipgloss.Style

	dim    lipgloss.Style
	faint  lipgloss.Style
	accent lipgloss.Style

	category lipgloss.Style
	marker   lipgloss.Style

	checkboxOn  lipgloss.Style
	checkboxOff lipgloss.Style

	statusOK      lipgloss.Style
	statusMissing lipgloss.Style
	statusUnknown lipgloss.Style

	pkgID       lipgloss.Style
	pkgIDActive lipgloss.Style
	tag         lipgloss.Style

	card      lipgloss.Style
	separator lipgloss.Style

	footer    lipgloss.Style
	lbl       lipgloss.Style
	lblValue  lipgloss.Style
	lblAccent lipgloss.Style

	key     lipgloss.Style
	keyDesc lipgloss.Style

	spinner lipgloss.Style

	okMark   lipgloss.Style
	badMark  lipgloss.Style
	skipMark lipgloss.Style
}

// newStyles builds the style set. When noColor is true every foreground and
// border colour collapses to lipgloss.NoColor, so the UI stays legible on a
// plain terminal (or when piped) while keeping its weight and spacing.
func newStyles(noColor bool) styles {
	c := func(col lipgloss.TerminalColor) lipgloss.TerminalColor {
		if noColor {
			return lipgloss.NoColor{}
		}
		return col
	}

	s := styles{noColor: noColor}

	s.appName = lipgloss.NewStyle().Bold(true).Foreground(c(colAccent))
	s.appTag = lipgloss.NewStyle().Foreground(c(colFaint))
	s.subtitle = lipgloss.NewStyle().Foreground(c(colMuted))

	s.dim = lipgloss.NewStyle().Foreground(c(colMuted))
	s.faint = lipgloss.NewStyle().Foreground(c(colFaint))
	s.accent = lipgloss.NewStyle().Foreground(c(colAccent))

	s.category = lipgloss.NewStyle().Bold(true).Foreground(c(colMuted))
	s.marker = lipgloss.NewStyle().Foreground(c(colAccent)).Bold(true)

	s.checkboxOn = lipgloss.NewStyle().Foreground(c(colAccent)).Bold(true)
	s.checkboxOff = lipgloss.NewStyle().Foreground(c(colFaint))

	s.statusOK = lipgloss.NewStyle().Foreground(c(colOK))
	s.statusMissing = lipgloss.NewStyle().Foreground(c(colFaint))
	s.statusUnknown = lipgloss.NewStyle().Foreground(c(colFaint))

	s.pkgID = lipgloss.NewStyle().Foreground(c(colFg))
	s.pkgIDActive = lipgloss.NewStyle().Foreground(c(colFg)).Bold(true)
	s.tag = lipgloss.NewStyle().Foreground(c(colFaint))

	s.card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(colBorder)).
		Padding(0, 1)

	s.separator = lipgloss.NewStyle().Foreground(c(colBorder))

	s.footer = lipgloss.NewStyle().Foreground(c(colMuted))
	s.lbl = lipgloss.NewStyle().Foreground(c(colMuted))
	s.lblValue = lipgloss.NewStyle().Foreground(c(colFg)).Bold(true)
	s.lblAccent = lipgloss.NewStyle().Foreground(c(colAccent)).Bold(true)

	s.key = lipgloss.NewStyle().Foreground(c(colAccent))
	s.keyDesc = lipgloss.NewStyle().Foreground(c(colMuted))

	s.spinner = lipgloss.NewStyle().Foreground(c(colAccent))

	s.okMark = lipgloss.NewStyle().Foreground(c(colOK))
	s.badMark = lipgloss.NewStyle().Foreground(c(colErr))
	s.skipMark = lipgloss.NewStyle().Foreground(c(colWarn))

	return s
}
