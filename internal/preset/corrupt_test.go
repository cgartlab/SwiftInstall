package preset

import (
	"os"
	"path/filepath"
	"testing"
)

// A corrupt preset file must not break listing; loading it must return an error.
func TestCorruptPresetTolerated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("no user config dir: %v", err)
	}
	pdir := filepath.Join(dir, "sis", "presets")
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pdir, "bad.yaml"), []byte("a: [1, 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	names, err := List()
	if err != nil {
		t.Fatalf("List should tolerate corrupt file, got %v", err)
	}
	found := false
	for _, n := range names {
		if n == "bad" {
			found = true
		}
	}
	if !found {
		t.Errorf("List should include 'bad', got %v", names)
	}
	if _, err := Load("bad"); err == nil {
		t.Error("Load of corrupt preset should error")
	}
}
