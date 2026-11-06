package gui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/llm"
)

func TestHandleAudit(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleAudit(rec, httptest.NewRequest(http.MethodGet, "/api/audit", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var out struct {
		CatalogID      string            `json:"catalog_id"`
		CatalogTitle   string            `json:"catalog_title"`
		InstalledCount int               `json:"installed_count"`
		Adds           []json.RawMessage `json:"adds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.CatalogID == "" {
		t.Error("catalog_id is empty")
	}
	if out.InstalledCount < 0 {
		t.Errorf("installed_count = %d", out.InstalledCount)
	}
	if len(out.Adds) == 0 {
		t.Error("expected at least one recommendation (catalog has items not installed)")
	}
}

func TestAuditPageEmbedded(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(sub, "audit.html")
	if err != nil {
		t.Fatalf("audit.html not embedded: %v", err)
	}
	if !bytes.Contains(b, []byte("环境体检")) {
		t.Error("audit.html missing expected marker")
	}
}

func TestHandleToollistExport(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleToollistExport(rec, httptest.NewRequest(http.MethodGet, "/api/toollist/export", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("id: mac-designers")) {
		t.Error("export body missing catalog id")
	}
	if rec.Header().Get("Content-Disposition") == "" {
		t.Error("missing Content-Disposition header")
	}
}

func TestHandleToollists(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleToollists(rec, httptest.NewRequest(http.MethodGet, "/api/toollists", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("developers")) {
		t.Error("toollists missing developers")
	}
}

func TestHandleAudit_ToollistParam(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleAudit(rec, httptest.NewRequest(http.MethodGet, "/api/audit?toollist=developers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	var out struct {
		CatalogID string `json:"catalog_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.CatalogID != "developers" {
		t.Errorf("catalog_id = %q, want developers", out.CatalogID)
	}
}

func TestHandleToollist(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleToollist(rec, httptest.NewRequest(http.MethodGet, "/api/toollist?toollist=developers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"id":"developers"`)) {
		t.Error("toollist body missing id")
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"items"`)) {
		t.Error("toollist body missing items")
	}
}

func TestToollistsPageEmbedded(t *testing.T) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(sub, "toollists.html")
	if err != nil {
		t.Fatalf("toollists.html not embedded: %v", err)
	}
	if !bytes.Contains(b, []byte("软件清单")) {
		t.Error("toollists.html missing marker")
	}
}

func TestHandleToollistImport(t *testing.T) {
	srv := &server{}
	yaml := "id: imported-x\nitems:\n  - name: Keka\n    why: w\n    license: oss\n    category: 文件\n"
	rec := httptest.NewRecorder()
	srv.handleToollistImport(rec, httptest.NewRequest(http.MethodPost, "/api/toollist/import", bytes.NewReader([]byte(yaml))))
	if rec.Code != http.StatusOK {
		t.Fatalf("import status = %d (%s)", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("imported-x")) {
		t.Error("import response missing id")
	}

	// appears in the list
	rec2 := httptest.NewRecorder()
	srv.handleToollists(rec2, httptest.NewRequest(http.MethodGet, "/api/toollists", nil))
	if !bytes.Contains(rec2.Body.Bytes(), []byte("imported-x")) {
		t.Error("imported toollist not listed")
	}

	// usable by audit
	rec3 := httptest.NewRecorder()
	srv.handleAudit(rec3, httptest.NewRequest(http.MethodGet, "/api/audit?toollist=imported-x", nil))
	var out struct {
		CatalogID string `json:"catalog_id"`
	}
	if err := json.Unmarshal(rec3.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.CatalogID != "imported-x" {
		t.Errorf("audit catalog_id = %q", out.CatalogID)
	}

	// bad yaml rejected
	rec4 := httptest.NewRecorder()
	srv.handleToollistImport(rec4, httptest.NewRequest(http.MethodPost, "/api/toollist/import", bytes.NewReader([]byte("items: [oops\n"))))
	if rec4.Code != http.StatusBadRequest {
		t.Errorf("bad yaml status = %d, want 400", rec4.Code)
	}
}

func TestHandleExplain_NoApp(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleExplain(rec, httptest.NewRequest(http.MethodGet, "/api/explain", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePresets(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handlePresets(rec, httptest.NewRequest(http.MethodGet, "/api/presets", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("presets")) {
		t.Error("missing presets key")
	}
}

func TestHandlePreset_NoName(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handlePreset(rec, httptest.NewRequest(http.MethodGet, "/api/preset", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleOutdated_NoCapability(t *testing.T) {
	rec := httptest.NewRecorder()
	testServer().handleOutdated(rec, httptest.NewRequest(http.MethodGet, "/api/outdated", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("fake backend lacks OutdatedProvider; status = %d, want 400", rec.Code)
	}
}

func TestHandleLLM_Get(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleLLM(rec, httptest.NewRequest(http.MethodGet, "/api/llm", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("configured")) {
		t.Error("missing configured field")
	}
}

// TestHandleExplain_Configured exercises the full GUI → advice → llm → HTTP
// chain against a local mock endpoint (config written to a temp HOME).
func TestHandleExplain_Configured(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Windows os.UserConfigDir() reads %APPDATA%; llm.Save writes there.
	t.Setenv("APPDATA", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"可用 Capso 直接替代。"}}]}`)
	}))
	defer srv.Close()
	if err := llm.Save(&llm.Config{BaseURL: srv.URL, APIKey: "sk-test", Model: "m"}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	(&server{}).handleExplain(rec, httptest.NewRequest(http.MethodGet, "/api/explain?app=CleanShot%20X", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Capso")) {
		t.Errorf("answer missing: %s", rec.Body.String())
	}
}

func TestHandleToollists_Freshness(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleToollists(rec, httptest.NewRequest(http.MethodGet, "/api/toollists", nil))
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"stale"`)) {
		t.Error("toollists response missing freshness field")
	}
}
