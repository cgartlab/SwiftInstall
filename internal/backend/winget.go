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
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (w *WingetBackend) Install(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	if opts.DryRun {
		return &Output{}, nil
	}

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
		if exitErr.ExitCode() == 0x8A150011 {
			return out, ErrAlreadyInstalled
		}
		return out, FormatWingetError(exitErr.ExitCode())
	}

	return out, err
}

func (w *WingetBackend) Uninstall(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	if opts.DryRun {
		return &Output{}, nil
	}

	args := []string{"uninstall", "--id", id, "--exact", "--silent"}

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
		return out, FormatWingetError(exitErr.ExitCode())
	}

	return out, err
}

func (w *WingetBackend) ListInstalled(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, w.execPath, "list", "--accept-source-agreements")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("winget list: %w", err)
	}
	return parseWingetListOutput(string(out)), nil
}

func (w *WingetBackend) IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrAlreadyInstalled) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "0x8A150019") ||
		strings.Contains(msg, "0x8A15003F") ||
		strings.Contains(msg, "No applicable package") ||
		strings.Contains(msg, "not found")
}

func parseWingetListOutput(output string) []string {
	lines := strings.Split(output, "\n")
	if len(lines) < 3 {
		return nil
	}

	lines = lines[2:]

	var ids []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			id := parts[1]
			if id != "" && id != "Id" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}
