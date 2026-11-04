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
	// Asset names the GitHub release asset (or substring) to download, used
	// by the GitHub backend. May contain {os}/{arch} placeholders.
	Asset string
}

// Output captures the result of a backend command execution.
type Output struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

// InstalledSetProvider is an optional capability a Backend may implement when
// it can report the full set of installed package identifiers in a single call.
// It is far cheaper than probing each package with IsInstalled (one `brew` call
// instead of two per package), so callers such as the GUI should prefer it and
// fall back to per-package probing when a backend does not implement it.
type InstalledSetProvider interface {
	// InstalledSet returns the set of installed package identifiers (formula
	// names and cask tokens together), keyed by identifier.
	InstalledSet(ctx context.Context) (map[string]bool, error)
}

// SearchHit is one result from a package-repository search.
type SearchHit struct {
	Name string `json:"name"`
	// Kind is the backend channel: "formula" or "cask" when known, else "".
	Kind string `json:"kind,omitempty"`
	// Source is where the hit came from: "brew", "npm" or "github".
	Source string `json:"source"`
	// Desc is a short description when available.
	Desc string `json:"desc,omitempty"`
}

// Searcher is an optional capability a Backend may implement to search the
// full package repository (not just installed packages). Homebrew implements
// it via `brew search`; winget does not.
type Searcher interface {
	Search(ctx context.Context, query string) ([]SearchHit, error)
}

// UpdateEntry is one item in a package's update history.
type UpdateEntry struct {
	Date    string `json:"date"`    // ISO date (short)
	Message string `json:"message"` // commit / release message
}

// PackageInfo is the metadata shown when a user expands a package row.
type PackageInfo struct {
	Name       string        `json:"name"`
	Kind       string        `json:"kind"` // formula | cask
	Desc       string        `json:"desc"`
	Version    string        `json:"version"`
	Homepage   string        `json:"homepage"`
	SourceURL  string        `json:"source_url"`  // download / source url
	Updated    string        `json:"updated"`     // last update date (short ISO)
	Updates    []UpdateEntry `json:"updates"`     // recent update history
	HistoryURL string        `json:"history_url"` // link to full history
}

// InfoProvider is an optional capability a Backend may implement to return
// detailed metadata for a single package (description, version, update history).
type InfoProvider interface {
	Info(ctx context.Context, name string) (*PackageInfo, error)
}
