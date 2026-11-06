package backend

import (
	"runtime"
	"testing"
)

func TestGithubBinName(t *testing.T) {
	g := &GithubBackend{}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	cases := map[string]string{
		"owner/repo":            "repo" + suffix,
		"repo":                  "repo" + suffix,
		"a/b/c":                 "c" + suffix,
		"jesseduffield/lazygit": "lazygit" + suffix,
	}
	for in, want := range cases {
		if got := g.binName(in); got != want {
			t.Errorf("binName(%q) = %q, want %q", in, got, want)
		}
	}
}
