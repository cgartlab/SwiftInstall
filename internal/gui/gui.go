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
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/llm"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
	"github.com/cgartlab/SwiftInstall/internal/preset"
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
}

type server struct {
	opts Options
	mu   sync.Mutex // serialises install/uninstall runs

	// manMu guards the manifest/source, which can be swapped at runtime when
	// the user loads a preset. Run holds mu; reads take manMu.RLock.
	manMu sync.RWMutex

	// extra backends (npm, github) detected at startup, for cross-source search.
	extra map[string]backend.Backend
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

// Run starts the GUI server on the configured address (loopback by default),
// optionally opens the browser, and blocks until ctx is cancelled.
func Run(ctx context.Context, opts Options) error {
	srv := &server{opts: opts, extra: backend.ExtraBackends()}

	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/state", srv.handleState)
	mux.HandleFunc("/api/status", srv.handleStatus)
	mux.HandleFunc("/api/search", srv.handleSearch)
	mux.HandleFunc("/api/info", srv.handleInfo)
	mux.HandleFunc("/api/outdated", srv.handleOutdated)
	mux.HandleFunc("/api/llm", srv.handleLLM)
	mux.HandleFunc("/api/presets", srv.handlePresets)
	mux.HandleFunc("/api/preset", srv.handlePreset)
	mux.HandleFunc("/api/run", srv.handleRun)

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
		openBrowser(localURL)
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

func (s *server) handleState(w http.ResponseWriter, r *http.Request) {
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
