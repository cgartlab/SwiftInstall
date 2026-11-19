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
	"regexp"
	"runtime"
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

// SystemLister scans the running machine for installed applications.
//
// 行为按操作系统划分：
//   - darwin：扫描标准 .app 目录（/Applications 等）并追加 Homebrew token。
//   - windows：扫描标准安装目录（Program Files / Program Files (x86) /
//     %LOCALAPPDATA%\Programs）的顶层应用文件夹，并追加 `winget list` 的
//     包 id 与显示名。
//   - 其它 OS：没有统一的应用目录，目录扫描降级为空，仅依赖包管理器（当前无）。
//
// 缺失的目录与不可用的包管理器都会被忽略，因此在受限环境下优雅降级。
type SystemLister struct {
	// Roots overrides the default application folders (mainly for tests).
	Roots []string
	// wingetList overrides the `winget list` invocation (mainly for tests).
	// When nil the real command is run; it returns the raw stdout bytes.
	// Only used when the effective OS is windows (see isWindows).
	wingetList func(ctx context.Context) ([]byte, error)
	// isWindowsOverride overrides the running-OS check (mainly for tests) so the
	// Windows 目录扫描与 winget token 解析在任何 OS 上都能被确定性测试，
	// 而不会在 CI 的 ubuntu 上被 runtime.GOOS 门控旁路掉。零值时回退到真实 GOOS。
	isWindowsOverride *bool
}

// isWindows reports whether the Windows 语义（Program Files 扫描 + winget token）
// 生效。默认取真实 runtime.GOOS，测试可通过 isWindowsOverride 字段覆盖。
func (s SystemLister) isWindows() bool {
	if s.isWindowsOverride != nil {
		return *s.isWindowsOverride
	}
	return runtime.GOOS == "windows"
}

// darwinRoots 返回 macOS 上常见的应用目录。
func darwinRoots() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/Applications",
		"/Applications/Utilities",
		"/System/Applications",
		"/System/Applications/Utilities",
		filepath.Join(home, "Applications"),
	}
}

// windowsRoots 返回 Windows 上常见的应用安装目录。优先读取标准环境变量，
// 缺失时回退到硬编码的默认路径，保证在环境变量被裁剪的极端环境下仍能扫描。
func windowsRoots() []string {
	var roots []string
	add := func(p string) {
		if p != "" {
			roots = append(roots, p)
		}
	}
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		pf = `C:\Program Files`
	}
	add(pf)
	pfx86 := os.Getenv("ProgramFiles(x86)")
	if pfx86 == "" {
		pfx86 = `C:\Program Files (x86)`
	}
	add(pfx86)
	if lad := os.Getenv("LOCALAPPDATA"); lad != "" {
		add(filepath.Join(lad, "Programs"))
	}
	return roots
}

// defaultRoots returns the application folders for the effective OS.
// windows 参数来自 SystemLister.isWindows()，使注入的 OS 语义（测试）与真实
// GOOS 走同一套选择逻辑。
func defaultRoots(windows bool) []string {
	switch {
	case windows:
		return windowsRoots()
	case runtime.GOOS == "darwin":
		return darwinRoots()
	default:
		// linux/其它：没有统一的应用目录，返回空以优雅降级。
		return nil
	}
}

// Installed implements Lister.
func (s SystemLister) Installed(ctx context.Context) ([]string, error) {
	set := map[string]bool{}

	roots := s.Roots
	if len(roots) == 0 {
		roots = defaultRoots(s.isWindows())
	}

	// .app bundle 扫描（macOS 语义）：所有平台都执行是无害的——非 macOS
	// 系统上几乎不存在 .app 目录，因此不会产生额外结果；同时让同一份目录
	// 契约在测试与边缘环境下保持一致。
	scanAppBundleDirs(roots, set)
	// Windows 额外记录标准安装目录下的顶层应用文件夹（Git、7-Zip、Steam 等）。
	if s.isWindows() {
		scanWindowsProgramDirs(roots, set)
	}

	// 包管理器 token：macOS 用 Homebrew，Windows 用 winget。
	if s.isWindows() {
		s.addWingetTokens(ctx, set)
	} else {
		addBrewTokens(ctx, set)
	}

	res := make([]string, 0, len(set))
	for k := range set {
		res = append(res, k)
	}
	sort.Strings(res)
	return res, nil
}

// scanAppBundleDirs 记录 roots 下出现的 .app bundle（去掉 .app 后缀），
// 且不会进入 bundle 内部（避免把 helper/内嵌应用重复计数）。深层非应用目录
// 会被跳过。缺失或不可读的目录被忽略。
func scanAppBundleDirs(roots []string, set map[string]bool) {
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
}

