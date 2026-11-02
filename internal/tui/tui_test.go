package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	tea "github.com/charmbracelet/bubbletea"
)

// fakeBackend is an inert backend used only to satisfy non-nil checks in tests.
type fakeBackend struct{}

func (fakeBackend) Name() string                                      { return "fake" }
func (fakeBackend) Detect() error                                     { return nil }
func (fakeBackend) IsInstalled(context.Context, string) (bool, error) { return false, nil }
func (fakeBackend) Install(context.Context, string, backend.InstallOptions) (*backend.Output, error) {
	return &backend.Output{}, nil
}
func (fakeBackend) Uninstall(context.Context, string) (*backend.Output, error) {
	return &backend.Output{}, nil
}

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{Packages: []manifest.Package{
		{ID: "git", Category: "dev"},
		{ID: "node", Category: "dev"},
		{ID: "slack", Category: "apps"},
		{ID: "fig", Category: "apps", Optional: true},
	}}
}

func newTestModel(be backend.Backend) *model {
	return newModel(Options{
		Manifest: testManifest(),
		Source:   "sis.yaml",
		Backend:  be,
		NoColor:  true,
	})
}

// ks builds a tea.KeyMsg for a symbolic or single-rune key.
func ks(s string) tea.KeyMsg {
	switch s {
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// press feeds one key through Update and returns the concrete model.
func press(m *model, key string) (*model, tea.Cmd) {
	got, cmd := m.Update(ks(key))
	return got.(*model), cmd
}

func cursorID(m *model) string {
	if p := m.current(); p != nil {
		return p.pkg.ID
	}
	return ""
}

// ---------------------------------------------------------------------------
// Grouping & navigation
// ---------------------------------------------------------------------------

func TestBuildRowsGroupsByCategory(t *testing.T) {
	m := newTestModel(nil)

	// Expected layout: header(dev), git, node, spacer, header(apps), slack, fig.
	if len(m.rows) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(m.rows))
	}
	if m.rows[0].kind != rowHeader || m.rows[0].category != "dev" {
		t.Errorf("row 0 = %+v, want dev header", m.rows[0])
	}
	if m.rows[3].kind != rowSpacer {
		t.Errorf("row 3 = %+v, want spacer", m.rows[3])
	}
	if m.rows[4].kind != rowHeader || m.rows[4].category != "apps" {
		t.Errorf("row 4 = %+v, want apps header", m.rows[4])
	}
	// Packages belong to the right buckets, in manifest order.
	if got := m.rows[1].pkgIndex; got != 0 {
		t.Errorf("row 1 pkgIndex = %d, want 0", got)
	}
	if got := m.rows[2].pkgIndex; got != 1 {
		t.Errorf("row 2 pkgIndex = %d, want 1", got)
	}
	if got := m.rows[5].pkgIndex; got != 2 {
		t.Errorf("row 5 pkgIndex = %d, want 2", got)
	}
	if got := m.rows[6].pkgIndex; got != 3 {
		t.Errorf("row 6 pkgIndex = %d, want 3", got)
	}
}

func TestNewModelSelectsAllAndFocusesFirst(t *testing.T) {
	m := newTestModel(nil)
	if m.selectedCount() != len(m.packages) {
		t.Errorf("selected = %d, want %d", m.selectedCount(), len(m.packages))
	}
	if got := cursorID(m); got != "git" {
		t.Errorf("cursor = %q, want git", got)
	}
	if m.loading {
		t.Error("loading should be false with nil backend")
	}
}

func TestMoveCursorSkipsHeadersAndSpacers(t *testing.T) {
	m := newTestModel(nil)

	if got := cursorID(m); got != "git" {
		t.Fatalf("start cursor = %q, want git", got)
	}
	m, _ = press(m, "down")
	if got := cursorID(m); got != "node" {
		t.Errorf("after down = %q, want node", got)
	}
	// Next down must skip the spacer + apps header straight to slack.
	m, _ = press(m, "down")
	if got := cursorID(m); got != "slack" {
		t.Errorf("after second down = %q, want slack", got)
	}
	m, _ = press(m, "down")
	if got := cursorID(m); got != "fig" {
		t.Errorf("after third down = %q, want fig", got)
	}
	// At the end, further downs are a no-op.
	m, _ = press(m, "down")
	if got := cursorID(m); got != "fig" {
		t.Errorf("past end = %q, want fig", got)
	}
	m, _ = press(m, "up")
	if got := cursorID(m); got != "slack" {
		t.Errorf("after up = %q, want slack", got)
	}
	// k works like up.
	m, _ = press(m, "k")
	if got := cursorID(m); got != "node" {
		t.Errorf("after k = %q, want node", got)
	}
}

