// Package audit implements SwiftInstall's first-run environment check:
// it scans which applications are installed and, against a curated toollist,
// classifies each into "replace" (a better free/open alternative exists),
// "add" (recommended by the toollist but not installed), or "keep".
//
// Judgements are based on verifiable dimensions (licence, free-tier scope,
// maintenance, privacy, exportable data) — never a subjective "this app is
// bad". The scan is local; the installed list never leaves the machine.
package audit

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/toollist"
)

// Lister returns the raw names/tokens of installed applications.
// Implementations may scan app bundles and/or ask a package manager.
type Lister interface {
	Installed(ctx context.Context) ([]string, error)
}

// SystemLister scans the running machine: standard macOS app folders plus
// Homebrew tokens. Missing folders are ignored, so it degrades gracefully.
type SystemLister struct {
	// Roots overrides the default application folders (mainly for tests).
	Roots []string
}

func defaultRoots() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/Applications",
		"/Applications/Utilities",
		"/System/Applications",
		"/System/Applications/Utilities",
		filepath.Join(home, "Applications"),
	}
}

// Installed implements Lister.
func (s SystemLister) Installed(ctx context.Context) ([]string, error) {
	set := map[string]bool{}

	roots := s.Roots
	if len(roots) == 0 {
		roots = defaultRoots()
	}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // ignore unreadable dirs
			}
			name := d.Name()
			// macOS .app bundles are directories: record the app and do NOT
			// descend into the bundle (avoids counting helper/embedded apps).
			if strings.HasSuffix(name, ".app") {
				set[strings.TrimSuffix(name, ".app")] = true
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				rel, _ := filepath.Rel(root, path)
				if len(strings.Split(rel, string(os.PathSeparator))) > 2 {
					return filepath.SkipDir
				}
			}
			return nil
		})
	}

	// Homebrew tokens (casks + formulae) improve matching for CLI tools and
	// for apps installed via brew.
	if _, err := exec.LookPath("brew"); err == nil {
		if out, err := exec.CommandContext(ctx, "brew", "list").Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					set[line] = true
				}
			}
		}
	}

	res := make([]string, 0, len(set))
	for k := range set {
		res = append(res, k)
	}
	sort.Strings(res)
	return res, nil
}

// Recommendation is one actionable item in the report.
type Recommendation struct {
	Kind           string             `json:"kind"` // replace | add
	From           string             `json:"from,omitempty"`
	Name           string             `json:"name"`
	Brew           string             `json:"brew,omitempty"`
	License        string             `json:"license"`
	Role           string             `json:"role,omitempty"`
	Category       string             `json:"category,omitempty"`
	FreeTierEnough bool               `json:"free_tier_enough"`
	Why            string             `json:"why,omitempty"`
	Sources        []string           `json:"sources,omitempty"`
	Playbook       *toollist.Playbook `json:"playbook,omitempty"`
	VerifiedAt     string             `json:"verified_at,omitempty"`
	Stale          string             `json:"stale,omitempty"`
}

// Report is the result of an audit run.
type Report struct {
	CatalogID       string           `json:"catalog_id"`
	CatalogTitle    string           `json:"catalog_title"`
	InstalledCount  int              `json:"installed_count"`
	Replaces        []Recommendation `json:"replaces"`
	Adds            []Recommendation `json:"adds"`
	Keeps           []string         `json:"keeps"`
	StaleCount      int              `json:"stale_count"`
	UnmatchedCount  int              `json:"unmatched_count"`
	UnmatchedSample []string         `json:"unmatched_sample,omitempty"`
}

