package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// WingetBackend implements Backend using Windows Package Manager (winget).
type WingetBackend struct {
	execPath string
	ready    bool

	// runFn executes a winget invocation. It is injectable so unit tests never
	// touch the real winget binary. NewWingetBackend wires execWinget; Detect
	// and every command route through runWinget, which falls back to the
	// package-level default when runFn is nil.
	runFn wingetRunner
	// lookPath locates the winget executable. Injectable for the same reason;
	// Detect falls back to exec.LookPath when nil.
	lookPath func(string) (string, error)
}

// wingetInvocation is a single, fully-resolved winget command line.
type wingetInvocation struct {
	ExecPath string
	Args     []string
}

// wingetRunner runs a wingetInvocation and captures its result. Tests replace
// this via WingetBackend.run.
type wingetRunner func(ctx context.Context, inv wingetInvocation) (*Output, error)

// wingetExitError is returned when winget runs to completion but exits
// non-zero. A non-zero exit is the normal "package not found" signal for
// `winget list`, so callers such as IsInstalled can detect it and treat it as
// "not installed" rather than a genuine failure.
type wingetExitError struct {
	ExitCode int
	Stderr   string
}

func (e *wingetExitError) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		return fmt.Sprintf("winget exited with code %d", e.ExitCode)
	}
	return fmt.Sprintf("winget exited with code %d: %s", e.ExitCode, msg)
}

// NewWingetBackend creates a new winget backend instance.
// Call Detect() before using any other methods.
func NewWingetBackend() *WingetBackend {
	return &WingetBackend{
		runFn:    execWinget,
		lookPath: exec.LookPath,
	}
}

func (w *WingetBackend) Name() string { return "winget" }

// Detect locates the winget executable and verifies it's usable.
func (w *WingetBackend) Detect() error {
	lookPath := w.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	path, err := lookPath("winget.exe")
	if err != nil {
		// Try common locations as fallback
		path, err = lookPath("winget")
		if err != nil {
			return fmt.Errorf("winget is not installed or not in PATH.\n" +
				"Please install it from: https://aka.ms/getwinget")
		}
	}
	w.execPath = path
	w.ready = true
	return nil
}

// IsInstalled checks if a package is installed using `winget list --id <id> --exact`.
func (w *WingetBackend) IsInstalled(ctx context.Context, id string) (bool, error) {
	if !w.ready {
		return false, fmt.Errorf("winget backend not initialized: call Detect() first")
	}

	out, err := w.runWinget(ctx,
		"list", "--id", id, "--exact", "--accept-source-agreements")
	if err != nil {
		// winget returns non-zero exit code when package is not found
		var exitErr *wingetExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, fmt.Errorf("check installed status: %w", err)
	}

	// If output contains the package ID, it's installed
	return strings.Contains(out.Stdout, id), nil
}

// Install installs a package using `winget install --id <id> --exact --silent`.
func (w *WingetBackend) Install(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	if !w.ready {
		return nil, fmt.Errorf("winget backend not initialized: call Detect() first")
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
		return nil, fmt.Errorf("winget backend not initialized: call Detect() first")
	}

	args := []string{
		"uninstall", "--id", id, "--exact", "--silent",
	}

	return w.run(ctx, args...)
}

// run executes a winget command and captures output.
func (w *WingetBackend) run(ctx context.Context, args ...string) (*Output, error) {
	return w.runWinget(ctx, args...)
}

// runWinget routes a command through the injectable runner.
func (w *WingetBackend) runWinget(ctx context.Context, args ...string) (*Output, error) {
	run := w.runFn
	if run == nil {
		run = execWinget
	}
	return run(ctx, wingetInvocation{ExecPath: w.execPath, Args: args})
}

// execWinget is the production runner: it executes winget with the given
// invocation, mirroring execBrew. A non-zero exit is surfaced as a
// *wingetExitError so IsInstalled can distinguish "not installed" from a
// genuine failure.
func execWinget(ctx context.Context, inv wingetInvocation) (*Output, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, inv.ExecPath, inv.Args...)

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

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out.ExitCode = exitErr.ExitCode()
		return out, &wingetExitError{ExitCode: out.ExitCode, Stderr: stderr.String()}
	}

	return out, fmt.Errorf("execute winget: %w", err)
}
