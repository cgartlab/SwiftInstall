package backend

import (
	"context"
)

// Backend defines the interface for a package manager backend.
// Each platform (Windows/macOS) implements this interface.
type Backend interface {
	// Name returns the backend identifier (e.g. "winget", "homebrew").
	Name() string

	// Detect checks if the package manager is available on this system.
	// Must be called before any other method. Returns a user-friendly error
	// if the package manager is not found or not usable.
	Detect() error

	// IsInstalled checks whether a specific package is installed.
	IsInstalled(ctx context.Context, id string) (bool, error)

	// Install installs a package by ID with the given options.
	Install(ctx context.Context, id string, opts InstallOptions) (*Output, error)

	// Uninstall removes a package by ID.
	Uninstall(ctx context.Context, id string) (*Output, error)
}

// InstallOptions holds parameters that affect installation behavior.
type InstallOptions struct {
	Proxy string // HTTP/HTTPS proxy URL
	// Kind selects the backend channel for the package (e.g. Homebrew
	// "formula" vs "cask"). Empty means auto-detect. Backends that do not
	// distinguish channels ignore it.
	Kind string
}

// Output captures the result of a backend command execution.
type Output struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}
