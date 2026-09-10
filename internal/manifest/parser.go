package manifest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseManifest reads and parses a manifest file.
//
// A .yaml/.yml extension always forces YAML. For every other extension the
// content decides: a YAML mapping carrying a `packages` or `settings` key is
// parsed as YAML, anything else as a plain line-per-package TXT list. The
// extension alone is not trustworthy — this repo ships YAML manifests named
// .txt (Windows/software_list.txt, macOS/packages.txt).
//
// Duplicate package IDs are collapsed, keeping the first occurrence.
func ParseManifest(path string) (*Manifest, error) {
	if path == "" {
		return nil, fmt.Errorf("manifest path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	var m *Manifest
	switch {
	case ext == ".yaml" || ext == ".yml":
		m, err = parseYAML(data)
	case isYAMLManifest(data):
		m, err = parseYAML(data)
	default:
		m, err = parseTXT(data)
	}
	if err != nil {
		return nil, err
	}

	m.Packages = Dedupe(m.Packages)
	return m, nil
}

// isYAMLManifest reports whether data is a YAML manifest — a mapping that has
// a `packages` or `settings` key — rather than a line-per-package TXT list.
func isYAMLManifest(data []byte) bool {
	text := strings.TrimSpace(string(stripBOM(data)))
	if text == "" || !strings.Contains(text, ":") {
		return false
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		return false
	}

	_, hasPackages := raw["packages"]
	_, hasSettings := raw["settings"]
	return hasPackages || hasSettings
}

// parseYAML supports two YAML formats for user convenience:
//
// Format A (简洁格式, recommended in README):
//
//	proxy: http://127.0.0.1:10809
//	packages:
//	  - id: Git.Git
//
// Format B (嵌套格式, structured):
//
//	settings:
//	  proxy: http://127.0.0.1:10809
//	  skip_existing: true
//	packages:
//	  - id: Git.Git
func parseYAML(data []byte) (*Manifest, error) {
	data = stripBOM(data)

	if len(strings.TrimSpace(string(data))) == 0 {
		return &Manifest{Packages: []Package{}}, nil
	}

	// Use a flexible raw structure that accepts both formats
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	m := &Manifest{Packages: []Package{}}

	// Extract settings — support both top-level and nested "settings" block
	if settings, ok := raw["settings"]; ok {
		settingsBytes, _ := yaml.Marshal(settings)
		yaml.Unmarshal(settingsBytes, &m.Settings)
	}

	// Top-level convenience fields override settings
	if v, ok := raw["proxy"]; ok {
		if s, ok := v.(string); ok {
			m.Settings.Proxy = s
		}
	}
	if v, ok := raw["skip_existing"]; ok {
		if b, ok := v.(bool); ok {
			m.Settings.SkipExisting = b
		}
	}
	if v, ok := raw["retry_count"]; ok {
		if n, ok := v.(int); ok {
			m.Settings.RetryCount = n
		}
	}
	if v, ok := raw["retry_delay"]; ok {
		if n, ok := v.(int); ok {
			m.Settings.RetryDelay = n
		}
	}

	// Extract packages
	if pkgs, ok := raw["packages"]; ok {
		pkgBytes, _ := yaml.Marshal(pkgs)
		var packages []Package
		if err := yaml.Unmarshal(pkgBytes, &packages); err != nil {
			return nil, fmt.Errorf("parse packages: %w", err)
		}
		for i := range packages {
			packages[i].ID = strings.TrimSpace(packages[i].ID)
		}
		m.Packages = packages
	}

	return m, nil
}

// parseTXT parses a plain-text manifest where each line is a package ID.
// Lines starting with # are comments; if the comment has text, it becomes
// the category for subsequent packages. Inline comments (after #) are stripped.
func parseTXT(data []byte) (*Manifest, error) {
	data = stripBOM(data)

	m := &Manifest{Packages: []Package{}}
	var currentCategory string

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Full-line comment → potential category header
		if strings.HasPrefix(line, "#") {
			category := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if category != "" {
				currentCategory = category
			}
			continue
		}

		// Strip inline comment
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		m.Packages = append(m.Packages, Package{
			ID:       line,
			Category: currentCategory,
		})
	}

	return m, scanner.Err()
}

// stripBOM removes a UTF-8 BOM prefix if present.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}
