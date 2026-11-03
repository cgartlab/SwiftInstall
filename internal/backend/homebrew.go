package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HomebrewBackend implements Backend using Homebrew on macOS.
//
// Homebrew ships two channels with distinct namespaces: "formula" (CLI tools,
// libraries, languages) and "cask" (primarily GUI apps and fonts). Installs and
// uninstalls must therefore know which channel to use. When the caller does not
// specify one (InstallOptions.Kind == ""), the backend probes both channels and
// falls back from formula to cask for installs.
type HomebrewBackend struct {
	execPath string
	ready    bool

	// run executes a brew invocation. It is injectable so unit tests never
	// touch the real Homebrew binary. NewHomebrewBackend wires execBrew.
	run brewRunner
	// lookPath / stat locate the brew executable. Injectable for the same
	// reason; Detect falls back to the package-level defaults when nil.
	lookPath func(string) (string, error)
	stat     func(string) (os.FileInfo, error)
}

// brewInvocation is a single, fully-resolved brew command line.
type brewInvocation struct {
	ExecPath string
	Env      []string
	Args     []string
}

// brewRunner runs a brewInvocation and captures its result. Tests replace this
// via HomebrewBackend.run.
type brewRunner func(ctx context.Context, inv brewInvocation) (*Output, error)

// brewExitError is returned when brew runs to completion but exits non-zero.
// A non-zero exit is the normal "package not found" signal for `brew list`, so
// callers such as IsInstalled can detect it and treat it as "not installed"
// rather than a genuine failure.
type brewExitError struct {
	ExitCode int
	Stderr   string
}

func (e *brewExitError) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		return fmt.Sprintf("brew exited with code %d", e.ExitCode)
	}
	return fmt.Sprintf("brew exited with code %d: %s", e.ExitCode, msg)
}

// homebrewFallbackPaths are the conventional install locations checked when
// `brew` is not on PATH (Apple Silicon and Intel prefixes respectively).
var homebrewFallbackPaths = []string{
	"/opt/homebrew/bin/brew",
	"/usr/local/bin/brew",
}

// errHomebrewNotReady is returned when a method is called before Detect.
var errHomebrewNotReady = errors.New("homebrew backend not initialized: call Detect() first")

// NewHomebrewBackend creates a new Homebrew backend instance.
// Call Detect() before using any other methods.
func NewHomebrewBackend() *HomebrewBackend {
	return &HomebrewBackend{
		run:      execBrew,
		lookPath: exec.LookPath,
		stat:     os.Stat,
	}
}

func (h *HomebrewBackend) Name() string { return "homebrew" }

// Detect locates the brew executable. It first asks PATH, then falls back to the
// standard Apple Silicon / Intel prefixes. Returns a user-friendly Chinese error
// pointing at https://brew.sh when Homebrew cannot be found.
func (h *HomebrewBackend) Detect() error {
	lookPath := h.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	stat := h.stat
	if stat == nil {
		stat = os.Stat
	}

	var candidates []string
	if p, err := lookPath("brew"); err == nil && p != "" {
		candidates = append(candidates, p)
	}
	candidates = append(candidates, homebrewFallbackPaths...)

	for _, p := range candidates {
		if _, err := stat(p); err == nil {
			h.execPath = p
			h.ready = true
			return nil
		}
	}

	return fmt.Errorf("未找到 Homebrew (brew)。请先安装 Homebrew：https://brew.sh ，" +
		"安装后确保 brew 位于 PATH，或存在于 /opt/homebrew/bin/brew 或 /usr/local/bin/brew")
}

// IsInstalled reports whether id is installed as either a formula or a cask.
// A non-zero exit from `brew list` means "not installed" (not an error); only
// genuine failures (e.g. brew cannot be executed) surface as errors.
func (h *HomebrewBackend) IsInstalled(ctx context.Context, id string) (bool, error) {
	if !h.ready {
		return false, errHomebrewNotReady
	}

	formula, err := h.installedInChannel(ctx, "formula", id)
	if err != nil {
		return false, err
	}
	if formula {
		return true, nil
	}
	return h.installedInChannel(ctx, "cask", id)
}

// installedInChannel runs `brew list --<channel> <id>`. Exit code 0 means the
// package is present in that channel; a non-zero exit means it is absent.
func (h *HomebrewBackend) installedInChannel(ctx context.Context, channel, id string) (bool, error) {
	out, err := h.runBrew(ctx, h.env(""), "list", "--"+channel, id)
	if err != nil {
		var exitErr *brewExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, fmt.Errorf("check %s %q installed: %w", channel, id, err)
	}
	_ = out
	return true, nil
}

