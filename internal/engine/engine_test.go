package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// fakeBackend is an in-package backend.Backend implementation that records
// calls and returns scripted results. It lets engine tests run without any
// real package manager.
type fakeBackend struct {
	name string

	installed map[string]bool

	// Scripted results consumed in order; once exhausted, calls succeed.
	installErrs    []error
	uninstallErrs  []error
	isInstalledErr error

	installCalls     []string
	installKinds     []string
	uninstallCalls   []string
	isInstalledCalls []string
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		name:      "fake",
		installed: map[string]bool{},
	}
}

func (f *fakeBackend) Name() string  { return f.name }
func (f *fakeBackend) Detect() error { return nil }

func (f *fakeBackend) IsInstalled(_ context.Context, id string) (bool, error) {
	f.isInstalledCalls = append(f.isInstalledCalls, id)
	if f.isInstalledErr != nil {
		return false, f.isInstalledErr
	}
	return f.installed[id], nil
}

func (f *fakeBackend) Install(_ context.Context, id string, opts backend.InstallOptions) (*backend.Output, error) {
	f.installCalls = append(f.installCalls, id)
	f.installKinds = append(f.installKinds, opts.Kind)
	if len(f.installErrs) > 0 {
		err := f.installErrs[0]
		f.installErrs = f.installErrs[1:]
		if err != nil {
			return &backend.Output{ExitCode: 1}, err
		}
	}
	return &backend.Output{ExitCode: 0}, nil
}

func (f *fakeBackend) Uninstall(_ context.Context, id string) (*backend.Output, error) {
	f.uninstallCalls = append(f.uninstallCalls, id)
	if len(f.uninstallErrs) > 0 {
		err := f.uninstallErrs[0]
		f.uninstallErrs = f.uninstallErrs[1:]
		if err != nil {
			return &backend.Output{ExitCode: 1}, err
		}
	}
	return &backend.Output{ExitCode: 0}, nil
}

func manifestOf(pkgs ...manifest.Package) *manifest.Manifest {
	return &manifest.Manifest{Packages: pkgs}
}

func statuses(results []Result) []Status {
	out := make([]Status, len(results))
	for i, r := range results {
		out[i] = r.Status
	}
	return out
}

func TestInstall_SkipExisting(t *testing.T) {
	fb := newFakeBackend()
	fb.installed["already"] = true

	e := New(fb, Options{SkipExisting: true})
	summary, err := e.Install(context.Background(), manifestOf(
		manifest.Package{ID: "already"},
		manifest.Package{ID: "new"},
	))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if summary.Total != 2 || summary.Succeeded != 1 || summary.Skipped != 1 || summary.Failed != 0 {
		t.Errorf("summary = %+v, want Total=2 Succeeded=1 Skipped=1 Failed=0", summary)
	}
	if want := []Status{StatusSkipped, StatusSuccess}; !reflect.DeepEqual(statuses(summary.Results), want) {
		t.Errorf("result statuses = %v, want %v", statuses(summary.Results), want)
	}
	// Only the not-installed package should be installed.
	if want := []string{"new"}; !reflect.DeepEqual(fb.installCalls, want) {
		t.Errorf("installCalls = %v, want %v", fb.installCalls, want)
	}
}

func TestInstall_SkipExistingDisabledInstallsAll(t *testing.T) {
	fb := newFakeBackend()
	fb.installed["already"] = true

	e := New(fb, Options{SkipExisting: false})
	summary, err := e.Install(context.Background(), manifestOf(
		manifest.Package{ID: "already"},
	))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if summary.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", summary.Succeeded)
	}
	if !reflect.DeepEqual(fb.installCalls, []string{"already"}) {
		t.Errorf("installCalls = %v, want [already]", fb.installCalls)
	}
}

func TestInstall_DryRunMakesNoCalls(t *testing.T) {
	fb := newFakeBackend()
	fb.installed["already"] = true

	e := New(fb, Options{DryRun: true, SkipExisting: true})
	summary, err := e.Install(context.Background(), manifestOf(
		manifest.Package{ID: "already"},
		manifest.Package{ID: "new"},
	))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if summary.Total != 2 || summary.Succeeded != 2 {
		t.Errorf("summary = %+v, want Total=2 Succeeded=2", summary)
	}
	if len(fb.installCalls) != 0 {
		t.Errorf("installCalls = %v, want none in dry-run", fb.installCalls)
	}
	if len(fb.isInstalledCalls) != 0 {
		t.Errorf("isInstalledCalls = %v, want none in dry-run", fb.isInstalledCalls)
	}
}

