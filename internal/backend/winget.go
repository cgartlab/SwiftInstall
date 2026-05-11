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
type WingetBackend struct {
	execPath string
	ready    bool
}

// NewWingetBackend creates a new winget backend instance.
// Call Detect() before using any other methods.
func NewWingetBackend() *WingetBackend {
	return &WingetBackend{}
}

func (w *WingetBackend) Name() string { return "winget" }

// Detect locates the winget executable and verifies it's usable.
func (w *WingetBackend) Detect() error {
	path, err := exec.LookPath("winget.exe")
	if err != nil {
		// Try common locations as fallback
		path, err = exec.LookPath("winget")
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

	cmd := exec.CommandContext(ctx, w.execPath,
		"list", "--id", id, "--exact", "--accept-source-agreements")
	out, err := cmd.Output()
	if err != nil {
		// winget returns non-zero exit code when package is not found
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("check installed status: %w", err)
	}

	// If output contains the package ID, it's installed
	return strings.Contains(string(out), id), nil
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
	start := time.Now()
	cmd := exec.CommandContext(ctx, w.execPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	dur := time.Since(start).Milliseconds()

	out := &Output{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: dur,
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
