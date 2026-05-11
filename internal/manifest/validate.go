package manifest

import (
	"fmt"
	"strings"
)

// Validate checks a parsed Manifest for logical correctness.
// Returns a descriptive error if any issue is found.
func Validate(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("manifest is nil")
	}

	if len(m.Packages) == 0 {
		return fmt.Errorf("manifest contains no packages")
	}

	seen := make(map[string]bool)
	for i, pkg := range m.Packages {
		id := strings.TrimSpace(pkg.ID)
		if id == "" {
			return fmt.Errorf("package #%d has an empty ID", i+1)
		}
		if seen[id] {
			return fmt.Errorf("duplicate package ID: %s", id)
		}
		seen[id] = true
	}

	if m.Settings.RetryCount < 0 {
		return fmt.Errorf("retry_count must be >= 0, got %d", m.Settings.RetryCount)
	}
	if m.Settings.RetryDelay < 0 {
		return fmt.Errorf("retry_delay must be >= 0, got %d", m.Settings.RetryDelay)
	}

	return nil
}
