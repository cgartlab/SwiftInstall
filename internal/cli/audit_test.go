package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLicenseLabel(t *testing.T) {
	cases := map[string]string{
		"oss": "开源", "free": "免费", "freemium": "免费版够用",
		"paid": "付费", "weird": "weird",
	}
	for in, want := range cases {
		if got := licenseLabel(in); got != want {
			t.Errorf("licenseLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadToollist_BuiltinAndFile(t *testing.T) {
	c, err := loadToollist("", "")
	if err != nil {
		t.Fatalf("builtin: %v", err)
	}
	if c.ID == "" || len(c.Items) == 0 {
		t.Fatalf("builtin catalog empty: %+v", c)
	}

	if dc, err := loadToollist("", c.ID); err != nil || dc.ID != c.ID || len(dc.Items) == 0 {
		t.Fatalf("load %s: %v %+v", c.ID, err, dc)
	}

	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte("id: custom\nitems:\n  - name: X\n    why: y\n    license: free\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c2, err := loadToollist(p, "")
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	if c2.ID != "custom" {
		t.Fatalf("want custom, got %q", c2.ID)
	}

	if _, err := loadToollist(filepath.Join(t.TempDir(), "nope.yaml"), ""); err == nil {
		t.Fatal("expected error for missing catalog file")
	}
}
