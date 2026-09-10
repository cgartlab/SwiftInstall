package manifest

// Package represents a single software package to install/uninstall.
type Package struct {
	ID       string `yaml:"id" json:"id"`
	Category string `yaml:"category,omitempty" json:"category,omitempty"`
	Optional bool   `yaml:"optional,omitempty" json:"optional,omitempty"`
}

// Settings holds manifest-level configuration that affects installation behavior.
type Settings struct {
	Proxy        string `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	SkipExisting bool   `yaml:"skip_existing,omitempty" json:"skip_existing,omitempty"`
	RetryCount   int    `yaml:"retry_count,omitempty" json:"retry_count,omitempty"`
	RetryDelay   int    `yaml:"retry_delay,omitempty" json:"retry_delay,omitempty"`
}

// Manifest is the top-level structure describing what to install and how.
type Manifest struct {
	Settings Settings  `json:"settings,omitempty"`
	Packages []Package `json:"packages"`
}

// Dedupe removes duplicate packages by ID, keeping the first occurrence.
func Dedupe(packages []Package) []Package {
	seen := make(map[string]bool, len(packages))
	result := make([]Package, 0, len(packages))
	for _, p := range packages {
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		result = append(result, p)
	}
	return result
}
