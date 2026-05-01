package mirror

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var mirrors = map[string]MirrorDef{
	"ustc": {
		Name:            "ustc",
		WingetSourceURL: "https://mirrors.ustc.edu.cn/winget-source",
	},
	"official": {
		Name:            "official",
		WingetSourceURL: "",
	},
}

type MirrorDef struct {
	Name            string
	WingetSourceURL string
}

func Set(name string) error {
	if runtime.GOOS != "windows" {
		return ErrNotOnWindows
	}
	def, ok := mirrors[name]
	if !ok {
		return fmt.Errorf("unsupported mirror: %s", name)
	}

	if def.WingetSourceURL == "" {
		return resetSource()
	}

	removeSource("winget")

	if err := exec.Command("winget", "source", "add", "winget", def.WingetSourceURL).Run(); err != nil {
		if rErr := resetSource(); rErr != nil {
			return fmt.Errorf("add winget source: %w (recovery failed: %w)", err, rErr)
		}
		return fmt.Errorf("add winget source: %w", err)
	}

	return nil
}

func Reset() error {
	if runtime.GOOS != "windows" {
		return ErrNotOnWindows
	}
	return resetSource()
}

func resetSource() error {
	cmd := exec.Command("winget", "source", "reset", "--force")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if isSourceNotFound(string(out)) {
			return nil
		}
		if isAdminRequired(string(out)) {
			return fmt.Errorf("administrator privileges required to reset winget source")
		}
		return fmt.Errorf("winget source reset: %w (output: %s)", err, string(out))
	}
	return nil
}

func removeSource(name string) error {
	cmd := exec.Command("winget", "source", "remove", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if isSourceNotFound(string(out)) {
			return nil
		}
		if isAdminRequired(string(out)) {
			return fmt.Errorf("administrator privileges required to modify winget source")
		}
		return fmt.Errorf("winget source remove: %w (output: %s)", err, string(out))
	}
	return nil
}

func isSourceNotFound(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "0x8a150019") ||
		strings.Contains(lower, "does not exist") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "找不到") ||
		strings.Contains(lower, "不存在")
}

func isAdminRequired(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "管理员") ||
		strings.Contains(lower, "administrator") ||
		strings.Contains(lower, "admin") ||
		strings.Contains(lower, "elevation") ||
		strings.Contains(lower, "access is denied")
}

func Current() string {
	if runtime.GOOS != "windows" {
		return "unknown"
	}
	out, err := exec.Command("winget", "source", "list").Output()
	if err != nil {
		return "unknown"
	}
	if strings.Contains(string(out), "mirrors.ustc.edu.cn/winget-source") {
		return "ustc"
	}
	return "official"
}

func Supported() []string {
	names := make([]string, 0, len(mirrors))
	for name := range mirrors {
		names = append(names, name)
	}
	return names
}

var ErrNotOnWindows = errors.New("mirror management is only supported on Windows")
