package toollist

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// Item is one curated software entry in a toollist.
type Item struct {
	Name           string   `yaml:"name" json:"name"`
	Brew           string   `yaml:"brew" json:"brew"` // homebrew cask/formula token, if installable
	Role           string   `yaml:"role" json:"role"`
	Category       string   `yaml:"category" json:"category"`
	License        string   `yaml:"license" json:"license"` // free | oss | freemium | paid
	FreeTierEnough bool     `yaml:"free_tier_enough" json:"free_tier_enough"`
	AlternativeTo  string   `yaml:"alternative_to" json:"alternative_to"`
	Why            string   `yaml:"why" json:"why"`
	VersionNote    string   `yaml:"version_note" json:"version_note"`
	VerifiedAt     string   `yaml:"verified_at" json:"verified_at"`
	Sources        []string `yaml:"sources" json:"sources,omitempty"`
}

// To describes the replacement target of an Equivalence.
type To struct {
	Name    string `yaml:"name" json:"name"`
	Brew    string `yaml:"brew" json:"brew"`
	License string `yaml:"license" json:"license"`
}

// Step is one ordered instruction in a migration playbook.
type Step struct {
	Title     string `yaml:"title" json:"title"`
	Detail    string `yaml:"detail" json:"detail"`
	SourceURL string `yaml:"source_url" json:"source_url"`
}

// Playbook is a curated, version-dated migration guide from one app to another.
// It is advisory only — it tells the user how to move, it never moves their data.
type Playbook struct {
	Summary      string `yaml:"summary" json:"summary"`
	DataFidelity string `yaml:"data_fidelity" json:"data_fidelity"` // what survives / what is lost
	Steps        []Step `yaml:"steps" json:"steps"`
	VerifiedAt   string `yaml:"verified_at" json:"verified_at"`
}

// Equivalence maps one or more installed (usually paid) apps to a better
// free/open alternative.
type Equivalence struct {
	From           []string  `yaml:"from" json:"from"` // names/tokens that identify the original
	FromRole       string    `yaml:"from_role" json:"from_role"`
	To             To        `yaml:"to" json:"to"`
	FreeTierEnough bool      `yaml:"free_tier_enough" json:"free_tier_enough"`
	Why            string    `yaml:"why" json:"why"`
	VerifiedAt     string    `yaml:"verified_at" json:"verified_at"`
	Sources        []string  `yaml:"sources" json:"sources,omitempty"`
	Playbook       *Playbook `yaml:"playbook,omitempty" json:"playbook,omitempty"`
}

// Catalog is a curated toollist plus an equivalence table.
type Catalog struct {
	ID           string        `yaml:"id" json:"id"`
	Title        string        `yaml:"title" json:"title"`
	Audience     string        `yaml:"audience" json:"audience"`
	Curator      string        `yaml:"curator" json:"curator"`
	Updated      string        `yaml:"updated" json:"updated"`
	Standard     []string      `yaml:"standard" json:"standard"`
	Items        []Item        `yaml:"items" json:"items"`
	Equivalences []Equivalence `yaml:"equivalences" json:"equivalences"`
}

// BuiltinName is the id/file name of the default shipped catalog.
const BuiltinName = "mac-designers.yaml"

// LoadBuiltin loads the catalog shipped inside the binary.
func LoadBuiltin() (*Catalog, error) {
	b, err := builtinFS.ReadFile("builtin/" + BuiltinName)
	if err != nil {
		return nil, fmt.Errorf("读取内置书单失败: %w", err)
	}
	return parse(b)
}

// LoadFile loads a catalog from an arbitrary YAML file.
func LoadFile(path string) (*Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parse(b)
}

func parse(b []byte) (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("解析书单失败: %w", err)
	}
	if c.ID == "" {
		return nil, fmt.Errorf("书单缺少 id 字段")
	}
	return &c, nil
}

// Marshal renders a catalog back to YAML, so a toollist can be exported and
// shared. The result round-trips through LoadFile / parse.
func Marshal(c *Catalog) ([]byte, error) {
	return yaml.Marshal(c)
}

// SaveFile writes the catalog to path as YAML.
func SaveFile(c *Catalog, path string) error {
	b, err := Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// EmbeddedNames lists the filenames of the built-in catalogs.
func EmbeddedNames() ([]string, error) {
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// LoadBuiltinNamed loads a built-in catalog by id ("mac-designers") or by
// filename ("mac-designers.yaml").
func LoadBuiltinNamed(name string) (*Catalog, error) {
	fname := name
	if !strings.HasSuffix(fname, ".yaml") {
		fname += ".yaml"
	}
	b, err := builtinFS.ReadFile("builtin/" + fname)
	if err != nil {
		return nil, fmt.Errorf("未找到内置书单 %q: %w", name, err)
	}
	return parse(b)
}

// Parse parses catalog YAML bytes. Exported for callers such as the GUI import.
func Parse(b []byte) (*Catalog, error) { return parse(b) }

// Freshness states for a verified_at date.
const (
	FreshnessFresh   = "fresh"   // recently verified
	FreshnessAging   = "aging"   // getting old, worth a re-check
	FreshnessStale   = "stale"   // needs re-verification
	FreshnessUnknown = "unknown" // missing/unparseable
)

// Freshness thresholds in days.
const (
	freshMaxDays = 180
	agingMaxDays = 365
)

// Staleness reports how old a verified_at value is relative to now and returns
// a state plus the age in days (-1 when the date cannot be parsed).
func Staleness(verifiedAt string, now time.Time) (string, int) {
	t, ok := parseVerified(verifiedAt)
	if !ok {
		return FreshnessUnknown, -1
	}
	days := int(now.Sub(t).Hours() / 24)
	if days < 0 {
		days = 0
	}
	switch {
	case days <= freshMaxDays:
		return FreshnessFresh, days
	case days <= agingMaxDays:
		return FreshnessAging, days
	default:
		return FreshnessStale, days
	}
}

func parseVerified(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// FreshnessCounts tallies items + equivalences in a catalog by freshness state.
func FreshnessCounts(c *Catalog, now time.Time) (total, fresh, aging, stale int) {
	add := func(verifiedAt string) {
		total++
		switch st, _ := Staleness(verifiedAt, now); st {
		case FreshnessFresh:
			fresh++
		case FreshnessAging:
			aging++
		case FreshnessStale:
			stale++
		}
	}
	for _, it := range c.Items {
		add(it.VerifiedAt)
	}
	for i := range c.Equivalences {
		add(c.Equivalences[i].VerifiedAt)
	}
	return
}