func TestHomeEndJumpToFirstLastPackage(t *testing.T) {
	m := newTestModel(nil)
	m, _ = press(m, "G")
	if got := cursorID(m); got != "fig" {
		t.Errorf("G = %q, want fig", got)
	}
	m, _ = press(m, "g")
	if got := cursorID(m); got != "git" {
		t.Errorf("g = %q, want git", got)
	}
}

func TestBuildRowsPreservesCursor(t *testing.T) {
	m := newTestModel(nil)
	m, _ = press(m, "down") // node
	m.buildRows()
	if got := cursorID(m); got != "node" {
		t.Errorf("cursor after rebuild = %q, want node", got)
	}
}

// ---------------------------------------------------------------------------
// Selection
// ---------------------------------------------------------------------------

func TestToggleSelection(t *testing.T) {
	m := newTestModel(nil)
	git := m.packages[0]

	m, _ = press(m, "space")
	if git.selected {
		t.Error("space should deselect git")
	}
	m, _ = press(m, "space")
	if !git.selected {
		t.Error("space again should reselect git")
	}
}

func TestSelectAllNone(t *testing.T) {
	m := newTestModel(nil)

	m, _ = press(m, "a")
	if got := m.selectedCount(); got != 0 {
		t.Errorf("after first 'a' selected = %d, want 0", got)
	}
	m, _ = press(m, "a")
	if got := m.selectedCount(); got != len(m.packages) {
		t.Errorf("after second 'a' selected = %d, want %d", got, len(m.packages))
	}
}

