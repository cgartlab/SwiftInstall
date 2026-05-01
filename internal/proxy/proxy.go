package proxy

import (
	"os/exec"
	"runtime"
	"strings"
)

type Info struct {
	Address string
	Running bool
	Source  string
}

func Detect() (Info, error) {
	if runtime.GOOS == "windows" {
		return detectWindows()
	}
	return Info{}, nil
}

func detectWindows() (Info, error) {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq v2rayN.exe", "/NH")
	out, err := cmd.Output()
	if err != nil {
		return Info{}, nil
	}
	if strings.Contains(string(out), "v2rayN.exe") {
		return Info{
			Address: "http://127.0.0.1:10809",
			Running: true,
			Source:  "v2rayN",
		}, nil
	}
	return Info{}, nil
}
