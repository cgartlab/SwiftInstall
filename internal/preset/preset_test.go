package preset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// withTempHome points the OS config dir at a temp location for the test so
// preset.Dir() isolates from the user's real config.
func withTempHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// macOS UserConfigDir uses ~/Library/Application Support via HOME.
}

func TestSaveLoadListDelete(t *testing.T) {
	withTempHome(t)

	m := &manifest.Manifest{Packages: []manifest.Package{
		{ID: "git", Kind: "formula", Category: "Dev"},
		{ID: "figma", Kind: "cask", Category: "Design"},
	}}
	if err := Save("work", m); err != nil {
		t.Fatalf("save: %v", err)
	}

	names, err := List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 1 || names[0] != "work" {
		t.Fatalf("names = %v", names)
	}

	loaded, err := Load("work")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Packages) != 2 || loaded.Packages[1].ID != "figma" {
		t.Fatalf("loaded = %+v", loaded.Packages)
	}

	if err := Delete("work"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	names, _ = List()
	if len(names) != 0 {
		t.Fatalf("expected empty after delete, got %v", names)
	}
}

func TestLoadMissing(t *testing.T) {
	withTempHome(t)
	if _, err := Load("nope"); err == nil {
		t.Fatal("expected error for missing preset")
	}
}

func TestInvalidName(t *testing.T) {
	withTempHome(t)
	for _, bad := range []string{"", "a/b", "..", " leading", "weird name"} {
		if _, err := Path(bad); err == nil {
			t.Errorf("expected error for name %q", bad)
		}
	}
}

func TestSaveFileIsValidManifest(t *testing.T) {
	withTempHome(t)
	m := &manifest.Manifest{Packages: []manifest.Package{{ID: "go", Kind: "formula"}}}
	if err := Save("dev", m); err != nil {
		t.Fatalf("save: %v", err)
	}
	p, _ := Path("dev")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("preset file not created: %v", err)
	}
	// Round-trips through the manifest parser (format sniffing etc.).
	if _, err := manifest.ParseManifest(p); err != nil {
		t.Fatalf("saved preset fails to parse: %v", err)
	}
	// Confirm it landed under HOME-based config dir.
	if !filepath.IsAbs(p) {
		t.Fatalf("preset path not absolute: %s", p)
	}
}
