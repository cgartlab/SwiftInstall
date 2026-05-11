package engine

import (
	"context"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// Status represents the outcome of a single package operation.
type Status string

const (
	StatusSuccess        Status = "success"
	StatusSkipped        Status = "skipped"
	StatusFailed         Status = "failed"
	StatusOptionalFailed Status = "optional_failed"
)

// Result holds the outcome of a single package operation.
type Result struct {
	Package  manifest.Package `json:"package"`
	Status   Status           `json:"status"`
	Error    string           `json:"error,omitempty"`
	Duration time.Duration    `json:"duration_ms,omitempty"`
}

// Summary holds the aggregated outcome of a batch operation.
type Summary struct {
	Total     int           `json:"total"`
	Succeeded int           `json:"succeeded"`
	Skipped   int           `json:"skipped"`
	Failed    int           `json:"failed"`
	Results   []Result      `json:"results"`
	Duration  time.Duration `json:"duration"`
}

// Options controls the behavior of an engine run.
type Options struct {
	DryRun       bool
	SkipExisting bool
	Proxy        string
	RetryCount   int // number of retries (0 = no retry, only 1 attempt)
	RetryDelay   int // seconds between retries
}

// ProgressFunc is called after each package is processed.
// index is 0-based position in the batch.
type ProgressFunc func(index int, total int, result Result)

// Engine orchestrates batch install/uninstall operations.
type Engine struct {
	backend    backend.Backend
	opts       Options
	onProgress ProgressFunc
}

// New creates an Engine with the given backend and options.
func New(be backend.Backend, opts Options) *Engine {
	return &Engine{
		backend: be,
		opts:    opts,
	}
}

// OnProgress sets the progress callback. Must be called before Install/Uninstall.
func (e *Engine) OnProgress(fn ProgressFunc) {
	e.onProgress = fn
}

// Install processes all packages in the manifest.
// It deduplicates, handles skip-existing, dry-run, and retries.
func (e *Engine) Install(ctx context.Context, m *manifest.Manifest) (*Summary, error) {
	packages := deduplicate(m.Packages)
	start := time.Now()
	summary := &Summary{Total: len(packages)}

	for i, pkg := range packages {
		select {
		case <-ctx.Done():
			summary.Duration = time.Since(start)
			return summary, ctx.Err()
		default:
		}

		result := e.installOne(ctx, pkg)
		summary.Results = append(summary.Results, result)

		switch result.Status {
		case StatusSuccess:
			summary.Succeeded++
		case StatusSkipped:
			summary.Skipped++
		case StatusOptionalFailed:
			summary.Skipped++
		case StatusFailed:
			summary.Failed++
		}

		if e.onProgress != nil {
			e.onProgress(i, len(packages), result)
		}
	}

	summary.Duration = time.Since(start)
	return summary, nil
}

// Uninstall removes all packages listed in the manifest.
func (e *Engine) Uninstall(ctx context.Context, m *manifest.Manifest) (*Summary, error) {
	packages := deduplicate(m.Packages)
	start := time.Now()
	summary := &Summary{Total: len(packages)}

	for i, pkg := range packages {
		select {
		case <-ctx.Done():
			summary.Duration = time.Since(start)
			return summary, ctx.Err()
		default:
		}

		result := e.uninstallOne(ctx, pkg)
		summary.Results = append(summary.Results, result)

		switch result.Status {
		case StatusSuccess:
			summary.Succeeded++
		case StatusSkipped:
			summary.Skipped++
		case StatusFailed:
			summary.Failed++
		}

		if e.onProgress != nil {
			e.onProgress(i, len(packages), result)
		}
	}

	summary.Duration = time.Since(start)
	return summary, nil
}

// installOne handles a single package: skip-existing check, dry-run, and retry loop.
func (e *Engine) installOne(ctx context.Context, pkg manifest.Package) Result {
	pkgStart := time.Now()

	// Skip if already installed
	if e.opts.SkipExisting && !e.opts.DryRun {
		installed, err := e.backend.IsInstalled(ctx, pkg.ID)
		if err == nil && installed {
			return Result{
				Package:  pkg,
				Status:   StatusSkipped,
				Duration: time.Since(pkgStart),
			}
		}
	}

	// Dry run: report success without doing anything
	if e.opts.DryRun {
		return Result{
			Package:  pkg,
			Status:   StatusSuccess,
			Duration: time.Since(pkgStart),
		}
	}

	// Actual install with retry
	err := e.retry(ctx, func() error {
		_, installErr := e.backend.Install(ctx, pkg.ID, backend.InstallOptions{
			Proxy: e.opts.Proxy,
		})
		return installErr
	})

	if err != nil {
		status := StatusFailed
		if pkg.Optional {
			status = StatusOptionalFailed
		}
		return Result{
			Package:  pkg,
			Status:   status,
			Error:    err.Error(),
			Duration: time.Since(pkgStart),
		}
	}

	return Result{
		Package:  pkg,
		Status:   StatusSuccess,
		Duration: time.Since(pkgStart),
	}
}

// uninstallOne handles a single package uninstall with retry.
func (e *Engine) uninstallOne(ctx context.Context, pkg manifest.Package) Result {
	pkgStart := time.Now()

	if e.opts.DryRun {
		return Result{
			Package:  pkg,
			Status:   StatusSuccess,
			Duration: time.Since(pkgStart),
		}
	}

	// Check if installed first — skip if not present
	installed, err := e.backend.IsInstalled(ctx, pkg.ID)
	if err == nil && !installed {
		return Result{
			Package:  pkg,
			Status:   StatusSkipped,
			Duration: time.Since(pkgStart),
		}
	}

	err = e.retry(ctx, func() error {
		_, uninstallErr := e.backend.Uninstall(ctx, pkg.ID)
		return uninstallErr
	})

	if err != nil {
		return Result{
			Package:  pkg,
			Status:   StatusFailed,
			Error:    err.Error(),
			Duration: time.Since(pkgStart),
		}
	}

	return Result{
		Package:  pkg,
		Status:   StatusSuccess,
		Duration: time.Since(pkgStart),
	}
}

// retry executes fn up to (RetryCount + 1) times with delay between attempts.
func (e *Engine) retry(ctx context.Context, fn func() error) error {
	maxAttempts := e.opts.RetryCount + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 && e.opts.RetryDelay > 0 {
			delay := time.Duration(e.opts.RetryDelay) * time.Second
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check context between retries
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return lastErr
}

// deduplicate removes duplicate packages by ID, keeping the first occurrence.
func deduplicate(packages []manifest.Package) []manifest.Package {
	seen := make(map[string]bool, len(packages))
	result := make([]manifest.Package, 0, len(packages))
	for _, p := range packages {
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		result = append(result, p)
	}
	return result
}
