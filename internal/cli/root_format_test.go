package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func tempManifest(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "m.yaml")
	if err := os.WriteFile(p, []byte("packages:\n  - id: git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInvalidFormatRejected(t *testing.T) {
	p := tempManifest(t)
	cmd := newRootCmd("v", "c", "d")
	cmd.SetArgs([]string{"list", "-f", p, "--format", "xml"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --format, got nil")
	}
}

func TestValidFormatAccepted(t *testing.T) {
	p := tempManifest(t)
	cmd := newRootCmd("v", "c", "d")
	cmd.SetArgs([]string{"list", "-f", p, "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("valid --format json should succeed, got %v", err)
	}
}

func TestRejectExtraArgs(t *testing.T) {
	p := tempManifest(t)
	for _, sub := range []string{"install", "uninstall"} {
		cmd := newRootCmd("v", "c", "d")
		cmd.SetArgs([]string{sub, "-f", p, "bogus-extra"})
		if err := cmd.Execute(); err == nil {
			t.Errorf("%s with extra arg should error, got nil", sub)
		}
	}
}
