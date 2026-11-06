package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// fixturePath resolves a repo-root fixture from the package test directory
// (internal/manifest -> repo root).
func fixturePath(name string) string {
	return filepath.Join("..", "..", name)
}

// writeTemp writes content to a named file inside a fresh temp dir.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp manifest %q: %v", name, err)
	}
	return p
}

func packageIDs(pkgs []Package) []string {
	out := make([]string, len(pkgs))
	for i, p := range pkgs {
		out[i] = p.ID
	}
	return out
}

func TestParseManifest_ValidYAML(t *testing.T) {
	m, err := ParseManifest(fixturePath("test_manifest_valid.yaml"))
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	wantIDs := []string{"Microsoft.VisualStudioCode", "Git.Git", "7zip.7zip"}
	if got := packageIDs(m.Packages); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("package IDs = %v, want %v", got, wantIDs)
	}
	if !m.Settings.SkipExisting {
		t.Errorf("SkipExisting = false, want true")
	}
	if m.Settings.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", m.Settings.RetryCount)
	}
	if m.Settings.RetryDelay != 3 {
		t.Errorf("RetryDelay = %d, want 3", m.Settings.RetryDelay)
	}
	if m.Settings.Proxy != "" {
		t.Errorf("Proxy = %q, want empty", m.Settings.Proxy)
	}
	if err := Validate(m); err != nil {
		t.Errorf("Validate(valid manifest) = %v, want nil", err)
	}
}

func TestParseManifest_PlainTXT(t *testing.T) {
	m, err := ParseManifest(fixturePath("test_manifest_txt.txt"))
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	wantIDs := []string{"Microsoft.VisualStudioCode", "Git.Git", "7zip.7zip"}
	if got := packageIDs(m.Packages); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("package IDs = %v, want %v", got, wantIDs)
	}
	wantCats := []string{"dev", "dev", "utils"}
	gotCats := make([]string, len(m.Packages))
	for i, p := range m.Packages {
		gotCats[i] = p.Category
	}
	if !reflect.DeepEqual(gotCats, wantCats) {
		t.Errorf("categories = %v, want %v", gotCats, wantCats)
	}
}

func TestParseManifest_Dedupe(t *testing.T) {
	m, err := ParseManifest(fixturePath("test_manifest_dup.yaml"))
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	wantIDs := []string{"Microsoft.VisualStudioCode", "Git.Git"}
	if got := packageIDs(m.Packages); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("deduped IDs = %v, want %v", got, wantIDs)
	}
}

func TestParseManifest_Optional(t *testing.T) {
	m, err := ParseManifest(fixturePath("test_manifest_optional.yaml"))
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	if len(m.Packages) != 2 {
		t.Fatalf("got %d packages, want 2", len(m.Packages))
	}
	if !m.Packages[0].Optional {
		t.Errorf("packages[0].Optional = false, want true")
	}
	if m.Packages[1].Optional {
		t.Errorf("packages[1].Optional = true, want false")
	}
}

func TestParseManifest_BrokenYAML(t *testing.T) {
	if _, err := ParseManifest(fixturePath("test_manifest_broken.yaml")); err == nil {
		t.Fatal("ParseManifest(broken) = nil error, want parse error")
	}
}

func TestParseManifest_Empty(t *testing.T) {
	m, err := ParseManifest(fixturePath("test_manifest_empty.yaml"))
	if err != nil {
		t.Fatalf("ParseManifest(empty): unexpected error: %v", err)
	}
	if len(m.Packages) != 0 {
		t.Errorf("got %d packages, want 0", len(m.Packages))
	}
	if err := Validate(m); err == nil {
		t.Error("Validate(empty manifest) = nil, want 'no packages' error")
	}
}

func TestParseManifest_InvalidEmptyID(t *testing.T) {
	// Parsing succeeds; the empty ID is a validation-level error.
	m, err := ParseManifest(fixturePath("test_manifest_invalid.yaml"))
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	if len(m.Packages) != 2 {
		t.Fatalf("got %d packages, want 2", len(m.Packages))
	}
	if err := Validate(m); err == nil {
		t.Fatal("Validate(invalid manifest) = nil, want empty-ID error")
	}
}

func TestParseManifest_UnknownFields(t *testing.T) {
	m, err := ParseManifest(fixturePath("test_manifest_unknown_fields.yaml"))
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	wantIDs := []string{"Git.Git"}
	if got := packageIDs(m.Packages); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("IDs = %v, want %v", got, wantIDs)
	}
}

