// Package gui serves SwiftInstall's local graphical interface: a single-page
// web app embedded in the binary and served on a loopback port, plus a small
// JSON/SSE API over the same manifest, backend and engine the CLI uses.
//
// `sis gui` opens it in the default browser. The design intentionally mirrors
// magpie's calm, minimal single screen: one page, one list, one accent colour.
package gui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/advice"
	"github.com/cgartlab/SwiftInstall/internal/audit"
	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/llm"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/cgartlab/SwiftInstall/internal/preset"
	"github.com/cgartlab/SwiftInstall/internal/toollist"
)

//go:embed assets
var assetsFS embed.FS

// Options configures a GUI session.
type Options struct {
	Manifest     *manifest.Manifest
	Source       string
	Backend      backend.Backend
	Proxy        string
	RetryCount   int
	RetryDelay   int
	SkipExisting bool
	// Addr is the listen address, e.g. "127.0.0.1:0" (loopback, ephemeral port)
	// or "0.0.0.0:8080" (all interfaces, for LAN preview). Empty = loopback.
	Addr string
	// Open asks the server to open the default browser once it is listening.
	Open bool
	// OpenPath is the path opened in the browser (e.g. "/audit"). Default "/".
	OpenPath string
}

type server struct {
	opts Options
	mu   sync.Mutex // serialises install/uninstall runs

	// manMu guards the manifest/source, which can be swapped at runtime when
	// the user loads a preset. Run holds mu; reads take manMu.RLock.
	manMu sync.RWMutex

	// extra backends (npm, github) detected at startup, for cross-source search.
	extra map[string]backend.Backend

	// catMu guards imported, the toollists the user imported this session.
	catMu    sync.RWMutex
	imported map[string]*toollist.Catalog
}

// currentManifest returns the manifest currently loaded in the session.
func (s *server) currentManifest() *manifest.Manifest {
	s.manMu.RLock()
	defer s.manMu.RUnlock()
	return s.opts.Manifest
}

// currentSource returns the display source for the current manifest.
func (s *server) currentSource() string {
	s.manMu.RLock()
	defer s.manMu.RUnlock()
	return s.opts.Source
}

// loadPreset swaps the session's manifest to a saved preset and updates the
// displayed source. Returns the preset's manifest.
func (s *server) loadPreset(name string) (*manifest.Manifest, error) {
	m, err := preset.Load(name)
	if err != nil {
		return nil, err
	}
	s.manMu.Lock()
	s.opts.Manifest = m
	s.opts.Source = "preset: " + name
	s.manMu.Unlock()
	return m, nil
}

// routes builds the GUI HTTP mux. Extracted from Run so tests can exercise the
// real route wiring without binding a network port.
func (s *server) routes(sub fs.FS) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/info", s.handleInfo)
	mux.HandleFunc("/api/outdated", s.handleOutdated)
	mux.HandleFunc("/api/audit", s.handleAudit)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/explain", s.handleExplain)
	mux.HandleFunc("/api/toollists", s.handleToollists)
	mux.HandleFunc("/api/toollist", s.handleToollist)
	mux.HandleFunc("/api/toollist/import", s.handleToollistImport)
	mux.HandleFunc("/api/toollist/export", s.handleToollistExport)
	mux.HandleFunc("/api/llm", s.handleLLM)
	mux.HandleFunc("/api/presets", s.handlePresets)
	mux.HandleFunc("/api/preset", s.handlePreset)
	mux.HandleFunc("/api/run", s.handleRun)
	serve := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			b, err := fs.ReadFile(sub, name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(b)
		}
	}
	mux.HandleFunc("/audit", serve("audit.html"))
	mux.HandleFunc("/toollists", serve("toollists.html"))
	mux.HandleFunc("/swap", serve("swap.html"))
	mux.HandleFunc("/health", serve("health.html"))
	mux.HandleFunc("/share", serve("share.html"))
	return mux
}

