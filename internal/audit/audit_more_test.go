package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/toollist"
)

func TestSystemLister_TempDir(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Foo.app"))          // app at root
	mustMkdir(t, filepath.Join(root, "Sub", "Bar.app"))   // nested one level
	mustWrite(t, filepath.Join(root, "notes.txt"))        // non-app file, ignored
	mustWrite(t, filepath.Join(root, "Tool", "bin", "x")) // deep non-app, ignored

	got, err := (SystemLister{Roots: []string{root}}).Installed(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, g := range got {
		set[g] = true
	}
	if !set["Foo"] || !set["Bar"] {
		t.Fatalf("expected Foo and Bar, got %v", got)
	}
	if set["notes"] || set["x"] {
		t.Fatalf("non-app entries should be ignored, got %v", got)
	}
}

func TestSystemLister_MissingRootIsIgnored(t *testing.T) {
	got, err := (SystemLister{Roots: []string{"/definitely/not/here/xyz"}}).Installed(context.Background())
	if err != nil {
		t.Fatalf("missing root should not error: %v", err)
	}
	_ = got // may still contain brew tokens on a dev machine
}

func TestRun_EmptyInstalledAllAdds(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Title: "T", Items: []toollist.Item{
		{Name: "A", License: "oss", Category: "设计", Why: "w"},
		{Name: "B", License: "free", Category: "效率", Why: "w"},
	}}
	rep, err := Run(context.Background(), fakeLister{}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Adds) != 2 || len(rep.Keeps) != 0 || len(rep.Replaces) != 0 {
		t.Fatalf("unexpected report: adds=%d keeps=%d replaces=%d", len(rep.Adds), len(rep.Keeps), len(rep.Replaces))
	}
	if rep.Adds[0].Category != "设计" {
		t.Errorf("category not propagated: %+v", rep.Adds[0])
	}
	if rep.InstalledCount != 0 {
		t.Errorf("installed_count = %d, want 0", rep.InstalledCount)
	}
}

func TestRun_EmptyCatalog(t *testing.T) {
	rep, err := Run(context.Background(), fakeLister{apps: []string{"Foo"}}, &toollist.Catalog{ID: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Adds) != 0 || len(rep.Keeps) != 0 || len(rep.Replaces) != 0 {
		t.Fatalf("empty catalog should yield empty report, got %+v", rep)
	}
	if rep.InstalledCount != 1 {
		t.Errorf("installed_count = %d, want 1", rep.InstalledCount)
	}
}

func TestRun_KeepMatchesByNameAndBrew(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Items: []toollist.Item{
		{Name: "Visual Studio Code", Brew: "visual-studio-code", License: "free"},
	}}
	// installed only via .app display name
	rep, _ := Run(context.Background(), fakeLister{apps: []string{"Visual Studio Code.app"}}, cat)
	if len(rep.Keeps) != 1 {
		t.Fatalf("expected keep by app name, got %+v", rep.Keeps)
	}
	// installed only via brew token
	rep2, _ := Run(context.Background(), fakeLister{apps: []string{"visual-studio-code"}}, cat)
	if len(rep2.Keeps) != 1 {
		t.Fatalf("expected keep by brew token, got %+v", rep2.Keeps)
	}
}

func TestNormalize_CJK(t *testing.T) {
	if got := Normalize("微信"); got != "微信" {
		t.Errorf("Normalize(微信) = %q", got)
	}
	if got := Normalize("  Raycast.app  "); got != "raycast" {
		t.Errorf("Normalize(Raycast.app) = %q", got)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindEquivalence(t *testing.T) {
	cat := sampleCatalog()
	if eq := FindEquivalence(cat, "CleanShot X.app"); eq == nil || eq.To.Name != "Capso" {
		t.Fatalf("expected Capso for CleanShot X.app, got %+v", eq)
	}
	if eq := FindEquivalence(cat, "cleanshot"); eq == nil {
		t.Fatal("expected match by brew token cleanshot")
	}
	if eq := FindEquivalence(cat, "no-such-app"); eq != nil {
		t.Fatalf("expected nil, got %+v", eq)
	}
}

func TestRun_PlaybookPropagates(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Equivalences: []toollist.Equivalence{
		{From: []string{"X"}, To: toollist.To{Name: "Y", License: "oss"},
			Playbook: &toollist.Playbook{Summary: "s", DataFidelity: "d", VerifiedAt: "2026-11",
				Steps: []toollist.Step{{Title: "t", Detail: "d", SourceURL: "u"}}}},
	}}
	rep, err := Run(context.Background(), fakeLister{apps: []string{"X"}}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Replaces) != 1 {
		t.Fatalf("want 1 replace, got %d", len(rep.Replaces))
	}
	pb := rep.Replaces[0].Playbook
	if pb == nil || pb.Summary != "s" || len(pb.Steps) != 1 {
		t.Fatalf("playbook not propagated: %+v", pb)
	}
}

func TestRun_Unmatched(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Items: []toollist.Item{{Name: "IINA", Brew: "iina", License: "oss"}}}
	rep, err := Run(context.Background(), fakeLister{apps: []string{"IINA", "Foo", "Bar"}}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if rep.UnmatchedCount != 2 {
		t.Fatalf("unmatched = %d, want 2", rep.UnmatchedCount)
	}
	set := map[string]bool{}
	for _, n := range rep.UnmatchedSample {
		set[n] = true
	}
	if !set["Foo"] || !set["Bar"] {
		t.Errorf("sample should include Foo & Bar: %v", rep.UnmatchedSample)
	}
}
