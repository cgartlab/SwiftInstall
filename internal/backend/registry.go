package backend

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
)

// PlatformFactory constructs the default backend for one GOOS value.
//
// The shared baseline only defines the registration mechanism; the mapping
// from operating system to package manager lives in platform_defaults.go and
// can be replaced by a platform line (Windows / macOS) via RegisterPlatform.
// This keeps platform-specific decisions out of the shared code path.
type PlatformFactory func() Backend

var (
	registryMu        sync.RWMutex
	platformFactories = map[string]PlatformFactory{}
)

// RegisterPlatform registers (or replaces) the default backend factory for a
// GOOS value. Passing an empty goos or a nil factory is a no-op.
//
// A platform line calls this from init() to install its own backend — for
// example a Windows line can supply an extended winget backend without
// changing any code in the shared baseline.
func RegisterPlatform(goos string, factory PlatformFactory) {
	if goos == "" || factory == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	platformFactories[goos] = factory
}

// PlatformFactories returns the sorted list of GOOS values with a registered
// backend. Useful for diagnostics and for platform lines to assert wiring.
func PlatformFactories() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(platformFactories))
	for goos := range platformFactories {
		out = append(out, goos)
	}
	sort.Strings(out)
	return out
}

// ForGOOS returns the backend registered for an explicit GOOS value.
//
// Keeping the lookup separate from runtime.GOOS lets the shared baseline be
// tested on any host: a test can ask for the darwin backend while running on
// Windows, without needing a second machine.
func ForGOOS(goos string) (Backend, error) {
	registryMu.RLock()
	factory := platformFactories[goos]
	registryMu.RUnlock()
	if factory == nil {
		return nil, fmt.Errorf(
			"当前平台 (%s) 暂不支持：已注册的平台有 %v",
			goos, PlatformFactories())
	}
	return factory(), nil
}

// ForCurrentOS returns the backend registered for the running operating
// system. Call Detect() on the returned backend before use; ForCurrentOS
// only chooses the implementation, it does not verify the package manager
// is installed.
func ForCurrentOS() (Backend, error) {
	return ForGOOS(runtime.GOOS)
}

// ExtraBackends returns non-default backends (npm, github) detected on this
// machine, keyed by their source name. A backend whose Detect() fails is
// omitted, so callers can safely register whatever this returns.
func ExtraBackends() map[string]Backend {
	out := map[string]Backend{}
	if n := NewNpmBackend(); n.Detect() == nil {
		out["npm"] = n
	}
	if g := NewGithubBackend(); g.Detect() == nil {
		out["github"] = g
	}
	return out
}
