package ui

import (
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

type Renderer interface {
	Start(total int, manifestPath string, action string)
	Progress(pkg manifest.Package, status string, err error)
	Done(summary *engine.Summary)
}