// Run starts the GUI server on the configured address (loopback by default),
// optionally opens the browser, and blocks until ctx is cancelled.
func Run(ctx context.Context, opts Options) error {
	srv := &server{opts: opts, extra: backend.ExtraBackends()}

	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return err
	}

	mux := srv.routes(sub)

	addr := opts.Addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听端口失败 (%s): %w", addr, err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	httpSrv := &http.Server{Handler: mux}

	go func() { _ = httpSrv.Serve(ln) }()

	localURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	urls := []string{localURL}
	exposed := isExposed(addr)
	if exposed {
		for _, ip := range lanIPv4s() {
			urls = append(urls, fmt.Sprintf("http://%s:%d/", ip, port))
		}
	}

	fmt.Fprintf(os.Stderr, "\n  SwiftInstall GUI 已启动，可访问地址：\n")
	for _, u := range urls {
		fmt.Fprintf(os.Stderr, "    %s\n", u)
	}
	if exposed {
		fmt.Fprintf(os.Stderr, "\n  注意：已监听非回环地址，同一局域网内的设备均可访问；\n"+
			"  安装/卸载会真实修改本机软件，请仅在可信网络中使用。\n")
	}
	fmt.Fprintf(os.Stderr, "\n  按 Ctrl+C 停止。\n\n")
	if opts.Open {
		openBrowser(localURL + strings.TrimPrefix(opts.OpenPath, "/"))
	}

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx)
}

// isExposed reports whether addr binds beyond the loopback interface.
func isExposed(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	switch host {
	case "", "0.0.0.0", "::":
		return true
	case "127.0.0.1", "localhost", "::1":
		return false
	default:
		return true
	}
}

// lanIPv4s returns the machine's private IPv4 addresses, for printing LAN URLs.
func lanIPv4s() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4.IsPrivate() {
			out = append(out, ip4.String())
		}
	}
	return out
}

// openBrowser opens url in the platform's default browser, best-effort.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", ""}
	default:
		cmd = "xdg-open"
	}
	_ = exec.Command(cmd, append(args, url)...).Start()
}

