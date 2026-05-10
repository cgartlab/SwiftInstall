package backend

import (
	"os"
	"runtime"
)

func NewBackend() Backend {
	if forced := os.Getenv("SIS_BACKEND"); forced != "" {
		switch forced {
		case "winget":
			return NewWingetBackend()
		}
	}
	switch runtime.GOOS {
	case "windows":
		return NewWingetBackend()
	default:
		return NewWingetBackend()
	}
}
