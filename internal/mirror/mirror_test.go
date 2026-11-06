package mirror

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyRestoreIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	// Windows os.UserConfigDir() reads %APPDATA%, not HOME/XDG_CONFIG_HOME.
	t.Setenv("APPDATA", filepath.Join(dir, "AppData", "Roaming"))
	// Make zshrc exist so sourcing is exercised.
	rc := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(rc, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	done, err := Apply()
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(done) == 0 {
		t.Fatal("apply produced no steps")
	}
	env, _ := EnvFile()
	if _, err := os.Stat(env); err != nil {
		t.Fatalf("env file not created: %v", err)
	}
	data, _ := os.ReadFile(rc)
	if !strings.Contains(string(data), beginMarker) {
		t.Fatal("rc not sourced")
	}
	// Apply again must not duplicate the block.
	_, _ = Apply()
	data2, _ := os.ReadFile(rc)
	if strings.Count(string(data2), beginMarker) != 1 {
		t.Fatalf("block duplicated: %d", strings.Count(string(data2), beginMarker))
	}

	_, err = Restore()
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := os.Stat(env); !os.IsNotExist(err) {
		t.Fatal("env file not removed")
	}
	data3, _ := os.ReadFile(rc)
	if strings.Contains(string(data3), beginMarker) {
		t.Fatal("block not removed on restore")
	}
}

func TestEnvContentHasMirrors(t *testing.T) {
	c := envContent()
	for _, want := range []string{"HOMEBREW_BOTTLE_DOMAIN", "SIS_GH_PROXY", "mirrors.ustc.edu.cn"} {
		if !strings.Contains(c, want) {
			t.Errorf("env missing %s", want)
		}
	}
}
