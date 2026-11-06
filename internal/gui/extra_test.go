package gui

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/backend"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(testServer().routes(sub))
}

func TestHandleInfo_NoProvider(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/info?name=git")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("info without provider = %d, want 400", resp.StatusCode)
	}
}

func TestHandleInfo_MissingName(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("info missing name = %d, want 400", resp.StatusCode)
	}
}

func TestHandleSearch_NoSearcher(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	for _, u := range []string{"/api/search?q=", "/api/search?q=git"} {
		resp, err := http.Get(ts.URL + u)
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Hits []any `json:"hits"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("%s decode: %v", u, err)
		}
		if resp.StatusCode != http.StatusOK || len(body.Hits) != 0 {
			t.Errorf("%s = %d hits=%d, want 200 hits=0", u, resp.StatusCode, len(body.Hits))
		}
	}
}

func TestHandleHealth_JSON(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health = %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	tot, ok := body["toollists_total"].(map[string]any)
	if !ok {
		t.Fatalf("missing toollists_total: %v", body["toollists_total"])
	}
	if items, _ := tot["items"].(float64); items <= 0 {
		t.Errorf("toollists_total.items = %v, want > 0", tot["items"])
	}
}

func TestCapPerSource(t *testing.T) {
	hits := []backend.SearchHit{
		{Name: "a1", Source: "brew"},
		{Name: "a2", Source: "brew"},
		{Name: "b1", Source: "npm"},
	}
	out := capPerSource(hits, 1)
	if len(out) != 2 {
		t.Fatalf("capPerSource len = %d, want 2", len(out))
	}
	bySrc := map[string]int{}
	for _, h := range out {
		bySrc[h.Source]++
	}
	if bySrc["brew"] != 1 || bySrc["npm"] != 1 {
		t.Errorf("per-source counts = %v, want brew=1 npm=1", bySrc)
	}
}

func TestCatalogNotFound_404(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	for _, u := range []string{
		"/api/toollist?toollist=bogus",
		"/api/toollist/export?toollist=bogus",
		"/api/audit?toollist=bogus",
	} {
		resp, err := http.Get(ts.URL + u)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", u, resp.StatusCode)
		}
	}
}

func TestPresetTraversalRejected(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	for _, name := range []string{"../../etc/passwd", "../x", "a/../../b"} {
		resp, err := http.Get(ts.URL + "/api/preset?name=" + name)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Errorf("preset name %q should be rejected, got 200", name)
		}
	}
}

func TestImportThenFetch(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	body := "id: clitest\ntitle: 测试清单\nitems:\n  - name: Foo\n    why: reason\n    license: oss\n    category: 效率\n"
	resp, err := http.Post(ts.URL+"/api/toollist/import", "application/x-yaml", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import = %d, want 200", resp.StatusCode)
	}
	var imp struct {
		Imported bool   `json:"imported"`
		ID       string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&imp); err != nil {
		t.Fatal(err)
	}
	if !imp.Imported || imp.ID != "clitest" {
		t.Fatalf("import result = %+v", imp)
	}
	// Fetch it back by id.
	resp2, err := http.Get(ts.URL + "/api/toollist?toollist=clitest")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("fetch imported toollist = %d, want 200", resp2.StatusCode)
	}
}