// pkgView is the JSON shape for one package in /api/state.
type pkgView struct {
	ID       string `json:"id"`
	Category string `json:"category,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Optional bool   `json:"optional"`
}

// requireGET enforces that a read-only handler is only reached via GET.
func requireGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return false
	}
	return true
}

func (s *server) handleState(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	m := s.currentManifest()
	pkgs := make([]pkgView, 0, len(m.Packages))
	for _, p := range m.Packages {
		pkgs = append(pkgs, pkgView{ID: p.ID, Category: p.Category, Kind: p.Kind, Optional: p.Optional})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"backend":  s.opts.Backend.Name(),
		"source":   s.currentSource(),
		"packages": pkgs,
		"settings": map[string]any{
			"skip_existing": s.opts.SkipExisting,
		},
	})
}

// handleStatus returns the installed state of every manifest package. It uses
// the backend's bulk InstalledSet when available so the whole list resolves in
// one call rather than one probe per package.
func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	ctx := r.Context()

	installed := map[string]bool{}
	m := s.currentManifest()
	if provider, ok := s.opts.Backend.(backend.InstalledSetProvider); ok {
		set, err := provider.InstalledSet(ctx)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		for _, p := range m.Packages {
			installed[p.ID] = set[p.ID]
		}
	} else {
		for _, p := range m.Packages {
			ok, err := s.opts.Backend.IsInstalled(ctx, p.ID)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
				return
			}
			installed[p.ID] = ok
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"installed": installed})
}

// handleAudit runs the first-run environment check: scan installed apps against
// the built-in toollist and return replace / add / keep recommendations. Advisory
// only — it never installs or uninstalls anything.
func (s *server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	cat, err := s.catalogByName(r.URL.Query().Get("toollist"))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	rep, err := audit.Run(r.Context(), audit.SystemLister{}, cat)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// handleExplain is the AI long-tail helper: describe an arbitrary app and
// suggest free/open alternatives via the configured LLM.
func (s *server) handleExplain(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	app := strings.TrimSpace(r.URL.Query().Get("app"))
	if app == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "缺少 app 参数"})
		return
	}
	cfg, err := llm.Load()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !cfg.Configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "LLM 未配置",
			"hint":  "先运行 sis llm set --base-url ... --api-key ... --model ...（或在主界面左下『AI / LLM 设置』里填写）。",
		})
		return
	}
	out, err := advice.Explain(r.Context(), llm.NewClient(cfg), app)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app": app, "answer": out, "notice": "AI 生成，仅供参考，请以官方信息为准。",
	})
}

// handleHealth is the GUI view of the CLI's `doctor`: platform, package
// manager, installed-app scan, toollist freshness and LLM status as JSON.
func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	out := map[string]any{"goos": runtime.GOOS, "goarch": runtime.GOARCH}

	if be, err := backend.ForCurrentOS(); err != nil {
		out["package_manager"] = map[string]any{"ok": false, "error": err.Error()}
	} else if derr := be.Detect(); derr != nil {
		out["package_manager"] = map[string]any{"name": be.Name(), "ok": false, "error": derr.Error()}
	} else {
		out["package_manager"] = map[string]any{"name": be.Name(), "ok": true}
	}

	if names, err := (audit.SystemLister{}).Installed(r.Context()); err != nil {
		out["installed"] = map[string]any{"ok": false, "error": err.Error()}
	} else {
		out["installed"] = map[string]any{"ok": true, "count": len(names)}
	}

	names, _ := toollist.EmbeddedNames()
	lists := make([]map[string]any, 0, len(names))
	totalItems, totalFresh, totalStale := 0, 0, 0
	for _, n := range names {
		c, err := toollist.LoadBuiltinNamed(n)
		if err != nil {
			continue
		}
		total, fresh, aging, stale := toollist.FreshnessCounts(c, time.Now())
		totalItems += total
		totalFresh += fresh
		totalStale += stale
		lists = append(lists, map[string]any{"id": c.ID, "title": c.Title, "total": total, "fresh": fresh, "aging": aging, "stale": stale})
	}
	out["toollists"] = lists
	out["toollists_total"] = map[string]any{"lists": len(lists), "items": totalItems, "fresh": totalFresh, "stale": totalStale}

	cfg, _ := llm.Load()
	out["llm"] = map[string]any{"configured": cfg.Configured(), "model": cfg.Model}

	writeJSON(w, http.StatusOK, out)
}

// catalogByName resolves a toollist id: imported (this session) first, then
// built-in; empty id means the shipped default.
func (s *server) catalogByName(id string) (*toollist.Catalog, error) {
	if id == "" {
		return toollist.LoadBuiltin()
	}
	s.catMu.RLock()
	c := s.imported[id]
	s.catMu.RUnlock()
	if c != nil {
		return c, nil
	}
	return toollist.LoadBuiltinNamed(id)
}

func (s *server) addImported(c *toollist.Catalog) {
	s.catMu.Lock()
	if s.imported == nil {
		s.imported = map[string]*toollist.Catalog{}
	}
	s.imported[c.ID] = c
	s.catMu.Unlock()
}

func catalogSummary(c *toollist.Catalog, imported bool) map[string]any {
	pb := 0
	for _, e := range c.Equivalences {
		if e.Playbook != nil {
			pb++
		}
	}
	total, fresh, aging, stale := toollist.FreshnessCounts(c, time.Now())
	m := map[string]any{
		"id": c.ID, "title": c.Title, "items": len(c.Items),
		"equivalences": len(c.Equivalences), "playbooks": pb,
		"updated": c.Updated, "total": total, "fresh": fresh, "aging": aging, "stale": stale,
	}
	if imported {
		m["imported"] = true
	}
	return m
}

// handleToollists lists built-in toollists plus any imported this session.
func (s *server) handleToollists(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	names, err := toollist.EmbeddedNames()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	out := make([]map[string]any, 0, len(names)+2)
	for _, n := range names {
		if c, err := toollist.LoadBuiltinNamed(n); err == nil {
			out = append(out, catalogSummary(c, false))
		}
	}
	s.catMu.RLock()
	for _, c := range s.imported {
		out = append(out, catalogSummary(c, true))
	}
	s.catMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"toollists": out})
}

var toollistIDRe = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N}._-]{0,63}$`)

