package backend

import (
	"context"
)

type Backend interface {
	Name() string
	Detect() error
	IsInstalled(ctx context.Context, id string) (bool, error)
	Install(ctx context.Context, id string, opts InstallOptions) (*Output, error)
}

type InstallOptions struct {
	Proxy string
}

type Output struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}
