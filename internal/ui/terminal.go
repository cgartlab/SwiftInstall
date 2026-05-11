package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// Color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
)

// TerminalRenderer outputs beautiful, human-readable progress to stderr.
type TerminalRenderer struct {
	color bool
}

// NewTerminalRenderer creates a renderer with optional color support.
func NewTerminalRenderer(color bool) *TerminalRenderer {
	return &TerminalRenderer{color: color}
}

func (t *TerminalRenderer) Header(total int, source string, action string) {
	fmt.Fprintf(os.Stderr, "\n%s %s %d packages from %s%s\n\n",
		t.style(colorBold), action, total, source, t.style(colorReset))
}

func (t *TerminalRenderer) Progress(index int, total int, pkg manifest.Package, result engine.Result) {
	counter := fmt.Sprintf("[%d/%d]", index+1, total)
	icon, color := t.statusIcon(result.Status)

	line := fmt.Sprintf("  %s%s%s %s%s %s",
		t.style(colorDim), counter, t.style(colorReset),
		t.style(color), icon, t.style(colorReset))

	line += " " + pkg.ID

	if pkg.Category != "" {
		line += t.style(colorDim) + " (" + pkg.Category + ")" + t.style(colorReset)
	}

	if result.Error != "" {
		// Show first line of error only to keep output clean
		errMsg := strings.Split(result.Error, "\n")[0]
		if len(errMsg) > 60 {
			errMsg = errMsg[:57] + "..."
		}
		line += t.style(colorDim) + " — " + errMsg + t.style(colorReset)
	}

	fmt.Fprintln(os.Stderr, line)
}

func (t *TerminalRenderer) Summary(summary *engine.Summary) {
	if summary == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "\n%s%s────────────────────────────────────────%s\n",
		t.style(colorDim), t.style(colorBold), t.style(colorReset))

	// Build a compact summary line
	parts := []string{}
	if summary.Succeeded > 0 {
		parts = append(parts, fmt.Sprintf("%s%s%d succeeded%s",
			t.style(colorGreen), t.style(colorBold), summary.Succeeded, t.style(colorReset)))
	}
	if summary.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%s%d skipped%s",
			t.style(colorYellow), summary.Skipped, t.style(colorReset)))
	}
	if summary.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%s%s%d failed%s",
			t.style(colorRed), t.style(colorBold), summary.Failed, t.style(colorReset)))
	}

	fmt.Fprintf(os.Stderr, "  %s%sTotal: %d%s  |  %s  %s(%s)%s\n",
		t.style(colorBold), t.style(colorCyan), summary.Total, t.style(colorReset),
		strings.Join(parts, "  "),
		t.style(colorDim), formatDuration(summary.Duration), t.style(colorReset))

	// List failures at the end for easy identification
	if summary.Failed > 0 {
		fmt.Fprintf(os.Stderr, "\n  %sFailed packages:%s\n", t.style(colorRed), t.style(colorReset))
		for _, r := range summary.Results {
			if r.Status == engine.StatusFailed {
				errMsg := r.Error
				if len(errMsg) > 80 {
					errMsg = errMsg[:77] + "..."
				}
				fmt.Fprintf(os.Stderr, "    %s✗%s %s — %s\n",
					t.style(colorRed), t.style(colorReset), r.Package.ID, errMsg)
			}
		}
	}

	fmt.Fprintln(os.Stderr)
}

// statusIcon returns the icon and color for a given status.
func (t *TerminalRenderer) statusIcon(status engine.Status) (string, string) {
	switch status {
	case engine.StatusSuccess:
		return "✓", colorGreen
	case engine.StatusSkipped:
		return "○", colorYellow
	case engine.StatusOptionalFailed:
		return "⚠", colorYellow
	case engine.StatusFailed:
		return "✗", colorRed
	default:
		return "?", colorReset
	}
}

// style returns the ANSI escape code if color is enabled, empty string otherwise.
func (t *TerminalRenderer) style(code string) string {
	if t.color {
		return code
	}
	return ""
}

// formatDuration formats a duration in a human-friendly way.
func formatDuration(d time.Duration) string {
	secs := d.Seconds()
	if secs < 1 {
		return "<1s"
	}
	if secs < 60 {
		return fmt.Sprintf("%.1fs", secs)
	}
	mins := int(secs) / 60
	remainSecs := int(secs) % 60
	return fmt.Sprintf("%dm%ds", mins, remainSecs)
}
