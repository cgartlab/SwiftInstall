package backend

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// WingetBackend implements Backend using Windows Package Manager (winget).
//
// lookPath and run are injectable so the Windows line can unit-test the winget
// backend on any host (including Linux CI runners) without winget installed.
// Production behaviour is unchanged: NewWingetBackend wires the real
// implementations.
type WingetBackend struct {
	execPath string
	ready    bool
	lookPath func(file string) (string, error)
	run      func(ctx context.Context, args ...string) (*Output, error)
}

// NewWingetBackend creates a new winget backend instance.
// Call Detect() before using any other methods.
func NewWingetBackend() *WingetBackend {
	w := &WingetBackend{lookPath: exec.LookPath}
	w.run = w.execCommand
	return w
}

func (w *WingetBackend) Name() string { return "winget" }

// Detect locates the winget executable and verifies it's usable.
func (w *WingetBackend) Detect() error {
	path, err := w.lookPath("winget.exe")
	if err != nil {
		path, err = w.lookPath("winget")
		if err != nil {
			return fmt.Errorf("winget is not installed or not in PATH.\n" +
				"Please install it from: https://aka.ms/getwinget")
		}
	}
	w.execPath = path
	w.ready = true
	return nil
}

func (w *WingetBackend) notReady() error {
	return fmt.Errorf("winget backend not initialized: call Detect() first")
}

// IsInstalled checks if a package is installed using `winget list --id <id> --exact`.
func (w *WingetBackend) IsInstalled(ctx context.Context, id string) (bool, error) {
	if !w.ready {
		return false, w.notReady()
	}

	out, err := w.run(ctx, "list", "--id", id, "--exact", "--accept-source-agreements")
	if err != nil {
		// winget exits non-zero when the package is not found; that is a
		// negative answer, not a failure.
		if out != nil && out.ExitCode != 0 {
			return false, nil
		}
		return false, fmt.Errorf("check installed status: %w", err)
	}

	return strings.Contains(out.Stdout, id), nil
}

// Install installs a package using `winget install --id <id> --exact --silent`.
func (w *WingetBackend) Install(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	if !w.ready {
		return nil, w.notReady()
	}

	args := []string{
		"install", "--id", id, "--exact", "--silent",
		"--accept-package-agreements", "--accept-source-agreements",
	}
	if opts.Proxy != "" {
		args = append(args, "--proxy", opts.Proxy)
	}

	return w.run(ctx, args...)
}

// Uninstall removes a package using `winget uninstall --id <id> --exact --silent`.
func (w *WingetBackend) Uninstall(ctx context.Context, id string) (*Output, error) {
	if !w.ready {
		return nil, w.notReady()
	}

	args := []string{
		"uninstall", "--id", id, "--exact", "--silent",
	}

	return w.run(ctx, args...)
}

// execCommand is the production implementation of run.
func (w *WingetBackend) execCommand(ctx context.Context, args ...string) (*Output, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, w.execPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := &Output{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: time.Since(start).Milliseconds(),
	}

	if err == nil {
		out.ExitCode = 0
		return out, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		out.ExitCode = exitErr.ExitCode()
		return out, fmt.Errorf("winget exited with code %d: %s",
			out.ExitCode, strings.TrimSpace(stderr.String()))
	}

	return out, fmt.Errorf("execute winget: %w", err)
}