// Normalize lowercases and strips everything but letters/digits/CJK so that
// "Visual Studio Code.app", "visual-studio-code" and "Visual Studio Code"
// all collapse to the same key.
func Normalize(s string) string {
	low := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".app")
	var b strings.Builder
	for _, r := range low {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 0x4e00 && r <= 0x9fff: // CJK
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Run performs the audit against the given catalog.
func Run(ctx context.Context, l Lister, cat *toollist.Catalog) (*Report, error) {
	installed, err := l.Installed(ctx)
	if err != nil {
		return nil, err
	}
	norm := make(map[string]bool, len(installed))
	for _, n := range installed {
		norm[Normalize(n)] = true
	}

	rep := &Report{
		CatalogID:       cat.ID,
		CatalogTitle:    cat.Title,
		InstalledCount:  len(installed),
		Replaces:        []Recommendation{},
		Adds:            []Recommendation{},
		Keeps:           []string{},
		UnmatchedSample: []string{},
	}

	// Replacements: an installed original with a free/open equivalent that is
	// not itself already installed.
	for _, eq := range cat.Equivalences {
		matched := ""
		for _, f := range eq.From {
			if f != "" && norm[Normalize(f)] {
				matched = f
				break
			}
		}
		if matched == "" {
			continue
		}
		if norm[Normalize(eq.To.Name)] {
			continue
		}
		if eq.To.Brew != "" && norm[Normalize(eq.To.Brew)] {
			continue
		}
		rep.Replaces = append(rep.Replaces, Recommendation{
			Kind: "replace", From: matched, Name: eq.To.Name, Brew: eq.To.Brew,
			License: eq.To.License, Role: eq.FromRole, FreeTierEnough: eq.FreeTierEnough,
			Why: eq.Why, Sources: eq.Sources, Playbook: eq.Playbook,
			VerifiedAt: eq.VerifiedAt, Stale: staleState(eq.VerifiedAt),
		})
	}

	// Toollist items: keep when installed, otherwise recommend adding.
	for _, it := range cat.Items {
		installedItem := norm[Normalize(it.Name)] ||
			(it.Brew != "" && norm[Normalize(it.Brew)])
		if installedItem {
			rep.Keeps = append(rep.Keeps, it.Name)
			continue
		}
		rep.Adds = append(rep.Adds, Recommendation{
			Kind: "add", Name: it.Name, Brew: it.Brew, License: it.License,
			Role: it.Role, Category: it.Category, FreeTierEnough: it.FreeTierEnough, Why: it.Why, Sources: it.Sources,
			VerifiedAt: it.VerifiedAt, Stale: staleState(it.VerifiedAt),
		})
	}

	// Unmatched: installed apps not referenced by the catalog at all.
	known := map[string]bool{}
	for _, it := range cat.Items {
		if it.Name != "" {
			known[Normalize(it.Name)] = true
		}
		if it.Brew != "" {
			known[Normalize(it.Brew)] = true
		}
	}
	for _, eq := range cat.Equivalences {
		for _, f := range eq.From {
			known[Normalize(f)] = true
		}
		if eq.To.Name != "" {
			known[Normalize(eq.To.Name)] = true
		}
		if eq.To.Brew != "" {
			known[Normalize(eq.To.Brew)] = true
		}
	}
	var unmatched []string
	for _, n := range installed {
		if n == "" {
			continue
		}
		if !known[Normalize(n)] {
			unmatched = append(unmatched, n)
		}
	}
	sort.Strings(unmatched)
	rep.UnmatchedCount = len(unmatched)
	if len(unmatched) > 20 {
		unmatched = unmatched[:20]
	}
	rep.UnmatchedSample = unmatched

	sort.Slice(rep.Replaces, func(i, j int) bool { return rep.Replaces[i].From < rep.Replaces[j].From })
	sort.Slice(rep.Adds, func(i, j int) bool { return rep.Adds[i].Name < rep.Adds[j].Name })
	sort.Strings(rep.Keeps)
	for _, r := range rep.Replaces {
		if r.Stale == toollist.FreshnessStale {
			rep.StaleCount++
		}
	}
	for _, r := range rep.Adds {
		if r.Stale == toollist.FreshnessStale {
			rep.StaleCount++
		}
	}
	return rep, nil
}

// FindEquivalence returns the equivalence whose From list matches name
// (case/punctuation-insensitive), or nil if none matches.
func FindEquivalence(cat *toollist.Catalog, name string) *toollist.Equivalence {
	n := Normalize(name)
	if n == "" {
		return nil
	}
	for i := range cat.Equivalences {
		for _, f := range cat.Equivalences[i].From {
			if Normalize(f) == n {
				return &cat.Equivalences[i]
			}
		}
	}
	return nil
}

func staleState(verifiedAt string) string {
	s, _ := toollist.Staleness(verifiedAt, time.Now())
	return s
}