// InstalledSet reports every installed formula name and cask token in one pass,
// using `brew list --formula -1` and `brew list --cask -1`. This powers a fast
// (sub-second) status load in the GUI instead of two `brew` calls per package.
// A non-zero exit from either listing means "none installed in that channel",
// not an error.
func (h *HomebrewBackend) InstalledSet(ctx context.Context) (map[string]bool, error) {
	if !h.ready {
		return nil, errHomebrewNotReady
	}

	set := make(map[string]bool)
	for _, channel := range []string{"formula", "cask"} {
		out, err := h.runBrew(ctx, h.env(""), "list", "--"+channel, "-1")
		if err != nil {
			var exitErr *brewExitError
			if errors.As(err, &exitErr) {
				continue // nothing installed in this channel
			}
			return nil, fmt.Errorf("list installed %s packages: %w", channel, err)
		}
		for _, line := range strings.Split(out.Stdout, "\n") {
			if name := strings.TrimSpace(line); name != "" {
				set[name] = true
			}
		}
	}
	return set, nil
}

// Install installs id honouring opts.Kind:
//   - "cask"    → brew install --cask <id>
//   - "formula" → brew install <id>
//   - ""        → try the formula channel first; if Homebrew reports that no
//     such formula exists, retry on the cask channel (auto-detect).
func (h *HomebrewBackend) Install(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	if !h.ready {
		return nil, errHomebrewNotReady
	}
	env := h.env(opts.Proxy)

	switch opts.Kind {
	case "cask":
		return h.runBrew(ctx, env, "install", "--cask", id)
	case "formula":
		return h.runBrew(ctx, env, "install", id)
	case "":
		out, err := h.runBrew(ctx, env, "install", id)
		if err == nil {
			return out, nil
		}
		if isFormulaUnavailable(out, err) {
			return h.runBrew(ctx, env, "install", "--cask", id)
		}
		return out, err
	default:
		return nil, fmt.Errorf("未知的 Homebrew kind %q（应为 \"formula\" 或 \"cask\"）", opts.Kind)
	}
}

// Uninstall removes id from whichever channel it is installed in. Casks are
// checked first; anything else falls back to the formula channel.
func (h *HomebrewBackend) Uninstall(ctx context.Context, id string) (*Output, error) {
	if !h.ready {
		return nil, errHomebrewNotReady
	}
	env := h.env("")

	cask, err := h.installedInChannel(ctx, "cask", id)
	if err != nil {
		return nil, err
	}
	if cask {
		return h.runBrew(ctx, env, "uninstall", "--cask", id)
	}
	return h.runBrew(ctx, env, "uninstall", id)
}

// env builds the process environment: the caller's environment plus the flags
// that keep brew non-interactive and quiet, and proxy variables when a proxy is
// configured.
func (h *HomebrewBackend) env(proxy string) []string {
	env := os.Environ()
	env = append(env,
		"HOMEBREW_NO_AUTO_UPDATE=1",
		"HOMEBREW_NO_ENV_HINTS=1",
		"HOMEBREW_NO_INSTALL_CLEANUP=1",
		"HOMEBREW_NO_ANALYTICS=1",
	)
	if proxy != "" {
		env = append(env,
			"ALL_PROXY="+proxy,
			"HTTPS_PROXY="+proxy,
			"HTTP_PROXY="+proxy,
			"all_proxy="+proxy,
			"https_proxy="+proxy,
			"http_proxy="+proxy,
		)
	}
	return env
}

// runBrew routes a command through the injectable runner.
func (h *HomebrewBackend) runBrew(ctx context.Context, env []string, args ...string) (*Output, error) {
	run := h.run
	if run == nil {
		run = execBrew
	}
	return run(ctx, brewInvocation{ExecPath: h.execPath, Env: env, Args: args})
}

// isFormulaUnavailable reports whether a failed `brew install <id>` failed
// specifically because the formula does not exist — the signal for the
// formula→cask auto-fallback.
func isFormulaUnavailable(out *Output, err error) bool {
	if err == nil {
		return false
	}
	var b strings.Builder
	if out != nil {
		b.WriteString(out.Stdout)
		b.WriteByte(' ')
		b.WriteString(out.Stderr)
	}
	b.WriteByte(' ')
	b.WriteString(err.Error())
	msg := strings.ToLower(b.String())

	for _, needle := range []string{
		"no available formula",
		"no formulae found",
		"no formula with",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// execBrew is the production runner: it executes brew with the given invocation.
func execBrew(ctx context.Context, inv brewInvocation) (*Output, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, inv.ExecPath, inv.Args...)
	if inv.Env != nil {
		cmd.Env = inv.Env
	}

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
		return out, &brewExitError{ExitCode: out.ExitCode, Stderr: stderr.String()}
	}

	return out, fmt.Errorf("execute brew: %w", err)
}