func TestSelectedPackages(t *testing.T) {
	m := newTestModel(nil)
	m.packages[1].selected = false // drop node
	m.packages[3].selected = false // drop fig

	got := m.selectedPackages()
	if len(got) != 2 {
		t.Fatalf("selected packages = %d, want 2", len(got))
	}
	ids := []string{got[0].ID, got[1].ID}
	if ids[0] != "git" || ids[1] != "slack" {
		t.Errorf("selected ids = %v, want [git slack]", ids)
	}
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

func TestInstalledCount(t *testing.T) {
	m := newTestModel(nil)
	m.packages[0].known, m.packages[0].installed = true, true
	m.packages[1].known, m.packages[1].installed = true, false
	// packages[2] stays unknown and must not count.
	if got := m.installedCount(); got != 1 {
		t.Errorf("installedCount = %d, want 1", got)
	}
}

func TestProbeMsgUpdatesStatus(t *testing.T) {
	m := newTestModel(&fakeBackend{})

	got, _ := m.Update(probeMsg{index: 2, installed: true})
	m = got.(*model)

	if !m.packages[2].known || !m.packages[2].installed {
		t.Error("probeMsg should mark package 2 installed & known")
	}
	if m.probed != 1 {
		t.Errorf("probed = %d, want 1", m.probed)
	}

	got, _ = m.Update(probeDoneMsg{})
	m = got.(*model)
	if m.loading {
		t.Error("probeDoneMsg should clear loading")
	}
}

// ---------------------------------------------------------------------------
// Batch progress / summary
// ---------------------------------------------------------------------------

func TestProgressUpdatesResultAndStatus(t *testing.T) {
	m := newTestModel(&fakeBackend{})
	m.mode = modeRunning
	m.action = "install"

	got, _ := m.Update(progressMsg{
		index: 0, total: 2,
		result: engine.Result{Package: manifest.Package{ID: "git"}, Status: engine.StatusSuccess},
	})
	m = got.(*model)

	if len(m.runResults) != 1 || m.runDone != 1 {
		t.Errorf("results=%d done=%d, want 1/1", len(m.runResults), m.runDone)
	}
	if !m.packages[0].installed {
		t.Error("successful install should mark package installed")
	}
}

func TestRunFinishedShowsSummary(t *testing.T) {
	m := newTestModel(&fakeBackend{})
	m.mode = modeRunning
	m.action = "install"

	summary := &engine.Summary{Total: 2, Succeeded: 1, Skipped: 1, Duration: 1500 * time.Millisecond}
	got, _ := m.Update(runFinishedMsg(runOutcome{summary: summary, action: "install"}))
	m = got.(*model)

	if m.mode != modeDone {
		t.Errorf("mode = %v, want modeDone", m.mode)
	}
	if m.summary != summary {
		t.Error("summary not stored")
	}

	// Enter returns to browse.
	m, _ = press(m, "enter")
	if m.mode != modeBrowse {
		t.Errorf("mode after enter = %v, want modeBrowse", m.mode)
	}
	if m.summary != nil {
		t.Error("summary should be cleared after returning to browse")
	}
}

func TestInstallWithNoSelectionIsNoop(t *testing.T) {
	m := newTestModel(&fakeBackend{})
	m, _ = press(m, "a") // clear all
	if m.selectedCount() != 0 {
		t.Fatal("expected no selection")
	}
	m, cmd := press(m, "i")
	if cmd != nil {
		t.Error("install with no selection should return nil cmd")
	}
	if m.mode != modeBrowse {
		t.Errorf("mode = %v, want modeBrowse", m.mode)
	}
	if m.notice == "" {
		t.Error("expected a notice explaining nothing was selected")
	}
}

func TestCancelDuringRunQuits(t *testing.T) {
	m := newTestModel(&fakeBackend{})
	m.mode = modeRunning
	cancelled := false
	m.cancel = func() { cancelled = true }

	_, cmd := press(m, "ctrl+c")
	if cmd == nil {
		t.Error("ctrl+c during a run should return quit cmd")
	}
	if !cancelled {
		t.Error("ctrl+c should cancel the running batch")
	}
}

// ---------------------------------------------------------------------------
// Utilities & rendering
// ---------------------------------------------------------------------------

func TestTruncateWidth(t *testing.T) {
	if got := truncateWidth("hello", 10); got != "hello" {
		t.Errorf("truncateWidth(hello,10) = %q", got)
	}
	got := truncateWidth("hello world", 8)
	if lipglossWidth(got) > 8 {
		t.Errorf("truncateWidth result width = %d, want <= 8 (%q)", lipglossWidth(got), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated string should end in ellipsis: %q", got)
	}
	if got := truncateWidth("abc", 0); got != "" {
		t.Errorf("truncateWidth(_,0) = %q, want empty", got)
	}
}

func TestViewRendersAtVariousWidths(t *testing.T) {
	for _, dims := range [][2]int{{80, 24}, {40, 12}, {24, 8}, {12, 6}, {5, 4}} {
		m := newTestModel(nil)
		m.width, m.height = dims[0], dims[1]
		out := m.View()
		if out == "" {
			t.Errorf("empty view at %dx%d", dims[0], dims[1])
		}
	}
}

func TestViewRunningAndDone(t *testing.T) {
	m := newTestModel(&fakeBackend{})
	m.width, m.height = 80, 24
	m.mode = modeRunning
	m.action = "install"
	m.runTotal = 2
	m.runResults = []engine.Result{{Package: manifest.Package{ID: "git"}, Status: engine.StatusSuccess}}
	if out := m.View(); !strings.Contains(out, "Installing") {
		t.Errorf("running view missing title: %q", out)
	}

	m.mode = modeDone
	m.summary = &engine.Summary{Total: 2, Succeeded: 1, Skipped: 1, Duration: time.Second}
	if out := m.View(); !strings.Contains(out, "Done") {
		t.Errorf("done view missing summary: %q", out)
	}
}

func TestQuitKey(t *testing.T) {
	m := newTestModel(nil)
	_, cmd := press(m, "q")
	if cmd == nil {
		t.Error("q should return a quit command")
	}
}
