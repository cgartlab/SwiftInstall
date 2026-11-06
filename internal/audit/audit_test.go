package audit

import (
	"context"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/toollist"
)

type fakeLister struct {
	apps []string
	err  error
}

func (f fakeLister) Installed(context.Context) ([]string, error) { return f.apps, f.err }

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Visual Studio Code.app": "visualstudiocode",
		"visual-studio-code":     "visualstudiocode",
		"CleanShot X":            "cleanshotx",
		"1Password":              "1password",
		"  Keka  ":               "keka",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func sampleCatalog() *toollist.Catalog {
	return &toollist.Catalog{
		ID:    "test",
		Title: "test toollist",
		Items: []toollist.Item{
			{Name: "IINA", Brew: "iina", License: "oss"},
			{Name: "Raycast", Brew: "raycast", License: "free"},
		},
		Equivalences: []toollist.Equivalence{
			{
				From: []string{"CleanShot X", "cleanshot"},
				To:   toollist.To{Name: "Capso", Brew: "lzhgus/tap/capso", License: "oss"},
				Why:  "open-source alternative",
			},
		},
	}
}

func TestRun_ReplaceAndKeep(t *testing.T) {
	l := fakeLister{apps: []string{"CleanShot X", "IINA"}}
	rep, err := Run(context.Background(), l, sampleCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Replaces) != 1 {
		t.Fatalf("want 1 replace, got %d", len(rep.Replaces))
	}
	if rep.Replaces[0].From != "CleanShot X" || rep.Replaces[0].Name != "Capso" {
		t.Errorf("unexpected replace: %+v", rep.Replaces[0])
	}
	if len(rep.Keeps) != 1 || rep.Keeps[0] != "IINA" {
		t.Errorf("want keeps=[IINA], got %v", rep.Keeps)
	}
	// Raycast not installed -> should be an "add"
	if len(rep.Adds) != 1 || rep.Adds[0].Name != "Raycast" {
		t.Errorf("want adds=[Raycast], got %v", rep.Adds)
	}
}

func TestRun_TargetAlreadyInstalledSuppressesReplace(t *testing.T) {
	l := fakeLister{apps: []string{"CleanShot X", "Capso"}}
	rep, err := Run(context.Background(), l, sampleCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Replaces) != 0 {
		t.Fatalf("expected no replace when target installed, got %+v", rep.Replaces)
	}
}

func TestRun_BrewTokenMatch(t *testing.T) {
	// Installed via brew token only (no .app name).
	l := fakeLister{apps: []string{"cleanshot"}}
	rep, err := Run(context.Background(), l, sampleCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Replaces) != 1 {
		t.Fatalf("brew token should match; got %+v", rep.Replaces)
	}
}
