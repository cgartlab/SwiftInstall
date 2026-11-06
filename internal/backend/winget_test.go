package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// wingetCall records one invocation of the injected runner.
type wingetCall struct{ args []string }

// newTestWinget builds a ready WingetBackend whose command execution is fully
// faked, so the Windows line can test the backend even on Linux/macOS CI
// runners where winget is not installed.
func newTestWinget(t *testing.T, fn func(args []string) (*Output, error)) (*WingetBackend, *[]wingetCall) {
	t.Helper()
	calls := &[]wingetCall{}
	w := NewWingetBackend()
	w.lookPath = func(string) (string, error) { return "C:/fake/winget.exe", nil }
	w.run = func(_ context.Context, args ...string) (*Output, error) {
		*calls = append(*calls, wingetCall{args: append([]string(nil), args...)})
		if fn != nil {
			return fn(args)
		}
		return &Output{ExitCode: 0}, nil
	}
	if err := w.Detect(); err != nil {
		t.Fatalf("Detect() = %v", err)
	}
	return w, calls
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
}

func TestWingetName(t *testing.T) {
	if got := NewWingetBackend().Name(); got != "winget" {
		t.Fatalf("Name() = %q, want winget", got)
	}
}

func TestWingetDetect(t *testing.T) {
	t.Run("prefers winget.exe", func(t *testing.T) {
		w := NewWingetBackend()
		w.lookPath = func(file string) (string, error) {
			if file != "winget.exe" {
				t.Fatalf("lookPath(%q), want winget.exe", file)
			}
			return "C:/fake/winget.exe", nil
		}
		if err := w.Detect(); err != nil {
			t.Fatal(err)
		}
		if w.execPath != "C:/fake/winget.exe" {
			t.Fatalf("execPath = %q", w.execPath)
		}
	})

	t.Run("falls back to winget", func(t *testing.T) {
		w := NewWingetBackend()
		w.lookPath = func(file string) (string, error) {
			if file == "winget.exe" {
				return "", errors.New("missing")
			}
			return "C:/fake/winget", nil
		}
		if err := w.Detect(); err != nil {
			t.Fatal(err)
		}
		if w.execPath != "C:/fake/winget" {
			t.Fatalf("execPath = %q", w.execPath)
		}
	})

	t.Run("missing gives friendly error", func(t *testing.T) {
		w := NewWingetBackend()
		w.lookPath = func(string) (string, error) { return "", errors.New("missing") }
		err := w.Detect()
		if err == nil || !strings.Contains(err.Error(), "winget is not installed") {
			t.Fatalf("Detect() error = %v", err)
		}
		if w.ready {
			t.Fatal("ready must stay false after a failed Detect()")
		}
	})
}

func TestWingetRequiresDetect(t *testing.T) {
	w := NewWingetBackend()
	w.run = func(context.Context, ...string) (*Output, error) { return &Output{}, nil }
	if _, err := w.IsInstalled(context.Background(), "Git.Git"); err == nil {
		t.Error("IsInstalled before Detect() returned no error")
	}
	if _, err := w.Install(context.Background(), "Git.Git", InstallOptions{}); err == nil {
		t.Error("Install before Detect() returned no error")
	}
	if _, err := w.Uninstall(context.Background(), "Git.Git"); err == nil {
		t.Error("Uninstall before Detect() returned no error")
	}
}

func TestWingetIsInstalled(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		w, calls := newTestWinget(t, func([]string) (*Output, error) {
			return &Output{ExitCode: 0, Stdout: "Git.Git  2.43.0"}, nil
		})
		ok, err := w.IsInstalled(context.Background(), "Git.Git")
		if err != nil || !ok {
			t.Fatalf("IsInstalled = %v, %v; want true, nil", ok, err)
		}
		assertArgs(t, (*calls)[0].args, []string{
			"list", "--id", "Git.Git", "--exact", "--accept-source-agreements",
		})
	})

	t.Run("not found is a negative answer", func(t *testing.T) {
		w, _ := newTestWinget(t, func([]string) (*Output, error) {
			return &Output{ExitCode: 1}, errors.New("exit 1")
		})
		ok, err := w.IsInstalled(context.Background(), "Missing.Package")
		if err != nil || ok {
			t.Fatalf("IsInstalled = %v, %v; want false, nil", ok, err)
		}
	})

	t.Run("genuine error propagates", func(t *testing.T) {
		w, _ := newTestWinget(t, func([]string) (*Output, error) {
			return &Output{ExitCode: 0}, errors.New("spawn failed")
		})
		if _, err := w.IsInstalled(context.Background(), "Git.Git"); err == nil {
			t.Fatal("expected error for a non-exit failure")
		}
	})
}

func TestWingetInstallArgs(t *testing.T) {
	w, calls := newTestWinget(t, nil)
	if _, err := w.Install(context.Background(), "Git.Git", InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	assertArgs(t, (*calls)[0].args, []string{
		"install", "--id", "Git.Git", "--exact", "--silent",
		"--accept-package-agreements", "--accept-source-agreements",
	})
}

func TestWingetInstallProxy(t *testing.T) {
	w, calls := newTestWinget(t, nil)
	if _, err := w.Install(context.Background(), "Git.Git", InstallOptions{Proxy: "http://127.0.0.1:7897"}); err != nil {
		t.Fatal(err)
	}
	assertArgs(t, (*calls)[0].args, []string{
		"install", "--id", "Git.Git", "--exact", "--silent",
		"--accept-package-agreements", "--accept-source-agreements",
		"--proxy", "http://127.0.0.1:7897",
	})
}

func TestWingetUninstallArgs(t *testing.T) {
	w, calls := newTestWinget(t, nil)
	if _, err := w.Uninstall(context.Background(), "Git.Git"); err != nil {
		t.Fatal(err)
	}
	assertArgs(t, (*calls)[0].args, []string{
		"uninstall", "--id", "Git.Git", "--exact", "--silent",
	})
}

// TestWingetHelperProcess is only ever run as a child process by
// TestWingetExecCommandCapturesOutput; as a normal test it does nothing.
func TestWingetHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_WINGET_HELPER") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "hello-out")
	fmt.Fprint(os.Stderr, "hello-err")
	os.Exit(0)
}

// The production runner must capture stdout, stderr, exit code and duration.
func TestWingetExecCommandCapturesOutput(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GO_WANT_WINGET_HELPER", "1")
	w := NewWingetBackend()
	w.execPath = exe
	out, err := w.execCommand(context.Background(), "-test.run=^TestWingetHelperProcess$")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if !strings.Contains(out.Stdout, "hello-out") {
		t.Fatalf("stdout = %q", out.Stdout)
	}
	if !strings.Contains(out.Stderr, "hello-err") {
		t.Fatalf("stderr = %q", out.Stderr)
	}
	if out.ExitCode != 0 {
		t.Fatalf("ExitCode = %d", out.ExitCode)
	}
	if out.DurationMs < 0 {
		t.Fatalf("DurationMs = %d", out.DurationMs)
	}
}
