package backend

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// wingetCall records one invocation captured by a fake winget runner.
type wingetCall = wingetInvocation

// scriptedWingetRunner returns a wingetRunner that records every invocation and
// lets a test decide each result by call index. It never touches the real
// winget binary, so the tests run on any OS (including the ubuntu CI job).
func scriptedWingetRunner(calls *[]wingetCall, fn func(idx int, inv wingetCall) (*Output, error)) wingetRunner {
	return func(_ context.Context, inv wingetCall) (*Output, error) {
		*calls = append(*calls, inv)
		return fn(len(*calls)-1, inv)
	}
}

// newTestWingetBackend returns a ready WingetBackend wired to a fake runner.
func newTestWingetBackend(run wingetRunner) *WingetBackend {
	w := NewWingetBackend()
	w.execPath = `C:\Users\test\AppData\Local\Microsoft\WindowsApps\winget.exe`
	w.ready = true
	w.runFn = run
	return w
}

// wingetOK is a successful (exit 0) winget result.
func wingetOK(stdout string) (*Output, error) {
	return &Output{ExitCode: 0, Stdout: stdout}, nil
}

// wingetExit is a completed-but-non-zero winget result.
func wingetExit(code int, stderr string) (*Output, error) {
	return &Output{ExitCode: code, Stderr: stderr},
		&wingetExitError{ExitCode: code, Stderr: stderr}
}

func TestWingetName(t *testing.T) {
	if got := NewWingetBackend().Name(); got != "winget" {
		t.Fatalf("Name() = %q, want %q", got, "winget")
	}
}

func TestWingetDetectSuccess(t *testing.T) {
	w := NewWingetBackend()
	w.lookPath = func(name string) (string, error) {
		if name == "winget.exe" {
			return `C:\WindowsApps\winget.exe`, nil
		}
		return "", errors.New("not found")
	}

	if err := w.Detect(); err != nil {
		t.Fatalf("Detect() error = %v, want nil", err)
	}
	if !w.ready {
		t.Fatal("ready = false, want true")
	}
	if w.execPath != `C:\WindowsApps\winget.exe` {
		t.Fatalf("execPath = %q, want C:\\WindowsApps\\winget.exe", w.execPath)
	}
}

// TestWingetDetectFallsBackToBareName covers the second LookPath("winget")
// probe used when "winget.exe" is not found.
func TestWingetDetectFallsBackToBareName(t *testing.T) {
	w := NewWingetBackend()
	w.lookPath = func(name string) (string, error) {
		if name == "winget" {
			return `/usr/local/bin/winget`, nil
		}
		return "", errors.New("not found")
	}

	if err := w.Detect(); err != nil {
		t.Fatalf("Detect() error = %v, want nil", err)
	}
	if w.execPath != `/usr/local/bin/winget` {
		t.Fatalf("execPath = %q, want /usr/local/bin/winget", w.execPath)
	}
}

func TestWingetDetectFailure(t *testing.T) {
	w := NewWingetBackend()
	w.lookPath = func(string) (string, error) { return "", errors.New("not found in PATH") }

	err := w.Detect()
	if err == nil {
		t.Fatal("Detect() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "aka.ms/getwinget") {
		t.Fatalf("Detect() error = %q, want it to mention aka.ms/getwinget", err.Error())
	}
	if w.ready {
		t.Fatal("ready = true after failed Detect(), want false")
	}
}

func TestWingetNotReady(t *testing.T) {
	w := NewWingetBackend() // ready == false
	if _, err := w.IsInstalled(context.Background(), "git"); err == nil {
		t.Fatal("IsInstalled() before Detect() = nil error, want error")
	}
	if _, err := w.Install(context.Background(), "git", InstallOptions{}); err == nil {
		t.Fatal("Install() before Detect() = nil error, want error")
	}
	if _, err := w.Uninstall(context.Background(), "git"); err == nil {
		t.Fatal("Uninstall() before Detect() = nil error, want error")
	}
}

func TestWingetIsInstalledTrue(t *testing.T) {
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return wingetOK("Git.Git Git 2.45.0\n")
	}))

	installed, err := w.IsInstalled(context.Background(), "Git.Git")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}
	if !installed {
		t.Fatal("IsInstalled() = false, want true")
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if got := strings.Join(calls[0].Args, " "); got != "list --id Git.Git --exact --accept-source-agreements" {
		t.Fatalf("args = %q", got)
	}
}

