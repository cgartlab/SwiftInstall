package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
)

type Package struct {
	ID       string `yaml:"id" json:"id"`
	Category string `yaml:"category,omitempty" json:"category,omitempty"`
}

type Manifest struct {
	Mirror   string    `yaml:"mirror,omitempty" json:"mirror,omitempty"`
	Proxy    string    `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Packages []Package `yaml:"packages" json:"packages"`
}

type Status string

const (
	StatusSuccess Status = "success"
	StatusSkipped Status = "skipped"
	StatusFailed  Status = "failed"
)

type InstallResult struct {
	Package    Package  `json:"package"`
	Status     Status   `json:"status"`
	Errors     []string `json:"errors,omitempty"`
	DurationMs int64    `json:"duration_ms,omitempty"`
}

type Summary struct {
	Total     int             `json:"total"`
	Succeeded int             `json:"succeeded"`
	Skipped   int             `json:"skipped"`
	Failed    int             `json:"failed"`
	Results   []InstallResult `json:"results"`
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	Duration  time.Duration   `json:"duration"`
}

type PackageStatus struct {
	Package   Package `json:"package"`
	Installed bool    `json:"installed"`
	Version   string  `json:"version,omitempty"`
}

type Config struct {
	DryRun       bool
	SkipExisting bool
	RetryCount   int
	RetryDelay   time.Duration
	Proxy        string
}

type Engine struct {
	backend    backend.Backend
	cfg        *Config
	progressFn func(InstallResult)
}

func New(be backend.Backend, cfg *Config) *Engine {
	return &Engine{
		backend: be,
		cfg:     cfg,
	}
}

func (e *Engine) SetProgressHook(fn func(InstallResult)) {
	e.progressFn = fn
}

func (e *Engine) Install(ctx context.Context, manifest *Manifest) (*Summary, error) {
	start := time.Now()
	summary := &Summary{StartTime: start}

	seen := map[string]bool{}
	var packages []Package
	for _, p := range manifest.Packages {
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		packages = append(packages, p)
	}
	summary.Total = len(packages)

	for _, pkg := range packages {
		select {
		case <-ctx.Done():
			summary.EndTime = time.Now()
			summary.Duration = summary.EndTime.Sub(summary.StartTime)
			return summary, ctx.Err()
		default:
		}

		result := InstallResult{Package: pkg}
		pkgStart := time.Now()

		if e.cfg.SkipExisting && !e.cfg.DryRun {
			installed, err := e.backend.IsInstalled(ctx, pkg.ID)
			if err == nil && installed {
				result.Status = StatusSkipped
				result.DurationMs = time.Since(pkgStart).Milliseconds()
				summary.Skipped++
				summary.Results = append(summary.Results, result)
				e.fireProgress(result)
				continue
			}
		}

		if e.cfg.DryRun {
			result.Status = StatusSuccess
			result.DurationMs = time.Since(pkgStart).Milliseconds()
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			e.fireProgress(result)
			continue
		}

		err := e.installWithRetry(ctx, pkg.ID)
		result.DurationMs = time.Since(pkgStart).Milliseconds()
		if err != nil {
			if errors.Is(err, backend.ErrAlreadyInstalled) {
				result.Status = StatusSkipped
				summary.Skipped++
			} else {
				result.Status = StatusFailed
				result.Errors = []string{err.Error()}
				summary.Failed++
			}
		} else {
			result.Status = StatusSuccess
			summary.Succeeded++
		}

		summary.Results = append(summary.Results, result)
		e.fireProgress(result)
	}

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)
	return summary, nil
}

func (e *Engine) Uninstall(ctx context.Context, manifest *Manifest) (*Summary, error) {
	start := time.Now()
	summary := &Summary{StartTime: start, Total: len(manifest.Packages)}

	for _, pkg := range manifest.Packages {
		select {
		case <-ctx.Done():
			summary.EndTime = time.Now()
			summary.Duration = summary.EndTime.Sub(summary.StartTime)
			return summary, ctx.Err()
		default:
		}

		result := InstallResult{Package: pkg}
		pkgStart := time.Now()

		if e.cfg.DryRun {
			result.Status = StatusSuccess
			result.DurationMs = time.Since(pkgStart).Milliseconds()
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			e.fireProgress(result)
			continue
		}

		err := e.uninstallWithRetry(ctx, pkg.ID)
		result.DurationMs = time.Since(pkgStart).Milliseconds()
		if err != nil {
			result.Status = StatusFailed
			result.Errors = []string{err.Error()}
			summary.Failed++
		} else {
			result.Status = StatusSuccess
			summary.Succeeded++
		}

		summary.Results = append(summary.Results, result)
		e.fireProgress(result)
	}

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)
	return summary, nil
}

func (e *Engine) CheckStatus(ctx context.Context, manifest *Manifest) ([]PackageStatus, error) {
	installed, err := e.backend.ListInstalled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installed: %w", err)
	}

	installedSet := make(map[string]bool, len(installed))
	for _, id := range installed {
		installedSet[id] = true
	}

	results := make([]PackageStatus, len(manifest.Packages))
	for i, pkg := range manifest.Packages {
		results[i] = PackageStatus{
			Package:   pkg,
			Installed: installedSet[pkg.ID],
		}
	}
	return results, nil
}

func (e *Engine) installWithRetry(ctx context.Context, id string) error {
	return retry(ctx, e.cfg.RetryCount, e.cfg.RetryDelay, func() error {
		_, err := e.backend.Install(ctx, id, backend.InstallOptions{
			Proxy: e.cfg.Proxy,
		})
		if err != nil {
			if e.backend.IsPermanent(err) {
				return stop(err)
			}
			return err
		}
		return nil
	})
}

func (e *Engine) uninstallWithRetry(ctx context.Context, id string) error {
	return retry(ctx, e.cfg.RetryCount, e.cfg.RetryDelay, func() error {
		_, err := e.backend.Uninstall(ctx, id, backend.InstallOptions{
			DryRun: e.cfg.DryRun,
		})
		if err != nil {
			if e.backend.IsPermanent(err) {
				return stop(err)
			}
			return err
		}
		return nil
	})
}

func (e *Engine) fireProgress(r InstallResult) {
	if e.progressFn != nil {
		e.progressFn(r)
	}
}

type stopErr struct {
	err error
}

func (s *stopErr) Error() string { return s.err.Error() }
func (s *stopErr) Unwrap() error { return s.err }

func stop(err error) error { return &stopErr{err: err} }

func isStop(err error) bool {
	var se *stopErr
	return errors.As(err, &se)
}

func retry(ctx context.Context, maxRetries int, delay time.Duration, fn func() error) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		err := fn()
		if err == nil {
			return nil
		}
		if isStop(err) {
			var se *stopErr
			errors.As(err, &se)
			return se.err
		}
		lastErr = err
	}
	return lastErr
}


