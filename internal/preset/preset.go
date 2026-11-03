// Package preset persists named software lists ("presets") so a user can keep
// several manifests tuned to their habits — e.g. "work", "design", "minimal" —
// and switch between them. Presets are ordinary manifest YAML files stored under
// the OS config dir (~/Library/Application Support/sis/presets on macOS).
//
// This is the one piece of persistence sis keeps; it is data the user explicitly
// saves, not hidden configuration. There is still no config file governing CLI
// behaviour.
package preset

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"gopkg.in/yaml.v3"
)

var nameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// Dir returns the preset directory, creating it if needed.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		base = filepath.Join(os.Getenv("HOME"), ".config")
	}
	dir := filepath.Join(base, "sis", "presets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建预设目录失败: %w", err)
	}
	return dir, nil
}

// Path returns the file path for a preset name. The name is validated.
func Path(name string) (string, error) {
	if !validName(name) {
		return "", fmt.Errorf("预设名只能包含字母、数字、点、下划线或连字符，且不能以符号开头：%q", name)
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".yaml"), nil
}

// List returns the names of all saved presets, sorted.
func List() ([]string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".yaml") {
			names = append(names, strings.TrimSuffix(n, ".yaml"))
		}
	}
	sort.Strings(names)
	return names, nil
}

// Save writes m as a preset. Existing presets with the same name are overwritten.
func Save(name string, m *manifest.Manifest) error {
	p, err := Path(name)
	if err != nil {
		return err
	}
	if m == nil {
		m = &manifest.Manifest{}
	}
	m.Packages = manifest.Dedupe(m.Packages)

	var buf strings.Builder
	fmt.Fprintf(&buf, "# SwiftInstall preset: %s\n", name)
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("编码预设失败: %w", err)
	}
	buf.Write(data)
	return os.WriteFile(p, []byte(buf.String()), 0o644)
}

// Load reads and parses a preset.
func Load(name string) (*manifest.Manifest, error) {
	p, err := Path(name)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil, fmt.Errorf("预设 %q 不存在", name)
	}
	return manifest.ParseManifest(p)
}

// Delete removes a preset. Missing presets are reported as an error.
func Delete(name string) error {
	p, err := Path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("预设 %q 不存在", name)
		}
		return err
	}
	return nil
}

func validName(name string) bool {
	return name != "" && len(name) <= 64 && nameRe.MatchString(name)
}