// safeFilename reduces an id to a conservative filename stem (no quotes,
// slashes or control chars) so it is safe inside Content-Disposition.
func safeFilename(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" {
		out = "toollist"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

// handleToollistImport parses an uploaded toollist YAML and keeps it for this session.
func (s *server) handleToollistImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST required"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	cat, err := toollist.Parse(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if !toollistIDRe.MatchString(cat.ID) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "清单 id 非法：仅允许字母/数字/点/下划线/连字符，且不以符号开头（≤64）"})
		return
	}
	s.addImported(cat)
	writeJSON(w, http.StatusOK, catalogSummary(cat, true))
}

// handleToollist returns the full catalog (items + equivalences) for one toollist.
// writeCatalogError maps "toollist not found" to 404 and everything else to 500.
func writeCatalogError(w http.ResponseWriter, err error) {
	if errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), "未找到") {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
}

func (s *server) handleToollist(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	cat, err := s.catalogByName(r.URL.Query().Get("toollist"))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cat)
}

// handleToollistExport streams a toollist as a shareable YAML file.
func (s *server) handleToollistExport(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	cat, err := s.catalogByName(r.URL.Query().Get("toollist"))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	b, err := toollist.Marshal(cat)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+safeFilename(cat.ID)+`.yaml"`)
	_, _ = w.Write(b)
}

// aiDescription returns a cached or freshly-generated description for a
// package whose source has none (e.g. fonts). It only runs when the user has
// configured an LLM; otherwise it returns false and the empty desc stays.
func aiDescription(ctx context.Context, name, kind string) (string, bool) {
	if desc, ok := readAICache(name); ok {
		return desc, true
	}
	cfg, err := llm.Load()
	if err != nil || !cfg.Configured() {
		return "", false
	}
	cl := llm.NewClient(cfg)
	prompt := fmt.Sprintf("用一句简体中文（不超过 30 字）介绍软件包 %q", name)
	if kind != "" {
		prompt += fmt.Sprintf("（Homebrew %s）", kind)
	}
	prompt += "。只输出介绍内容，不要引号或多余说明。"
	out, err := cl.Complete(ctx, prompt)
	if err != nil || out == "" {
		return "", false
	}
	_ = writeAICache(name, out)
	return out, true
}

func aiCacheDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	dir := filepath.Join(base, "sis", "ai-cache")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func readAICache(name string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(aiCacheDir(), strings.ReplaceAll(name, "/", "_")+".txt"))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

func writeAICache(name, desc string) error {
	p := filepath.Join(aiCacheDir(), strings.ReplaceAll(name, "/", "_")+".txt")
	return os.WriteFile(p, []byte(desc), 0o644)
}

// handleOutdated lists installed packages with updates available.
func (s *server) handleOutdated(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	provider, ok := s.opts.Backend.(backend.OutdatedProvider)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "当前后端不支持检查更新"})
		return
	}
	names, err := provider.Outdated(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"outdated": names})
}

// handleLLM gets or sets the LLM configuration.
//
//	GET  /api/llm          → current config (api key masked)
//	POST /api/llm          body {"base_url","api_key","model"}
func (s *server) handleLLM(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg, err := llm.Load()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"base_url":   cfg.BaseURL,
			"api_key":    cfg.MaskedKey(),
			"model":      cfg.Model,
			"configured": cfg.Configured(),
		})
		return
	}
	if r.Method == http.MethodPost {
		var body struct {
			BaseURL string `json:"base_url"`
			APIKey  string `json:"api_key"`
			Model   string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "无效 JSON"})
			return
		}
		cfg, _ := llm.Load()
		if body.BaseURL != "" {
			cfg.BaseURL = body.BaseURL
		}
		if body.APIKey != "" && !isMasked(body.APIKey) {
			cfg.APIKey = body.APIKey
		}
		if body.Model != "" {
			cfg.Model = body.Model
		}
		if err := llm.Save(cfg); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "configured": cfg.Configured()})
		return
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
}

