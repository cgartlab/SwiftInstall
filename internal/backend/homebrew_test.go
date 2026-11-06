package backend

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// fakeCall records one invocation captured by a fake runner.
type fakeCall = brewInvocation

// scriptedRunner returns a brewRunner that records every invocation and lets a
// test decide each result by call index. It never touches the real Homebrew
// binary.
func scriptedRunner(calls *[]fakeCall, fn func(idx int, inv brewInvocation) (*Output, error)) brewRunner {
	return func(_ context.Context, inv brewInvocation) (*Output, error) {
		*calls = append(*calls, inv)
		return fn(len(*calls)-1, inv)
	}
}

// newTestBackend returns a ready HomebrewBackend wired to a fake runner.
func newTestBackend(run brewRunner) *HomebrewBackend {
	h := NewHomebrewBackend()
	h.execPath = "/opt/homebrew/bin/brew"
	h.ready = true
	h.run = run
	return h
}

func okOutput() *Output { return &Output{ExitCode: 0} }

func exitError(code int, stderr string) (*Output, error) {
	return &Output{ExitCode: code, Stderr: stderr},
		&brewExitError{ExitCode: code, Stderr: stderr}
}

func hasEnv(env []string, want string) bool {
	for _, kv := range env {
		if kv == want {
			return true
		}
	}
	return false
}

func envValue(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return strings.TrimPrefix(kv, prefix), true
		}
	}
	return "", false
}

func TestHomebrewName(t *testing.T) {
	if got := NewHomebrewBackend().Name(); got != "homebrew" {
		t.Fatalf("Name() = %q, want %q", got, "homebrew")
	}
}

func TestHomebrewDetectSuccessFromLookPath(t *testing.T) {
	h := NewHomebrewBackend()
	h.lookPath = func(string) (string, error) { return "/opt/homebrew/bin/brew", nil }
	h.stat = func(string) (os.FileInfo, error) { return nil, nil }

	if err := h.Detect(); err != nil {
		t.Fatalf("Detect() error = %v, want nil", err)
	}
	if !h.ready {
		t.Fatal("ready = false, want true")
	}
	if h.execPath != "/opt/homebrew/bin/brew" {
		t.Fatalf("execPath = %q, want /opt/homebrew/bin/brew", h.execPath)
	}
}

func TestHomebrewDetectFallsBackToKnownPrefix(t *testing.T) {
	h := NewHomebrewBackend()
	h.lookPath = func(string) (string, error) { return "", errors.New("not found in PATH") }
	// Only the Intel prefix "exists".
	h.stat = func(p string) (os.FileInfo, error) {
		if p == "/usr/local/bin/brew" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}

	if err := h.Detect(); err != nil {
		t.Fatalf("Detect() error = %v, want nil", err)
	}
	if h.execPath != "/usr/local/bin/brew" {
		t.Fatalf("execPath = %q, want /usr/local/bin/brew", h.execPath)
	}
}

func TestHomebrewDetectFailure(t *testing.T) {
	h := NewHomebrewBackend()
	h.lookPath = func(string) (string, error) { return "", errors.New("not found in PATH") }
	h.stat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }

	err := h.Detect()
	if err == nil {
		t.Fatal("Detect() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "brew.sh") {
		t.Fatalf("Detect() error = %q, want it to mention brew.sh", err.Error())
	}
	if h.ready {
		t.Fatal("ready = true after failed Detect(), want false")
	}
}

func TestHomebrewNotReady(t *testing.T) {
	h := NewHomebrewBackend() // ready == false
	if _, err := h.Install(context.Background(), "git", InstallOptions{}); err == nil {
		t.Fatal("Install() before Detect() = nil error, want error")
	}
	if _, err := h.IsInstalled(context.Background(), "git"); err == nil {
		t.Fatal("IsInstalled() before Detect() = nil error, want error")
	}
	if _, err := h.Uninstall(context.Background(), "git"); err == nil {
		t.Fatal("Uninstall() before Detect() = nil error, want error")
	}
}

func TestHomebrewInstallFormulaArgs(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return okOutput(), nil
	}))

	if _, err := h.Install(context.Background(), "git", InstallOptions{Kind: "formula"}); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	got := strings.Join(calls[0].Args, " ")
	if got != "install git" {
		t.Fatalf("args = %q, want %q", got, "install git")
	}
	if !hasEnv(calls[0].Env, "HOMEBREW_NO_AUTO_UPDATE=1") {
		t.Fatalf("env missing HOMEBREW_NO_AUTO_UPDATE=1: %v", calls[0].Env)
	}
	if !hasEnv(calls[0].Env, "HOMEBREW_NO_ENV_HINTS=1") {
		t.Fatalf("env missing HOMEBREW_NO_ENV_HINTS=1")
	}
	if !hasEnv(calls[0].Env, "HOMEBREW_NO_INSTALL_CLEANUP=1") {
		t.Fatalf("env missing HOMEBREW_NO_INSTALL_CLEANUP=1")
	}
}

func TestHomebrewInstallCaskArgs(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return okOutput(), nil
	}))

	if _, err := h.Install(context.Background(), "visual-studio-code", InstallOptions{Kind: "cask"}); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if got := strings.Join(calls[0].Args, " "); got != "install --cask visual-studio-code" {
		t.Fatalf("args = %q, want %q", got, "install --cask visual-studio-code")
	}
}

