package backend

import (
	"fmt"
	"runtime"
)

// ForCurrentOS returns the package-manager backend appropriate for the
// running operating system. macOS uses Homebrew, Windows uses winget.
//
// Call Detect() on the returned backend before use; ForCurrentOS only
// chooses the implementation, it does not verify the package manager is
// installed.
func ForCurrentOS() (Backend, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewHomebrewBackend(), nil
	case "windows":
		return NewWingetBackend(), nil
	default:
		return nil, fmt.Errorf(
			"当前平台 (%s) 暂不支持：SwiftInstall 目前支持 macOS (Homebrew) 与 Windows (winget)",
			runtime.GOOS)
	}
}
