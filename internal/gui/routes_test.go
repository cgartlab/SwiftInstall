package gui

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRoutes exercises the real mux wiring (the thing that a route rename could
// silently break), without binding the fixed port used by Run.
func TestRoutes(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(testServer().routes(sub))
	defer ts.Close()

	get := func(path string, want int) {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("%s = %d, want %d", path, resp.StatusCode, want)
		}
	}
	get("/", http.StatusOK)
	get("/audit", http.StatusOK)
	get("/toollists", http.StatusOK)
	get("/api/state", http.StatusOK)
	get("/api/status", http.StatusOK)
	get("/api/audit", http.StatusOK)
	get("/api/toollists", http.StatusOK)
	get("/api/toollist?toollist=developers", http.StatusOK)
	get("/api/toollist/export?toollist=developers", http.StatusOK)
	get("/api/presets", http.StatusOK)
	get("/api/outdated", http.StatusBadRequest) // fake backend lacks capability
	get("/api/explain", http.StatusBadRequest)  // missing app

	// POST import
	resp, err := http.Post(ts.URL+"/api/toollist/import", "application/x-yaml",
		strings.NewReader("id: rt-import\nitems:\n  - name: Keka\n    why: w\n    license: oss\n    category: 文件\n"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("import status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestPresetSaveFromToollist verifies the "设为安装清单" path: a toollist payload
// saved as a preset via POST /api/preset?action=save.
func TestPresetSaveFromToollist(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Windows os.UserConfigDir() reads %APPDATA%; preset.Save writes there.
	t.Setenv("APPDATA", t.TempDir())
	srv := testServer()
	rec := httptest.NewRecorder()
	body := `{"name":"toollist-x","packages":[{"id":"keka","category":"文件"},{"id":"raycast","category":"效率"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/preset?action=save", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.handlePreset(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	var out struct {
		OK    bool `json:"ok"`
		Total int  `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.Total != 2 {
		t.Errorf("unexpected response: %+v", out)
	}
}

func TestSwapRoute(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(testServer().routes(sub))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/swap")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/swap = %d, want 200", resp.StatusCode)
	}
}

func TestCommandPaletteAsset(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "__cmdkOpen") {
		t.Error("command palette not embedded in index.html")
	}
}

func TestCommandPaletteScript(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.ReadFile(sub, "cmdk.js"); err != nil {
		t.Fatalf("cmdk.js not embedded: %v", err)
	}
	ts := httptest.NewServer(testServer().routes(sub))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/cmdk.js")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/cmdk.js = %d, want 200", resp.StatusCode)
	}
}

func TestHealthRoutes(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(testServer().routes(sub))
	defer ts.Close()
	for _, p := range []string{"/health", "/api/health"} {
		resp, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s = %d, want 200", p, resp.StatusCode)
		}
	}
}

func TestShareRoute(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(testServer().routes(sub))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/share")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/share = %d, want 200", resp.StatusCode)
	}
}
