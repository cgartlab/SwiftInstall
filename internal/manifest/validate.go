package manifest

import (
	"fmt"
	"strings"
)

func Validate(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("manifest is nil")
	}

	seen := make(map[string]bool)
	for _, pkg := range m.Packages {
		if strings.TrimSpace(pkg.ID) == "" {
			return fmt.Errorf("package ID cannot be empty")
		}
		if seen[pkg.ID] {
			return fmt.Errorf("duplicate package ID: %s", pkg.ID)
		}
		seen[pkg.ID] = true
	}

	return nil
}
