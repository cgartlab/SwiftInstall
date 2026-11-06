package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCRLFAndBOM(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"crlf.txt":  "git\r\ngh\r\n",
		"crlf.yaml": "packages:\r\n  - id: git\r\n  - id: gh\r\n",
		"bom.txt":   "\ufeffgit\ngh\n",
		"bom.yaml":  "\ufeffpackages:\n  - id: git\n",
	}
	for name, content := range cases {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		m, err := ParseManifest(p)
		if err != nil {
			t.Errorf("%s: parse error: %v", name, err)
			continue
		}
		if len(m.Packages) == 0 {
			t.Errorf("%s: no packages", name)
			continue
		}
		for _, pkg := range m.Packages {
			if strings.ContainsAny(pkg.ID, "\r\ufeff") || strings.TrimSpace(pkg.ID) != pkg.ID {
				t.Errorf("%s: dirty id %q", name, pkg.ID)
			}
		}
	}
}
