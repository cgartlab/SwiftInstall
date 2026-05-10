package ui

import (
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

type TerminalRenderer struct{}

func NewTerminalRenderer() *TerminalRenderer {
	return &TerminalRenderer{}
}

func (t *TerminalRenderer) Start(total int, manifestPath string, action string) {
	fmt.Fprintf(os.Stderr, "%s %d packages from %s\n", action, total, manifestPath)
}

func (t *TerminalRenderer) Progress(pkg manifest.Package, status string, err error) {
	icon := "✓"
	switch status {
	case string(engine.StatusFailed):
		icon = "✗"
	case string(engine.StatusSkipped):
		icon = "⚠"
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s %s — %v\n", icon, pkg.ID, err)
	} else {
		fmt.Fprintf(os.Stderr, "  %s %s\n", icon, pkg.ID)
	}
}

func (t *TerminalRenderer) Done(summary *engine.Summary) {
	if summary == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "\nSummary: %d total, %d succeeded, %d skipped, %d failed (%.1fs)\n",
		summary.Total, summary.Succeeded, summary.Skipped, summary.Failed, summary.Duration.Seconds())

	for _, r := range summary.Results {
		if r.Status != engine.StatusFailed {
			continue
		}
		desc := string(r.Status)
		if len(r.Errors) > 0 {
			desc = r.Errors[0]
		}
		fmt.Fprintf(os.Stderr, "  ✗ %s — %s\n", r.Package.ID, desc)
	}
}
