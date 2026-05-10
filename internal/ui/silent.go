package ui

import (
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

type SilentRenderer struct{}

func NewSilentRenderer() *SilentRenderer {
	return &SilentRenderer{}
}

func (s *SilentRenderer) Start(total int, manifestPath string, action string) {}

func (s *SilentRenderer) Progress(pkg manifest.Package, status string, err error) {}

func (s *SilentRenderer) Done(summary *engine.Summary) {}
