package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/toollist"
)

// The GUI must never surface a catalog meant for another platform, even when
// a hand-written URL asks for it.
func TestHandleToollists_PlatformFiltered(t *testing.T) {
	rec := httptest.NewRecorder()
	(&server{}).handleToollists(rec, httptest.NewRequest(http.MethodGet, "/api/toollists", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var out struct {
		Toollists []struct {
			ID string `json:"id"`
		} `json:"toollists"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Toollists) == 0 {
		t.Fatal("no catalogs listed")
	}

	all, err := toollist.EmbeddedNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range all {
		c, err := toollist.LoadBuiltinNamed(n)
		if err != nil {
			t.Fatal(err)
		}
		if toollist.AvailableFor(c, runtime.GOOS) {
			continue
		}
		for _, got := range out.Toollists {
			if got.ID == c.ID {
				t.Errorf("cross-platform catalog %q leaked into /api/toollists on %s", c.ID, runtime.GOOS)
			}
		}
	}
}

func TestHandleToollist_RejectsOtherPlatform(t *testing.T) {
	cp := toollist.CurrentPlatform()
	if cp != toollist.PlatformWindows && cp != toollist.PlatformMacOS {
		t.Skipf("platform isolation applies only to windows/darwin, got %s", cp)
	}
	other := "mac-designers"
	if cp == toollist.PlatformMacOS {
		other = "windows-essentials"
	}
	rec := httptest.NewRecorder()
	(&server{}).handleToollist(rec, httptest.NewRequest(http.MethodGet, "/api/toollist?toollist="+other, nil))
	if rec.Code == http.StatusOK {
		t.Errorf("catalog from another platform (%q) was served on %s", other, runtime.GOOS)
	}
}
