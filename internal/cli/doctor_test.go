package cli

import (
	"testing"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/toollist"
)

func TestSummarizeFreshness(t *testing.T) {
	names, err := toollist.EmbeddedNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no builtin toollists")
	}
	// Use a fixed "now" just after the data's verified dates so everything reads fresh.
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	total, fresh, aging, stale := summarizeFreshness(names, now)
	if total == 0 {
		t.Fatal("total should be > 0")
	}
	if fresh != total || aging != 0 || stale != 0 {
		t.Errorf("expected all fresh, got total=%d fresh=%d aging=%d stale=%d", total, fresh, aging, stale)
	}

	// Far future → everything stale.
	_, f2, a2, s2 := summarizeFreshness(names, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if s2 != total || f2 != 0 || a2 != 0 {
		t.Errorf("expected all stale, got fresh=%d aging=%d stale=%d", f2, a2, s2)
	}
}
