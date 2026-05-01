package engine

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"golang.org/x/sys/windows"
)

type CheckStatus string

const (
	CheckPass CheckStatus = "pass"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
)

type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message,omitempty"`
}

type CheckSuite struct {
	Results []CheckResult `json:"results"`
}

func (s *CheckSuite) AllPass() bool {
	for _, r := range s.Results {
		if r.Status == CheckFail {
			return false
		}
	}
	return true
}

type CheckConfig struct {
	ManifestPath string
	Backend      backend.Backend
}

func RunChecks(ctx context.Context, cfg *CheckConfig) *CheckSuite {
	suite := &CheckSuite{}

	suite.Results = append(suite.Results, checkBackend(cfg.Backend))
	suite.Results = append(suite.Results, checkManifest(cfg.ManifestPath))

	if runtime.GOOS == "windows" {
		suite.Results = append(suite.Results, checkAdmin())
	}

	return suite
}

func checkBackend(be backend.Backend) CheckResult {
	if err := be.Detect(); err != nil {
		return CheckResult{
			Name:    "Package Manager",
			Status:  CheckFail,
			Message: fmt.Sprintf("%s not found: %v", be.Name(), err),
		}
	}
	return CheckResult{
		Name:    "Package Manager",
		Status:  CheckPass,
		Message: fmt.Sprintf("%s available", be.Name()),
	}
}

func checkManifest(path string) CheckResult {
	if path == "" {
		return CheckResult{
			Name:    "Manifest",
			Status:  CheckFail,
			Message: "no manifest file specified",
		}
	}
	fi, err := os.Stat(path)
	if err != nil {
		return CheckResult{
			Name:    "Manifest",
			Status:  CheckFail,
			Message: fmt.Sprintf("manifest not found: %s", path),
		}
	}
	if fi.Size() == 0 {
		return CheckResult{
			Name:    "Manifest",
			Status:  CheckWarn,
			Message: fmt.Sprintf("manifest is empty: %s", path),
		}
	}
	return CheckResult{
		Name:    "Manifest",
		Status:  CheckPass,
		Message: fmt.Sprintf("manifest found: %s (%d bytes)", path, fi.Size()),
	}
}

func checkAdmin() CheckResult {
	if runtime.GOOS != "windows" {
		return CheckResult{
			Name:   "Admin Rights",
			Status: CheckPass,
			Message: "not required on this platform",
		}
	}

	if isWindowsAdmin() {
		return CheckResult{
			Name:    "Admin Rights",
			Status:  CheckPass,
			Message: "running with administrator privileges",
		}
	}

	return CheckResult{
		Name:    "Admin Rights",
		Status:  CheckWarn,
		Message: "not running as administrator — some operations may fail",
	}
}

func isWindowsAdmin() bool {
	if os.Getenv("SIS_ADMIN_CHECK") != "" {
		return true
	}
	return isProcessElevated()
}

func isProcessElevated() bool {
	var token windows.Token
	h := windows.CurrentProcess()
	err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	var elevation uint32
	var size uint32
	err = windows.GetTokenInformation(token, windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &size)
	return err == nil && elevation != 0
}