func TestInstall_RetrySucceedsAfterFailures(t *testing.T) {
	fb := newFakeBackend()
	fb.installErrs = []error{errors.New("fail 1"), errors.New("fail 2")}

	e := New(fb, Options{RetryCount: 2, RetryDelay: 0})
	summary, err := e.Install(context.Background(), manifestOf(manifest.Package{ID: "pkg"}))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if summary.Succeeded != 1 || summary.Failed != 0 {
		t.Errorf("summary = %+v, want Succeeded=1 Failed=0", summary)
	}
	if len(fb.installCalls) != 3 {
		t.Errorf("install attempted %d times, want 3 (1 try + 2 retries)", len(fb.installCalls))
	}
}

func TestInstall_RetryExhausted(t *testing.T) {
	fb := newFakeBackend()
	boom := errors.New("always fails")
	fb.installErrs = []error{boom, boom, boom}

	e := New(fb, Options{RetryCount: 2, RetryDelay: 0})
	summary, err := e.Install(context.Background(), manifestOf(manifest.Package{ID: "pkg"}))
	if err != nil {
		t.Fatalf("Install: unexpected top-level error: %v", err)
	}
	if summary.Failed != 1 || summary.Succeeded != 0 {
		t.Errorf("summary = %+v, want Failed=1 Succeeded=0", summary)
	}
	if len(fb.installCalls) != 3 {
		t.Errorf("install attempted %d times, want 3 (RetryCount+1)", len(fb.installCalls))
	}
	if summary.Results[0].Error == "" {
		t.Error("result.Error is empty, want failure message")
	}
}

func TestInstall_RetryCountZeroSingleAttempt(t *testing.T) {
	fb := newFakeBackend()
	fb.installErrs = []error{errors.New("fail")}

	e := New(fb, Options{RetryCount: 0})
	summary, _ := e.Install(context.Background(), manifestOf(manifest.Package{ID: "pkg"}))
	if summary.Failed != 1 {
		t.Errorf("summary = %+v, want Failed=1", summary)
	}
	if len(fb.installCalls) != 1 {
		t.Errorf("install attempted %d times, want 1 (no retries)", len(fb.installCalls))
	}
}

func TestInstall_OptionalFailureBecomesSkipped(t *testing.T) {
	fb := newFakeBackend()
	fb.installErrs = []error{errors.New("optional package missing")}

	e := New(fb, Options{RetryCount: 0})
	summary, err := e.Install(context.Background(), manifestOf(
		manifest.Package{ID: "opt", Optional: true},
	))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if got := summary.Results[0].Status; got != StatusOptionalFailed {
		t.Errorf("status = %q, want %q", got, StatusOptionalFailed)
	}
	if summary.Skipped != 1 || summary.Failed != 0 {
		t.Errorf("summary = %+v, want Skipped=1 Failed=0 (optional failure counts as skipped)", summary)
	}
}

func TestInstall_KindPropagatesToInstallOptions(t *testing.T) {
	fb := newFakeBackend()

	e := New(fb, Options{})
	_, err := e.Install(context.Background(), manifestOf(
		manifest.Package{ID: "Figma", Kind: "cask"},
	))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if len(fb.installKinds) != 1 {
		t.Fatalf("installKinds = %v, want exactly 1 entry", fb.installKinds)
	}
	if got := fb.installKinds[0]; got != "cask" {
		t.Errorf("InstallOptions.Kind = %q, want cask", got)
	}
}

func TestInstall_ProxyPropagates(t *testing.T) {
	fb := newFakeBackend()
	var seenProxies []string
	// Wrap by asserting through a second install after recording opts via kind slice
	// (proxy is not recorded by fake; use a dedicated backend here).
	proxyBackend := &proxyRecorder{fakeBackend: newFakeBackend(), proxies: &seenProxies}

	e := New(proxyBackend, Options{Proxy: "http://127.0.0.1:8080"})
	_, err := e.Install(context.Background(), manifestOf(manifest.Package{ID: "pkg"}))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if len(seenProxies) != 1 || seenProxies[0] != "http://127.0.0.1:8080" {
		t.Errorf("seen proxies = %v, want [http://127.0.0.1:8080]", seenProxies)
	}
	_ = fb
}

