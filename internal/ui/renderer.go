package ui

import (
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// Renderer defines how batch operations are presented to the user.
type Renderer interface {
	// Header prints the operation header (e.g. "Installing 5 packages from sis.yaml")
	Header(total int, source string, action string)

	// Progress reports a single package result during the operation.
	Progress(index int, total int, pkg manifest.Package, result engine.Result)

	// Summary prints the final aggregated result.
	Summary(summary *engine.Summary)
}