func isMasked(s string) bool { return strings.Contains(s, "•") }

// handleInfo returns detailed metadata for one package (description, version,
// homepage, update history) for the row-expand UI.
func (s *server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "缺少 name"})
		return
	}
	provider, ok := s.opts.Backend.(backend.InfoProvider)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "当前后端不支持详情"})
		return
	}
	info, err := provider.Info(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	// If the package has no description (common for fonts and some casks),
	// ask the configured LLM to write one and cache it. This is the first
	// "AI in the surface" integration; future ones reuse the same llm config.
	if info.Desc == "" {
		if desc, ok := aiDescription(r.Context(), name, info.Kind); ok {
			info.Desc = desc
			info.AIDesc = true
		}
	}

	writeJSON(w, http.StatusOK, info)
}

// handleSearch queries every available source (Homebrew + npm + github) in
// parallel and returns merged hits tagged with source, plus installed and
// in-manifest flags so the UI can render add buttons.
func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"hits": []any{}})
		return
	}

	type srcBackend struct {
		name string
		be   backend.Backend
	}
	candidates := []srcBackend{{"brew", s.opts.Backend}}
	for name, be := range s.extra {
		candidates = append(candidates, srcBackend{name, be})
	}

	var mu sync.Mutex
	var allHits []backend.SearchHit
	var wg sync.WaitGroup
	for _, c := range candidates {
		searcher, ok := c.be.(backend.Searcher)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(name string, be backend.Backend) {
			defer wg.Done()
			hits, err := searcher.Search(r.Context(), q)
			if err != nil || len(hits) == 0 {
				return
			}
			mu.Lock()
			allHits = append(allHits, hits...)
			mu.Unlock()
		}(c.name, c.be)
	}
	wg.Wait()

	// Cap total results per source to keep the list manageable.
	allHits = capPerSource(allHits, 12)

	installed := map[string]bool{}
	// Homebrew installed set covers brew hits; npm/github IsInstalled per hit.
	if provider, ok := s.opts.Backend.(backend.InstalledSetProvider); ok {
		if set, err := provider.InstalledSet(r.Context()); err == nil {
			for _, h := range allHits {
				if h.Source == "brew" {
					installed[h.Name] = set[h.Name]
				}
			}
		}
	}
	if npmBe, ok := s.extra["npm"]; ok {
		if provider, ok := npmBe.(backend.InstalledSetProvider); ok {
			if set, err := provider.InstalledSet(r.Context()); err == nil {
				for _, h := range allHits {
					if h.Source == "npm" {
						installed[h.Name] = set[h.Name]
					}
				}
			}
		}
	}

	inManifest := map[string]bool{}
	for _, p := range s.currentManifest().Packages {
		inManifest[p.ID] = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"hits":       allHits,
		"installed":  installed,
		"inManifest": inManifest,
	})
}

// capPerSource keeps at most n hits per source.
func capPerSource(hits []backend.SearchHit, n int) []backend.SearchHit {
	count := map[string]int{}
	out := make([]backend.SearchHit, 0, len(hits))
	for _, h := range hits {
		if count[h.Source] >= n {
			continue
		}
		count[h.Source]++
		out = append(out, h)
	}
	return out
}

// handlePresets lists saved preset names.
func (s *server) handlePresets(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	names, err := preset.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"presets": names})
}

