package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NpmBackend installs global npm packages via `npm install -g`.
type NpmBackend struct {
	execPath string
	ready    bool
}

func NewNpmBackend() *NpmBackend { return &NpmBackend{} }

func (n *NpmBackend) Name() string { return "npm" }

func (n *NpmBackend) Detect() error {
	p, err := exec.LookPath("npm")
	if err != nil {
		return fmt.Errorf("未找到 npm。请先安装 Node.js（如 `brew install node`）")
	}
	n.execPath = p
	n.ready = true
	return nil
}

// IsInstalled checks `npm ls -g <pkg>`; a non-zero exit means not installed.
func (n *NpmBackend) IsInstalled(ctx context.Context, id string) (bool, error) {
	if !n.ready {
		return false, errors.New("npm backend not initialized: call Detect() first")
	}
	cmd := exec.CommandContext(ctx, n.execPath, "ls", "-g", "--depth=0", "--json", id)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return false, nil // not installed / npm errors → treat as not installed
	}
	var res struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return false, nil
	}
	_, ok := res.Dependencies[id]
	return ok, nil
}

// InstalledSet returns all globally-installed npm package names via one call.
func (n *NpmBackend) InstalledSet(ctx context.Context) (map[string]bool, error) {
	if !n.ready {
		return nil, errors.New("npm backend not initialized")
	}
	out, err := exec.CommandContext(ctx, n.execPath, "ls", "-g", "--depth=0", "--json").Output()
	if err != nil {
		return map[string]bool{}, nil
	}
	var res struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("parse npm ls: %w", err)
	}
	set := make(map[string]bool, len(res.Dependencies))
	for name := range res.Dependencies {
		set[name] = true
	}
	return set, nil
}

func (n *NpmBackend) Install(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	if !n.ready {
		return nil, errors.New("npm backend not initialized")
	}
	return n.run(ctx, "install", "-g", id)
}

func (n *NpmBackend) Uninstall(ctx context.Context, id string) (*Output, error) {
	if !n.ready {
		return nil, errors.New("npm backend not initialized")
	}
	return n.run(ctx, "uninstall", "-g", id)
}

func (n *NpmBackend) run(ctx context.Context, args ...string) (*Output, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, n.execPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := &Output{Stdout: stdout.String(), Stderr: stderr.String(), DurationMs: time.Since(start).Milliseconds()}
	if err == nil {
		out.ExitCode = 0
		return out, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		out.ExitCode = exitErr.ExitCode()
		return out, fmt.Errorf("npm exited with code %d: %s", out.ExitCode, strings.TrimSpace(stderr.String()))
	}
	return out, fmt.Errorf("execute npm: %w", err)
}
