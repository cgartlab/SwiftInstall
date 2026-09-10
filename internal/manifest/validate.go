package manifest

import (
	"fmt"
	"strings"
)

// Validate checks a parsed Manifest for logical correctness.
// Returns a descriptive error if any issue is found.
// Duplicate IDs are not an error — they are collapsed by ParseManifest/Dedupe.
func Validate(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("manifest is nil")
	}

	if len(m.Packages) == 0 {
		return fmt.Errorf("manifest contains no packages")
	}

	for i, pkg := range m.Packages {
		if strings.TrimSpace(pkg.ID) == "" {
			return fmt.Errorf("package #%d has an empty ID", i+1)
		}
	}

	if m.Settings.RetryCount < 0 {
		return fmt.Errorf("retry_count must be >= 0, got %d", m.Settings.RetryCount)
	}
	if m.Settings.RetryDelay < 0 {
		return fmt.Errorf("retry_delay must be >= 0, got %d", m.Settings.RetryDelay)
	}

	return nil
}
