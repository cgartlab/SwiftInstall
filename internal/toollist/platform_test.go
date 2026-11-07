package toollist

import "testing"

func TestAvailableNamesFor_PlatformIsolation(t *testing.T) {
	win, err := AvailableNamesFor("windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(win) == 0 {
		t.Fatal("no builtin catalog available on windows")
	}
	mac, err := AvailableNamesFor("darwin")
	if err != nil {
		t.Fatal(err)
	}
	if len(mac) == 0 {
		t.Fatal("no builtin catalog available on darwin")
	}

	for _, n := range win {
		c, err := LoadBuiltinNamed(n)
		if err != nil {
			t.Fatal(err)
		}
		if c.Platform == PlatformMacOS {
			t.Errorf("windows list leaked macOS catalog %q", n)
		}
	}
	for _, n := range mac {
		c, err := LoadBuiltinNamed(n)
		if err != nil {
			t.Fatal(err)
		}
		if c.Platform == PlatformWindows {
			t.Errorf("darwin list leaked windows catalog %q", n)
		}
	}
}

func TestLoadBuiltinNamedFor_RejectsOtherPlatform(t *testing.T) {
	if _, err := LoadBuiltinNamedFor("mac-designers", "windows"); err == nil {
		t.Error("macOS catalog was accepted on windows")
	}
	if _, err := LoadBuiltinNamedFor("windows-essentials", "darwin"); err == nil {
		t.Error("windows catalog was accepted on darwin")
	}
	if _, err := LoadBuiltinNamedFor("windows-essentials", "windows"); err != nil {
		t.Errorf("windows catalog rejected on windows: %v", err)
	}
	if _, err := LoadBuiltinNamedFor("mac-designers", "darwin"); err != nil {
		t.Errorf("macOS catalog rejected on darwin: %v", err)
	}
}

func TestDefaultNameFor(t *testing.T) {
	if got := DefaultNameFor("windows"); got != "windows-essentials.yaml" {
		t.Errorf("windows default = %q", got)
	}
	if got := DefaultNameFor("darwin"); got != BuiltinName {
		t.Errorf("darwin default = %q, want %q", got, BuiltinName)
	}
}

func TestLoadBuiltinForPlatform(t *testing.T) {
	c, err := LoadBuiltinForPlatform()
	if err != nil {
		t.Fatal(err)
	}
	if !AvailableFor(c, CurrentPlatform()) {
		t.Errorf("default catalog %q (platform %q) is not available on %s", c.ID, c.Platform, CurrentPlatform())
	}
	if len(c.Items) == 0 {
		t.Error("default catalog has no items")
	}
}

func TestAllBuiltinsHaveValidPlatform(t *testing.T) {
	names, err := EmbeddedNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		c, err := LoadBuiltinNamed(n)
		if err != nil {
			t.Fatal(err)
		}
		switch c.Platform {
		case PlatformMacOS, PlatformWindows, PlatformAny:
		default:
			t.Errorf("%s has invalid platform %q", n, c.Platform)
		}
	}
}

func TestAvailableFor_ImportedCatalogIsPlatformNeutral(t *testing.T) {
	c := &Catalog{ID: "shared"}
	if !AvailableFor(c, "windows") || !AvailableFor(c, "darwin") {
		t.Error("an imported catalog without platform must be available everywhere")
	}
	c.Platform = PlatformAny
	if !AvailableFor(c, "windows") || !AvailableFor(c, "darwin") {
		t.Error("platform=any must be available everywhere")
	}
}
