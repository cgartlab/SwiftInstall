package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// fakeBackend implements backend.Backend plus backend.InstalledSetProvider so
// the GUI handlers can be exercised without touching Homebrew.
type fakeBackend struct{ installed map[string]bool }

func (f fakeBackend) Name() string  { return "fake" }
func (f fakeBackend) Detect() error { return nil }

func (f fakeBackend) IsInstalled(_ context.Context, id string) (bool, error) {
	return f.installed[id], nil
}

func (f fakeBackend) Install(context.Context, string, backend.InstallOptions) (*backend.Output, error) {
	return &backend.Output{}, nil
}

func (f fakeBackend) Uninstall(context.Context, string) (*backend.Output, error) {
	return &backend.Output{}, nil
}

func (f fakeBackend) InstalledSet(context.Context) (map[string]bool, error) {
	return f.installed, nil
}

func testServer() *server {
	m := &manifest.Manifest{Packages: []manifest.Package{
		{ID: "git", Category: "Dev", Kind: "formula"},
		{ID: "figma", Category: "Design", Kind: "cask"},
	}}
	return &server{opts: Options{
		Manifest: m,
		Source:   "test.yaml",
		Backend:  fakeBackend{installed: map[string]bool{"git": true}},
	}}
}

func TestHandleState(t *testing.T) {
	rec := httptest.NewRecorder()
	testServer().handleState(rec, httptest.NewRequest(http.MethodGet, "/api/state", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out struct {
		Backend  string    `json:"backend"`
		Source   string    `json:"source"`
		Packages []pkgView `json:"packages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Backend != "fake" || out.Source != "test.yaml" {
		t.Errorf("meta = %q/%q", out.Backend, out.Source)
	}
	if len(out.Packages) != 2 || out.Packages[1].Kind != "cask" {
		t.Errorf("packages = %+v", out.Packages)
	}
}

func TestHandleStatusBulk(t *testing.T) {
	rec := httptest.NewRecorder()
	testServer().handleStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out struct {
		Installed map[string]bool `json:"installed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.Installed["git"] {
		t.Error("git should be installed")
	}
	if out.Installed["figma"] {
		t.Error("figma should be missing")
	}
}

func TestHandleRunRejectsBadInput(t *testing.T) {
	cases := []string{
		"/api/run?action=bogus&ids=git",
		"/api/run?action=install", // no ids
	}
	for _, url := range cases {
		rec := httptest.NewRecorder()
		testServer().handleRun(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", url, rec.Code)
		}
	}
}
