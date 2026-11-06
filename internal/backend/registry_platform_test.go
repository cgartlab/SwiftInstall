package backend

import (
	"context"
	"testing"
)

type stubBackend struct{ name string }

func (s *stubBackend) Name() string                                      { return s.name }
func (s *stubBackend) Detect() error                                     { return nil }
func (s *stubBackend) IsInstalled(context.Context, string) (bool, error) { return false, nil }
func (s *stubBackend) Install(context.Context, string, InstallOptions) (*Output, error) {
	return &Output{}, nil
}
func (s *stubBackend) Uninstall(context.Context, string) (*Output, error) { return &Output{}, nil }

func TestPlatformFactories_ContainsDefaults(t *testing.T) {
	got := PlatformFactories()
	for _, want := range []string{"darwin", "windows"} {
		found := false
		for _, g := range got {
			if g == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("PlatformFactories() = %v, missing %q", got, want)
		}
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("PlatformFactories() not sorted: %v", got)
		}
	}
}

// The shared baseline must be able to resolve any registered platform on any
// host — this is what lets the Windows CI exercise both platform mappings.
func TestForGOOS_AnyRegisteredPlatform(t *testing.T) {
	for _, goos := range []string{"windows", "darwin"} {
		be, err := ForGOOS(goos)
		if err != nil {
			t.Fatalf("ForGOOS(%q) returned error: %v", goos, err)
		}
		if be == nil {
			t.Fatalf("ForGOOS(%q) returned nil backend", goos)
		}
	}
}

func TestForGOOS_UnregisteredReturnsError(t *testing.T) {
	if _, err := ForGOOS("no-such-os"); err == nil {
		t.Fatal("ForGOOS on an unregistered GOOS returned no error")
	}
}

// A platform line replaces its backend without touching shared code.
func TestRegisterPlatform_Override(t *testing.T) {
	const goos = "test-os"
	RegisterPlatform(goos, func() Backend { return &stubBackend{name: "first"} })
	t.Cleanup(func() {
		registryMu.Lock()
		delete(platformFactories, goos)
		registryMu.Unlock()
	})

	be, err := ForGOOS(goos)
	if err != nil {
		t.Fatalf("ForGOOS(%q) after first registration: %v", goos, err)
	}
	if got := be.Name(); got != "first" {
		t.Fatalf("backend name = %q, want first", got)
	}

	RegisterPlatform(goos, func() Backend { return &stubBackend{name: "second"} })
	be, err = ForGOOS(goos)
	if err != nil {
		t.Fatalf("ForGOOS(%q) after override: %v", goos, err)
	}
	if got := be.Name(); got != "second" {
		t.Fatalf("backend name = %q, want second after override", got)
	}
}