// TestWingetIsInstalledExitErrorIsNotInstalled asserts the important semantic:
// a non-zero exit from `winget list` means "not installed" and must NOT be
// surfaced as an error.
func TestWingetIsInstalledExitErrorIsNotInstalled(t *testing.T) {
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return wingetExit(0x8A15002B, "No installed package found matching input criteria.")
	}))

	installed, err := w.IsInstalled(context.Background(), "Git.Git")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v, want nil (not-installed is not an error)", err)
	}
	if installed {
		t.Fatal("IsInstalled() = true, want false")
	}
}

func TestWingetIsInstalledOutputWithoutID(t *testing.T) {
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		// Exit 0 but the id never appears in the listing → not installed.
		return wingetOK("Name    Id      Version\n----    --      -------\n")
	}))

	installed, err := w.IsInstalled(context.Background(), "Git.Git")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}
	if installed {
		t.Fatal("IsInstalled() = true, want false")
	}
}

func TestWingetIsInstalledGenuineError(t *testing.T) {
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return nil, errors.New("fork/exec: permission denied")
	}))

	if _, err := w.IsInstalled(context.Background(), "Git.Git"); err == nil {
		t.Fatal("IsInstalled() error = nil, want genuine error surfaced")
	}
}

// TestWingetIsInstalledSubstringFalsePositive documents a KNOWN weakness of the
// `strings.Contains` judgment in IsInstalled. Because the whole listing is
// substring-matched, an id like "git" can be reported installed from a row that
// merely contains it as a substring (e.g. "git-lfs"), even though `--exact` is
// passed. We record the behaviour here WITHOUT changing IsInstalled's
// semantics, so the risk is visible and a future stricter match is a conscious
// decision.
func TestWingetIsInstalledSubstringFalsePositive(t *testing.T) {
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		// Only "git-lfs" is present; a bare "git" is not. Contains("git") still matches.
		return wingetOK("Git.LFS  git-lfs  3.5.0\n")
	}))

	installed, err := w.IsInstalled(context.Background(), "git")
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}
	if !installed {
		t.Fatal("IsInstalled() = false; documents substring false positive (id \"git\" matched \"git-lfs\")")
	}
}

func TestWingetInstallArgs(t *testing.T) {
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return wingetOK("")
	}))

	if _, err := w.Install(context.Background(), "Git.Git", InstallOptions{}); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	want := "install --id Git.Git --exact --silent --accept-package-agreements --accept-source-agreements"
	if got := strings.Join(calls[0].Args, " "); got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

// TestWingetInstallProxyArg asserts that a non-empty opts.Proxy appends
// `--proxy <proxy>` to the install command line.
func TestWingetInstallProxyArg(t *testing.T) {
	const proxy = "http://127.0.0.1:7890"
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return wingetOK("")
	}))

	if _, err := w.Install(context.Background(), "Git.Git", InstallOptions{Proxy: proxy}); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	want := "install --id Git.Git --exact --silent --accept-package-agreements --accept-source-agreements --proxy " + proxy
	if got := strings.Join(calls[0].Args, " "); got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestWingetUninstallArgs(t *testing.T) {
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return wingetOK("")
	}))

	if _, err := w.Uninstall(context.Background(), "Git.Git"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	want := "uninstall --id Git.Git --exact --silent"
	if got := strings.Join(calls[0].Args, " "); got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

// TestWingetRunCapturesOutput exercises run() directly: a successful invocation
// returns the captured stdout and a zero exit code.
func TestWingetRunCapturesOutput(t *testing.T) {
	const stdout = "installed Git.Git"
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return wingetOK(stdout)
	}))

	out, err := w.run(context.Background(), "install", "--id", "Git.Git")
	if err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", out.ExitCode)
	}
	if out.Stdout != stdout {
		t.Fatalf("Stdout = %q, want %q", out.Stdout, stdout)
	}
}

// TestWingetRunNonZeroWrapsStderrAndExitCode asserts a non-zero exit is wrapped
// into an error that carries both the exit code and the trimmed stderr.
func TestWingetRunNonZeroWrapsStderrAndExitCode(t *testing.T) {
	const stderr = "  0x8a15002b: deployment failed  \n"
	var calls []wingetCall
	w := newTestWingetBackend(scriptedWingetRunner(&calls, func(int, wingetCall) (*Output, error) {
		return wingetExit(1, stderr)
	}))

	out, err := w.run(context.Background(), "install", "--id", "Git.Git")
	if err == nil {
		t.Fatal("run() error = nil, want non-nil on non-zero exit")
	}
	if out == nil {
		t.Fatal("run() Output = nil, want populated even on error")
	}
	if out.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", out.ExitCode)
	}
	if !strings.Contains(err.Error(), "1") {
		t.Fatalf("error %q should mention exit code 1", err.Error())
	}
	if !strings.Contains(err.Error(), "deployment failed") {
		t.Fatalf("error %q should carry trimmed stderr", err.Error())
	}
}
