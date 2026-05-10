package engine

import (
	"context"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusSkipped Status = "skipped"
	StatusFailed  Status = "failed"
)

type InstallResult struct {
	Package    manifest.Package `json:"package"`
	Status     Status           `json:"status"`
	Errors     []string         `json:"errors,omitempty"`
	DurationMs int64            `json:"duration_ms,omitempty"`
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

type InstallOptions struct {
	DryRun       bool
	SkipExisting bool
	Proxy        string
}

type Engine struct {
	backend    backend.Backend
	opts       InstallOptions
	progressFn func(InstallResult)
}

func New(be backend.Backend, opts InstallOptions) *Engine {
	return &Engine{
		backend: be,
		opts:    opts,
	}
}

func (e *Engine) SetProgressHook(fn func(InstallResult)) {
	e.progressFn = fn
}

func (e *Engine) Install(ctx context.Context, m *manifest.Manifest) (*Summary, error) {
	start := time.Now()
	summary := &Summary{StartTime: start}

	seen := map[string]bool{}
	var packages []manifest.Package
	for _, p := range m.Packages {
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

		if e.opts.SkipExisting && !e.opts.DryRun {
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

		if e.opts.DryRun {
			result.Status = StatusSuccess
			result.DurationMs = time.Since(pkgStart).Milliseconds()
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			e.fireProgress(result)
			continue
		}

		_, err := e.backend.Install(ctx, pkg.ID, backend.InstallOptions{
			Proxy: e.opts.Proxy,
		})
		result.DurationMs = time.Since(pkgStart).Milliseconds()
		if err != nil {
			result.Status = StatusFailed
			result.Errors = []string{err.Error()}
			if pkg.Optional {
				summary.Skipped++
			} else {
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

func (e *Engine) fireProgress(r InstallResult) {
	if e.progressFn != nil {
		e.progressFn(r)
	}
}