func TestHomebrewInstallProxyEnv(t *testing.T) {
	const proxy = "http://127.0.0.1:7890"
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return okOutput(), nil
	}))

	if _, err := h.Install(context.Background(), "git", InstallOptions{Kind: "formula", Proxy: proxy}); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	for _, key := range []string{"ALL_PROXY", "HTTPS_PROXY", "HTTP_PROXY"} {
		v, ok := envValue(calls[0].Env, key)
		if !ok {
			t.Fatalf("env missing %s", key)
		}
		if v != proxy {
			t.Fatalf("%s = %q, want %q", key, v, proxy)
		}
	}
}

func TestHomebrewInstallAutoFallbackFormulaToCask(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(idx int, inv brewInvocation) (*Output, error) {
		if idx == 0 {
			return exitError(1, `Error: No available formula with the name "foo".`)
		}
		return okOutput(), nil
	}))

	out, err := h.Install(context.Background(), "foo", InstallOptions{Kind: ""})
	if err != nil {
		t.Fatalf("Install() error = %v, want nil", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", out.ExitCode)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	if got := strings.Join(calls[0].Args, " "); got != "install foo" {
		t.Fatalf("call[0] args = %q, want %q", got, "install foo")
	}
	if got := strings.Join(calls[1].Args, " "); got != "install --cask foo" {
		t.Fatalf("call[1] args = %q, want %q", got, "install --cask foo")
	}
}

func TestHomebrewInstallAutoNoFallbackOnOtherError(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return exitError(1, "Error: network failure")
	}))

	if _, err := h.Install(context.Background(), "foo", InstallOptions{Kind: ""}); err == nil {
		t.Fatal("Install() error = nil, want non-nil")
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1 (no cask retry for non-formula errors)", len(calls))
	}
}

func TestHomebrewIsInstalledFormulaTrue(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return okOutput(), nil
	}))

	installed, err := h.IsInstalled(context.Background(), "git")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}
	if !installed {
		t.Fatal("IsInstalled() = false, want true")
	}
	if len(calls) != 1 || strings.Join(calls[0].Args, " ") != "list --formula git" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestHomebrewIsInstalledCaskTrue(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(idx int, inv brewInvocation) (*Output, error) {
		if strings.Contains(strings.Join(inv.Args, " "), "--cask") {
			return okOutput(), nil
		}
		return exitError(1, "Error: No such keg: git")
	}))

	installed, err := h.IsInstalled(context.Background(), "visual-studio-code")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}
	if !installed {
		t.Fatal("IsInstalled() = false, want true")
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2 (formula miss then cask hit)", len(calls))
	}
}

func TestHomebrewIsInstalledFalse(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return exitError(1, "Error: No such keg")
	}))

	installed, err := h.IsInstalled(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v, want nil (not-installed is not an error)", err)
	}
	if installed {
		t.Fatal("IsInstalled() = true, want false")
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2 (formula then cask)", len(calls))
	}
}

func TestHomebrewIsInstalledGenuineError(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return nil, errors.New("fork/exec: permission denied")
	}))

	if _, err := h.IsInstalled(context.Background(), "git"); err == nil {
		t.Fatal("IsInstalled() error = nil, want genuine error surfaced")
	}
}

func TestHomebrewUninstallCask(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(idx int, inv brewInvocation) (*Output, error) {
		// First call: list --cask → installed. Second: uninstall succeeds.
		return okOutput(), nil
	}))

	if _, err := h.Uninstall(context.Background(), "visual-studio-code"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	if got := strings.Join(calls[0].Args, " "); got != "list --cask visual-studio-code" {
		t.Fatalf("call[0] args = %q", got)
	}
	if got := strings.Join(calls[1].Args, " "); got != "uninstall --cask visual-studio-code" {
		t.Fatalf("call[1] args = %q, want %q", got, "uninstall --cask visual-studio-code")
	}
}

func TestHomebrewUninstallFormula(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(idx int, inv brewInvocation) (*Output, error) {
		if idx == 0 { // list --cask → not installed
			return exitError(1, "Error: Cask 'git' is unavailable")
		}
		return okOutput(), nil
	}))

	if _, err := h.Uninstall(context.Background(), "git"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	if got := strings.Join(calls[1].Args, " "); got != "uninstall git" {
		t.Fatalf("call[1] args = %q, want %q", got, "uninstall git")
	}
}

func TestHomebrewInstallUnknownKind(t *testing.T) {
	var calls []fakeCall
	h := newTestBackend(scriptedRunner(&calls, func(int, brewInvocation) (*Output, error) {
		return okOutput(), nil
	}))

	if _, err := h.Install(context.Background(), "git", InstallOptions{Kind: "bogus"}); err == nil {
		t.Fatal("Install() with unknown kind = nil error, want error")
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(calls))
	}
}

func TestIsFormulaUnavailable(t *testing.T) {
	cases := []struct {
		name string
		out  *Output
		err  error
		want bool
	}{
		{"no available formula", &Output{ExitCode: 1, Stderr: `Error: No available formula with the name "foo".`}, &brewExitError{ExitCode: 1}, true},
		{"no formulae found", &Output{ExitCode: 1, Stderr: "Error: No formulae found in taps."}, &brewExitError{ExitCode: 1}, true},
		{"neither exists", &Output{ExitCode: 1, Stderr: "Error: No formulae or casks found for foo."}, &brewExitError{ExitCode: 1}, false},
		{"network", &Output{ExitCode: 1, Stderr: "Error: network timeout"}, &brewExitError{ExitCode: 1}, false},
		{"nil error", nil, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isFormulaUnavailable(c.out, c.err); got != c.want {
				t.Fatalf("isFormulaUnavailable() = %v, want %v", got, c.want)
			}
		})
	}
}
