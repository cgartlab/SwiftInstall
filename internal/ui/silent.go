package ui

import (
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// SilentRenderer suppresses all output. Useful for scripting scenarios
// where only the exit code matters.
type SilentRenderer struct{}

func NewSilentRenderer() *SilentRenderer {
	return &SilentRenderer{}
}

func (s *SilentRenderer) Header(total int, source string, action string) {}

func (s *SilentRenderer) Progress(index int, total int, pkg manifest.Package, result engine.Result) {}

func (s *SilentRenderer) Summary(summary *engine.Summary) {}