// handlePreset loads, saves or deletes a preset.
//
//	GET  /api/preset?name=work        → load preset into the session (returns state)
//	POST /api/preset/save             body {"name","packages":[{id,category,kind,optional}]}
//	POST /api/preset/delete           body {"name"}
func (s *server) handlePreset(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "缺少 name"})
			return
		}
		m, err := s.loadPreset(name)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		pkgs := make([]pkgView, 0, len(m.Packages))
		for _, p := range m.Packages {
			pkgs = append(pkgs, pkgView{ID: p.ID, Category: p.Category, Kind: p.Kind, Optional: p.Optional})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"backend":  s.opts.Backend.Name(),
			"source":   s.currentSource(),
			"packages": pkgs,
		})

	case http.MethodPost:
		action := strings.TrimSpace(r.URL.Query().Get("action"))
		var body struct {
			Name     string           `json:"name"`
			Packages []map[string]any `json:"packages"`
			Pkgs     []pkgView        `json:"pkgs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "无效的 JSON"})
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "缺少 name"})
			return
		}

		switch action {
		case "delete":
			if err := preset.Delete(body.Name); err != nil {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})

		case "save", "":
			// Accept either "packages" (from the browser) or "pkgs".
			src := body.Packages
			if len(src) == 0 {
				for _, p := range body.Pkgs {
					src = append(src, map[string]any{
						"id": p.ID, "category": p.Category, "kind": p.Kind, "optional": p.Optional,
					})
				}
			}
			m := &manifest.Manifest{}
			for _, raw := range src {
				p := manifest.Package{}
				if v, ok := raw["id"].(string); ok {
					p.ID = v
				}
				if v, ok := raw["category"].(string); ok {
					p.Category = v
				}
				if v, ok := raw["kind"].(string); ok {
					p.Kind = v
				}
				if v, ok := raw["optional"].(bool); ok {
					p.Optional = v
				}
				if p.ID != "" {
					m.Packages = append(m.Packages, p)
				}
			}
			if err := preset.Save(body.Name, m); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			// Switch the session to the just-saved preset.
			s.loadPreset(body.Name)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": body.Name, "total": len(m.Packages)})

		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "未知 action"})
		}

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// handleRun streams an install or uninstall batch over Server-Sent Events.
//
//	GET /api/run?action=install&ids=git,gh,figma
//
// Events: start, progress (one per package), done.
func (s *server) handleRun(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	if action != "install" && action != "uninstall" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "action must be install or uninstall"})
		return
	}

	ids := map[string]bool{}
	for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids[id] = true
		}
	}
	if len(ids) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no packages selected"})
		return
	}

	// Build a filtered manifest preserving only the selected packages.
	cur := s.currentManifest()
	filtered := &manifest.Manifest{Settings: cur.Settings}
	for _, p := range cur.Packages {
		if ids[p.ID] {
			filtered.Packages = append(filtered.Packages, p)
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	s.mu.Lock()
	defer s.mu.Unlock()

	eng := engine.New(s.opts.Backend, engine.Options{
		SkipExisting: action == "install" && s.opts.SkipExisting,
		Proxy:        s.opts.Proxy,
		RetryCount:   s.opts.RetryCount,
		RetryDelay:   s.opts.RetryDelay,
	})
	eng.SetBackends(backend.ExtraBackends())
	eng.OnProgress(func(index, total int, res engine.Result) {
		sse(w, flusher, "progress", map[string]any{
			"index":       index,
			"total":       total,
			"id":          res.Package.ID,
			"status":      string(res.Status),
			"error":       firstLine(res.Error),
			"duration_ms": res.Duration.Milliseconds(),
		})
	})

	sse(w, flusher, "start", map[string]any{"action": action, "total": len(filtered.Packages)})

	var summary *engine.Summary
	var runErr error
	if action == "install" {
		summary, runErr = eng.Install(r.Context(), filtered)
	} else {
		summary, runErr = eng.Uninstall(r.Context(), filtered)
	}

	if runErr != nil && summary == nil {
		sse(w, flusher, "done", map[string]any{"error": runErr.Error()})
		return
	}
	sse(w, flusher, "done", map[string]any{
		"total":     summary.Total,
		"succeeded": summary.Succeeded,
		"skipped":   summary.Skipped,
		"failed":    summary.Failed,
		"duration":  summary.Duration.Round(time.Millisecond).String(),
	})
}

// sse writes one Server-Sent Event and flushes it.
func sse(w http.ResponseWriter, f http.Flusher, event string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	f.Flush()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
