package backend

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type WingetBackend struct {
	execPath string
}

func NewWingetBackend() *WingetBackend {
	return &WingetBackend{}
}

func (w *WingetBackend) Name() string { return "winget" }

func (w *WingetBackend) Detect() error {
	path, err := exec.LookPath("winget.exe")
	if err != nil {
		return fmt.Errorf("winget not found: %w", err)
	}
	w.execPath = path
	return nil
}

func (w *WingetBackend) IsInstalled(ctx context.Context, id string) (bool, error) {
	cmd := exec.CommandContext(ctx, w.execPath, "list", "--id", id, "--exact", "--accept-source-agreements")
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	output := strings.TrimSpace(string(out))
	return strings.Contains(output, id), nil
}

func (w *WingetBackend) Install(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	args := []string{"install", "--id", id, "--exact", "--silent",
		"--accept-package-agreements", "--accept-source-agreements"}
	if opts.Proxy != "" {
		args = append(args, "--proxy", opts.Proxy)
	}

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
	}

	return out, err
}
