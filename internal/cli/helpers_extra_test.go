package cli

import "testing"

func TestFreshnessHelpers(t *testing.T) {
	cases := map[string]string{
		"fresh": "新鲜", "aging": "较久", "stale": "需复核", "unknown": "未知", "": "未知",
	}
	for in, want := range cases {
		if got := freshnessLabel(in); got != want {
			t.Errorf("freshnessLabel(%q)=%q want %q", in, got, want)
		}
	}
	if ageStr(-1) != "—" {
		t.Errorf("ageStr(-1)=%q", ageStr(-1))
	}
	if ageStr(5) != "5 天前" {
		t.Errorf("ageStr(5)=%q", ageStr(5))
	}
}

func TestJoinURLs(t *testing.T) {
	if joinURLs(nil) != "—" {
		t.Error("empty should be —")
	}
	if got := joinURLs([]string{"a", "b"}); got != "a  ·  b" {
		t.Errorf("joinURLs=%q", got)
	}
}

func TestRootRegistersCommands(t *testing.T) {
	cmd := newRootCmd("v", "c", "d")
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"audit", "guide", "toollist", "explain", "llm", "gui", "tui", "install", "uninstall", "list", "status", "version", "preset", "mirror"} {
		if !names[want] {
			t.Errorf("root missing command %q", want)
		}
	}
}
