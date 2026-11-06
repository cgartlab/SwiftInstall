package backend

import (
	"runtime"
	"testing"
)

// TestForCurrentOS_ReturnsErrorOrBackend asserts the registry always gives us
// exactly one of: a backend or an error (never neither, never both).
func TestForCurrentOS_ReturnsErrorOrBackend(t *testing.T) {
	be, err := ForCurrentOS()
	if be == nil && err == nil {
		t.Fatal("ForCurrentOS() returned (nil, nil); want a backend or an error")
	}
	if be != nil && err != nil {
		t.Fatalf("ForCurrentOS() returned both backend %q and error %v; want exactly one", be.Name(), err)
	}
}

// TestForCurrentOS_Darwin verifies that on this darwin host the registry selects
// the Homebrew backend.
func TestForCurrentOS_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skipf("host GOOS = %q, not darwin", runtime.GOOS)
	}
	be, err := ForCurrentOS()
	if err != nil {
		t.Fatalf("ForCurrentOS() on darwin returned error: %v", err)
	}
	if be == nil {
		t.Fatal("ForCurrentOS() on darwin returned nil backend")
	}
	if got := be.Name(); got != "homebrew" {
		t.Fatalf("backend.Name() = %q, want homebrew", got)
	}
}