// scanWindowsProgramDirs 记录 roots 下每个顶层子目录的名字。Windows 上一个
// 安装目录通常对应一个应用（Git、7-Zip、Node.js、Steam、WeChat 等），其名字
// 足够与清单里的显示名做归一化匹配。只取顶层，避免把 bin/ 卸载器等子目录当应用。
func scanWindowsProgramDirs(roots []string, set map[string]bool) {
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue // 缺失或无权限的目录直接跳过
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if name := e.Name(); name != "" {
				set[name] = true
			}
		}
	}
}

// wingetColSep 匹配 winget 表格输出里的列分隔（至少两个空白）。
// Name 列本身可能含空格，因此不能用 strings.Fields。
var wingetColSep = regexp.MustCompile(`\s{2,}`)

// addWingetTokens 运行 `winget list`，把每个已装包的 Id 与显示名加入 set。
// Id 形如 "Git.Git"，用于与清单的 Winget 字段匹配；显示名（如 "Git"）用于与
// 清单的 Name 匹配。winget 不可用或输出无法解析时静默跳过（目录扫描仍有效）。
func (s SystemLister) addWingetTokens(ctx context.Context, set map[string]bool) {
	var out []byte
	var err error
	if s.wingetList != nil {
		// 测试注入：跳过 PATH 探测，直接使用注入的输出，使解析逻辑在任意 OS
		// 上都能被确定性验证（CI 的 ubuntu 上没有 winget 可执行文件）。
		out, err = s.wingetList(ctx)
	} else {
		// 真实调用：先确认 winget 在 PATH 上，缺失则静默跳过（非 Windows 环境）。
		if _, e := exec.LookPath("winget.exe"); e != nil {
			if _, e := exec.LookPath("winget"); e != nil {
				return
			}
		}
		out, err = exec.CommandContext(ctx, "winget", "list",
			"--accept-source-agreements", "--disable-interactivity").Output()
	}
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, " \t\r")
		if line == "" || strings.HasPrefix(line, "---") {
			continue // 空行 / 表格分隔线
		}
		cols := wingetColSep.Split(line, -1)
		if len(cols) < 2 {
			continue
		}
		id := strings.TrimSpace(cols[1])
		name := strings.TrimSpace(cols[0])
		if id != "" && !strings.EqualFold(id, "Id") {
			set[id] = true
		}
		if name != "" && !strings.EqualFold(name, "Name") {
			set[name] = true
		}
	}
}

// addBrewTokens 追加 Homebrew 的 formulae + casks token，用于改进 CLI 工具与
// 通过 brew 安装的应用的匹配。brew 不可用时静默跳过。
func addBrewTokens(ctx context.Context, set map[string]bool) {
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
}

// Recommendation is one actionable item in the report.
type Recommendation struct {
	Kind           string             `json:"kind"` // replace | add
	From           string             `json:"from,omitempty"`
	Name           string             `json:"name"`
	Brew           string             `json:"brew,omitempty"`
	Winget         string             `json:"winget,omitempty"` // winget 包 id（Windows）
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
		if eq.To.Winget != "" && norm[Normalize(eq.To.Winget)] {
			continue
		}
		rep.Replaces = append(rep.Replaces, Recommendation{
			Kind: "replace", From: matched, Name: eq.To.Name, Brew: eq.To.Brew,
			Winget: eq.To.Winget,
			License: eq.To.License, Role: eq.FromRole, FreeTierEnough: eq.FreeTierEnough,
			Why: eq.Why, Sources: eq.Sources, Playbook: eq.Playbook,
			VerifiedAt: eq.VerifiedAt, Stale: staleState(eq.VerifiedAt),
		})
	}

	// Toollist items: keep when installed, otherwise recommend adding.
	// 已装判定同时看显示名（Name）与两个包管理器 token（Brew / Winget），
	// 这样 Windows 上的 winget 条目也能参与匹配。
	for _, it := range cat.Items {
		installedItem := norm[Normalize(it.Name)] ||
			(it.Brew != "" && norm[Normalize(it.Brew)]) ||
			(it.Winget != "" && norm[Normalize(it.Winget)])
		if installedItem {
			rep.Keeps = append(rep.Keeps, it.Name)
			continue
		}
		rep.Adds = append(rep.Adds, Recommendation{
			Kind: "add", Name: it.Name, Brew: it.Brew, Winget: it.Winget, License: it.License,
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
		if it.Winget != "" {
			known[Normalize(it.Winget)] = true
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
		if eq.To.Winget != "" {
			known[Normalize(eq.To.Winget)] = true
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