// proxyRecorder embeds fakeBackend and additionally records InstallOptions.Proxy.
type proxyRecorder struct {
	*fakeBackend
	proxies *[]string
}

func (p *proxyRecorder) Install(ctx context.Context, id string, opts backend.InstallOptions) (*backend.Output, error) {
	*p.proxies = append(*p.proxies, opts.Proxy)
	return p.fakeBackend.Install(ctx, id, opts)
}

func TestUninstall_SkipsNotInstalledSucceedsInstalled(t *testing.T) {
	fb := newFakeBackend()
	fb.installed["present"] = true // "absent" stays false

	e := New(fb, Options{})
	summary, err := e.Uninstall(context.Background(), manifestOf(
		manifest.Package{ID: "absent"},
		manifest.Package{ID: "present"},
	))
	if err != nil {
		t.Fatalf("Uninstall: unexpected error: %v", err)
	}
	if want := []Status{StatusSkipped, StatusSuccess}; !reflect.DeepEqual(statuses(summary.Results), want) {
		t.Errorf("statuses = %v, want %v", statuses(summary.Results), want)
	}
	if summary.Succeeded != 1 || summary.Skipped != 1 || summary.Failed != 0 {
		t.Errorf("summary = %+v, want Succeeded=1 Skipped=1 Failed=0", summary)
	}
	// Only installed package should be uninstalled.
	if want := []string{"present"}; !reflect.DeepEqual(fb.uninstallCalls, want) {
		t.Errorf("uninstallCalls = %v, want %v", fb.uninstallCalls, want)
	}
}

func TestUninstall_DryRunMakesNoCalls(t *testing.T) {
	fb := newFakeBackend()
	fb.installed["present"] = true

	e := New(fb, Options{DryRun: true})
	summary, err := e.Uninstall(context.Background(), manifestOf(manifest.Package{ID: "present"}))
	if err != nil {
		t.Fatalf("Uninstall: unexpected error: %v", err)
	}
	if summary.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", summary.Succeeded)
	}
	if len(fb.uninstallCalls) != 0 {
		t.Errorf("uninstallCalls = %v, want none in dry-run", fb.uninstallCalls)
	}
}

func TestInstall_ContextCancelledStopsLoop(t *testing.T) {
	fb := newFakeBackend()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before Install starts

	e := New(fb, Options{})
	summary, err := e.Install(ctx, manifestOf(
		manifest.Package{ID: "a"},
		manifest.Package{ID: "b"},
		manifest.Package{ID: "c"},
	))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Install error = %v, want context.Canceled", err)
	}
	if summary.Total != 3 {
		t.Errorf("Total = %d, want 3", summary.Total)
	}
	if len(summary.Results) != 0 {
		t.Errorf("got %d results, want 0 (loop should stop before processing)", len(summary.Results))
	}
	if len(fb.installCalls) != 0 {
		t.Errorf("installCalls = %v, want none", fb.installCalls)
	}
}

func TestInstall_ProgressCallback(t *testing.T) {
	fb := newFakeBackend()

	var (
		indices []int
		totals  []int
	)
	e := New(fb, Options{})
	e.OnProgress(func(index, total int, result Result) {
		indices = append(indices, index)
		totals = append(totals, total)
	})
	summary, err := e.Install(context.Background(), manifestOf(
		manifest.Package{ID: "a"},
		manifest.Package{ID: "b"},
	))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if summary.Total != 2 {
		t.Fatalf("Total = %d, want 2", summary.Total)
	}
	if !reflect.DeepEqual(indices, []int{0, 1}) {
		t.Errorf("progress indices = %v, want [0 1]", indices)
	}
	if !reflect.DeepEqual(totals, []int{2, 2}) {
		t.Errorf("progress totals = %v, want [2 2]", totals)
	}
}

func TestInstall_DeduplicatesPackages(t *testing.T) {
	fb := newFakeBackend()

	e := New(fb, Options{})
	summary, err := e.Install(context.Background(), manifestOf(
		manifest.Package{ID: "dup"},
		manifest.Package{ID: "dup"},
		manifest.Package{ID: "other"},
	))
	if err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if summary.Total != 2 {
		t.Errorf("Total = %d, want 2 (duplicates collapsed)", summary.Total)
	}
	if want := []string{"dup", "other"}; !reflect.DeepEqual(fb.installCalls, want) {
		t.Errorf("installCalls = %v, want %v", fb.installCalls, want)
	}
}