// TestParseManifest_ContentSniffing verifies that format is decided by content,
// not extension: a .txt carrying a `packages:`/`settings:` mapping is parsed as
// YAML, while a colon-free .txt stays a plain line list.
func TestParseManifest_ContentSniffing(t *testing.T) {
	yamlInTxt := "settings:\n  skip_existing: true\npackages:\n  - id: Git.Git\n    category: dev\n    kind: formula\n"
	p := writeTemp(t, "manifest.txt", yamlInTxt)
	m, err := ParseManifest(p)
	if err != nil {
		t.Fatalf("ParseManifest(.txt with YAML content): unexpected error: %v", err)
	}
	wantIDs := []string{"Git.Git"}
	if got := packageIDs(m.Packages); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("IDs = %v, want %v (should have been parsed as YAML)", got, wantIDs)
	}
	if !m.Settings.SkipExisting {
		t.Errorf("SkipExisting = false, want true (YAML settings not read)")
	}
	if m.Packages[0].Category != "dev" {
		t.Errorf("Category = %q, want dev", m.Packages[0].Category)
	}
	if m.Packages[0].Kind != "formula" {
		t.Errorf("Kind = %q, want formula", m.Packages[0].Kind)
	}
}

func TestParseManifest_SettingsOnlyTXTIsYAML(t *testing.T) {
	// Only a settings key, no packages: still a YAML mapping by content.
	p := writeTemp(t, "settings_only.txt", "settings:\n  retry_count: 4\n")
	m, err := ParseManifest(p)
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	if m.Settings.RetryCount != 4 {
		t.Errorf("RetryCount = %d, want 4", m.Settings.RetryCount)
	}
	if len(m.Packages) != 0 {
		t.Errorf("got %d packages, want 0", len(m.Packages))
	}
}

// TestParseManifest_Kind verifies the kind: cask channel parses into Package.Kind.
func TestParseManifest_Kind(t *testing.T) {
	p := writeTemp(t, "kind.yaml", "packages:\n  - id: Figma\n    kind: cask\n")
	m, err := ParseManifest(p)
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	if len(m.Packages) != 1 {
		t.Fatalf("got %d packages, want 1", len(m.Packages))
	}
	if m.Packages[0].Kind != "cask" {
		t.Errorf("Kind = %q, want cask", m.Packages[0].Kind)
	}
}

func TestParseManifest_BOM(t *testing.T) {
	const bom = "\uFEFF"

	t.Run("yaml file", func(t *testing.T) {
		p := writeTemp(t, "bom.yaml", bom+"packages:\n  - id: Git.Git\n")
		m, err := ParseManifest(p)
		if err != nil {
			t.Fatalf("ParseManifest: unexpected error: %v", err)
		}
		if got := packageIDs(m.Packages); !reflect.DeepEqual(got, []string{"Git.Git"}) {
			t.Errorf("IDs = %v, want [Git.Git]", got)
		}
	})

	t.Run("plain txt file", func(t *testing.T) {
		p := writeTemp(t, "bom.txt", bom+"Git.Git\n7zip.7zip\n")
		m, err := ParseManifest(p)
		if err != nil {
			t.Fatalf("ParseManifest: unexpected error: %v", err)
		}
		if got := packageIDs(m.Packages); !reflect.DeepEqual(got, []string{"Git.Git", "7zip.7zip"}) {
			t.Errorf("IDs = %v, want [Git.Git 7zip.7zip]", got)
		}
	})

	t.Run("txt file with yaml content", func(t *testing.T) {
		p := writeTemp(t, "bom_sniff.txt", bom+"packages:\n  - id: Git.Git\n")
		m, err := ParseManifest(p)
		if err != nil {
			t.Fatalf("ParseManifest: unexpected error: %v", err)
		}
		if got := packageIDs(m.Packages); !reflect.DeepEqual(got, []string{"Git.Git"}) {
			t.Errorf("IDs = %v, want [Git.Git] (BOM should not defeat content sniffing)", got)
		}
	})
}

func TestParseManifest_EmptyYAMLFile(t *testing.T) {
	p := writeTemp(t, "empty.yaml", "")
	m, err := ParseManifest(p)
	if err != nil {
		t.Fatalf("ParseManifest(empty file): unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("ParseManifest returned nil manifest")
	}
	if len(m.Packages) != 0 {
		t.Errorf("got %d packages, want 0", len(m.Packages))
	}
}

func TestParseManifest_TXTInlineComment(t *testing.T) {
	p := writeTemp(t, "inline.txt", "Git.Git  # version control\n7zip.7zip\n")
	m, err := ParseManifest(p)
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	if got := packageIDs(m.Packages); !reflect.DeepEqual(got, []string{"Git.Git", "7zip.7zip"}) {
		t.Errorf("IDs = %v, want [Git.Git 7zip.7zip] (inline comment should be stripped)", got)
	}
}

func TestParseManifest_Errors(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		if _, err := ParseManifest(""); err == nil {
			t.Error("ParseManifest(\"\") = nil error, want error")
		}
	})
	t.Run("missing file", func(t *testing.T) {
		if _, err := ParseManifest(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
			t.Error("ParseManifest(missing) = nil error, want read error")
		}
	})
}
