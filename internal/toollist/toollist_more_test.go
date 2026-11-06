package toollist

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadFile_OK(t *testing.T) {
	p := writeTemp(t, "id: x\ntitle: X\nitems:\n  - name: A\n    why: w\n    license: oss\n")
	c, err := LoadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != "x" || len(c.Items) != 1 || c.Items[0].Name != "A" {
		t.Fatalf("unexpected: %+v", c)
	}
}

func TestLoadFile_MissingID(t *testing.T) {
	p := writeTemp(t, "title: no id\nitems: []\n")
	if _, err := LoadFile(p); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestLoadFile_BadYAML(t *testing.T) {
	p := writeTemp(t, "id: x\nitems: [oops\n")
	if _, err := LoadFile(p); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestBuiltinIntegrity(t *testing.T) {
	c, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	allowedCat := map[string]bool{"效率": true, "系统": true, "文件": true, "设计": true,
		"开发": true, "写作": true, "媒体": true, "网络": true, "安全": true}
	allowedLic := map[string]bool{"free": true, "oss": true, "freemium": true}

	for _, it := range c.Items {
		if it.Name == "" || it.Why == "" {
			t.Errorf("item missing name/why: %+v", it)
		}
		if !allowedLic[it.License] {
			t.Errorf("item %q has unexpected license %q", it.Name, it.License)
		}
		if !allowedCat[it.Category] {
			t.Errorf("item %q has unexpected category %q", it.Name, it.Category)
		}
		if it.VerifiedAt == "" {
			t.Errorf("item %q missing verified_at", it.Name)
		}
		if len(it.Sources) == 0 {
			t.Errorf("item %q missing sources", it.Name)
		}
	}
	for _, eq := range c.Equivalences {
		if len(eq.From) == 0 || eq.To.Name == "" || eq.Why == "" {
			t.Errorf("malformed equivalence: %+v", eq)
		}
		if len(eq.Sources) == 0 {
			t.Errorf("equivalence to %q missing sources", eq.To.Name)
		}
	}
	if len(c.Items) < 20 {
		t.Errorf("builtin should have >=20 items, got %d", len(c.Items))
	}
}

func TestBuiltinPlaybooks(t *testing.T) {
	c, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, eq := range c.Equivalences {
		if eq.Playbook == nil {
			continue
		}
		found++
		pb := eq.Playbook
		if pb.Summary == "" || pb.DataFidelity == "" || pb.VerifiedAt == "" {
			t.Errorf("playbook to %q missing summary/data_fidelity/verified_at", eq.To.Name)
		}
		if len(pb.Steps) == 0 {
			t.Errorf("playbook to %q has no steps", eq.To.Name)
		}
		for _, st := range pb.Steps {
			if st.Title == "" || st.Detail == "" || st.SourceURL == "" {
				t.Errorf("playbook to %q has incomplete step: %+v", eq.To.Name, st)
			}
		}
	}
	if found < 3 {
		t.Errorf("expected >=3 playbooks in builtin, got %d", found)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	c, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	b, err := Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := parse(b)
	if err != nil {
		t.Fatalf("roundtrip parse: %v", err)
	}
	if c2.ID != c.ID || len(c2.Items) != len(c.Items) || len(c2.Equivalences) != len(c.Equivalences) {
		t.Fatalf("roundtrip mismatch: %s/%d/%d vs %s/%d/%d", c2.ID, len(c2.Items), len(c2.Equivalences), c.ID, len(c.Items), len(c.Equivalences))
	}
	found := false
	for _, e := range c2.Equivalences {
		if e.Playbook != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("playbook lost in roundtrip")
	}
}

func TestEmbeddedNames(t *testing.T) {
	names, err := EmbeddedNames()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range names {
		if n == "mac-designers.yaml" {
			found = true
		}
	}
	if !found {
		t.Fatalf("mac-designers.yaml not embedded: %v", names)
	}
}

func TestLoadBuiltinNamed(t *testing.T) {
	if _, err := LoadBuiltinNamed("mac-designers"); err != nil {
		t.Fatalf("by id: %v", err)
	}
	if _, err := LoadBuiltinNamed("mac-designers.yaml"); err != nil {
		t.Fatalf("by filename: %v", err)
	}
	if _, err := LoadBuiltinNamed("nope"); err == nil {
		t.Fatal("expected error for unknown catalog")
	}
}

func TestAllBuiltinsIntegrity(t *testing.T) {
	names, err := EmbeddedNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 2 {
		t.Fatalf("expected >=2 builtin toollists, got %v", names)
	}
	allowedCat := map[string]bool{"效率": true, "系统": true, "文件": true, "设计": true, "开发": true, "写作": true, "媒体": true, "网络": true, "安全": true}
	allowedLic := map[string]bool{"free": true, "oss": true, "freemium": true}
	for _, n := range names {
		c, err := LoadBuiltinNamed(n)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if len(c.Items) == 0 {
			t.Errorf("%s has no items", n)
		}
		for _, it := range c.Items {
			if !allowedLic[it.License] {
				t.Errorf("%s: item %q bad license %q", n, it.Name, it.License)
			}
			if !allowedCat[it.Category] {
				t.Errorf("%s: item %q bad category %q", n, it.Name, it.Category)
			}
			if it.VerifiedAt == "" || len(it.Sources) == 0 {
				t.Errorf("%s: item %q missing verified_at/sources", n, it.Name)
			}
		}
		for _, eq := range c.Equivalences {
			if len(eq.From) == 0 || eq.To.Name == "" || eq.Why == "" {
				t.Errorf("%s: malformed equivalence %+v", n, eq)
			}
		}
	}
}

func TestParse(t *testing.T) {
	c, err := Parse([]byte("id: z\nitems: []\n"))
	if err != nil || c.ID != "z" {
		t.Fatalf("Parse valid: %v %+v", err, c)
	}
	if _, err := Parse([]byte("items: []\n")); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestStaleness(t *testing.T) {
	now := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		v    string
		want string
		minD int
		maxD int
	}{
		{"2026-11-05", FreshnessFresh, 25, 27},
		{"2026-11", FreshnessFresh, 29, 31},
		{"2026-01-01", FreshnessAging, 333, 335},
		{"2025-01-01", FreshnessStale, 698, 700},
		{"", FreshnessUnknown, -1, -1},
	}
	for _, c := range cases {
		st, days := Staleness(c.v, now)
		if st != c.want {
			t.Errorf("Staleness(%q) state=%q want %q", c.v, st, c.want)
		}
		if days < c.minD || days > c.maxD {
			t.Errorf("Staleness(%q) days=%d want %d..%d", c.v, days, c.minD, c.maxD)
		}
	}
}

func TestFreshnessCounts(t *testing.T) {
	c, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	nowFresh := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	total, fresh, aging, stale := FreshnessCounts(c, nowFresh)
	if total == 0 || fresh != total || aging != 0 || stale != 0 {
		t.Errorf("expected all fresh: total=%d fresh=%d aging=%d stale=%d", total, fresh, aging, stale)
	}
	_, f2, a2, s2 := FreshnessCounts(c, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if s2 != total || f2 != 0 || a2 != 0 {
		t.Errorf("expected all stale: fresh=%d aging=%d stale=%d", f2, a2, s2)
	}
}
