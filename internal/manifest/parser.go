package manifest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseManifest(path string) (*Manifest, error) {
	if path == "" {
		return nil, fmt.Errorf("manifest path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		return parseYAML(data)
	case ".txt":
		return parseTXT(data)
	default:
		m, err := parseYAML(data)
		if err == nil {
			return m, nil
		}
		return parseTXT(data)
	}
}

func parseYAML(data []byte) (*Manifest, error) {
	data = stripBOM(data)

	if len(data) == 0 {
		return &Manifest{}, nil
	}

	var raw struct {
		Settings struct {
			Proxy        string `yaml:"proxy"`
			SkipExisting bool   `yaml:"skip_existing"`
			RetryCount   int    `yaml:"retry_count"`
			RetryDelay   int    `yaml:"retry_delay"`
		} `yaml:"settings"`
		Packages []struct {
			ID       string `yaml:"id"`
			Category string `yaml:"category,omitempty"`
			Optional bool   `yaml:"optional,omitempty"`
		} `yaml:"packages"`
	}

	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	m := &Manifest{
		Settings: Settings{
			Proxy:        raw.Settings.Proxy,
			SkipExisting: raw.Settings.SkipExisting,
			RetryCount:   raw.Settings.RetryCount,
			RetryDelay:   raw.Settings.RetryDelay,
		},
	}

	for _, p := range raw.Packages {
		id := strings.TrimSpace(p.ID)
		m.Packages = append(m.Packages, Package{
			ID:       id,
			Category: p.Category,
			Optional: p.Optional,
		})
	}

	return m, nil
}

func parseTXT(data []byte) (*Manifest, error) {
	data = stripBOM(data)

	m := &Manifest{}
	var currentCategory string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			category := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if category != "" {
				currentCategory = category
			}
			continue
		}

		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}

		if line == "" {
			continue
		}

		if seen[line] {
			continue
		}
		seen[line] = true
		m.Packages = append(m.Packages, Package{
			ID:       line,
			Category: currentCategory,
		})
	}

	return m, scanner.Err()
}

func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}
