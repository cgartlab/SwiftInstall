package backend

import (
	"context"
	"errors"
	"fmt"
)

type Backend interface {
	Name() string
	Detect() error
	IsInstalled(ctx context.Context, id string) (bool, error)
	Install(ctx context.Context, id string, opts InstallOptions) (*Output, error)
	Uninstall(ctx context.Context, id string, opts InstallOptions) (*Output, error)
	ListInstalled(ctx context.Context) ([]string, error)
	IsPermanent(err error) bool
}

type InstallOptions struct {
	Proxy  string
	DryRun bool
}

type Output struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

var ErrAlreadyInstalled = errors.New("package already installed")

func FormatWingetError(exitCode int) error {
	switch exitCode {
	case 0:
		return nil
	case 0x8A150011:
		return fmt.Errorf("%w: already installed (0x8A150011)", ErrAlreadyInstalled)
	case 0x8A150019:
		return fmt.Errorf("package not found (0x8A150019)")
	case 0x8A15003F:
		return fmt.Errorf("no applicable package found (0x8A15003F)")
	default:
		return fmt.Errorf("winget exit code 0x%08X", exitCode)
	}
}
