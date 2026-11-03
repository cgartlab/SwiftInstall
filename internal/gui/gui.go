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
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
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
}

// Run starts the GUI server on the configured address (loopback by default),
// optionally opens the browser, and blocks until ctx is cancelled.
func Run(ctx context.Context, opts Options) error {
	srv := &server{opts: opts}

	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/state", srv.handleState)
	mux.HandleFunc("/api/status", srv.handleStatus)
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
	pkgs := make([]pkgView, 0, len(s.opts.Manifest.Packages))
	for _, p := range s.opts.Manifest.Packages {
		pkgs = append(pkgs, pkgView{ID: p.ID, Category: p.Category, Kind: p.Kind, Optional: p.Optional})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"backend":  s.opts.Backend.Name(),
		"source":   s.opts.Source,
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
	if provider, ok := s.opts.Backend.(backend.InstalledSetProvider); ok {
		set, err := provider.InstalledSet(ctx)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		for _, p := range s.opts.Manifest.Packages {
			installed[p.ID] = set[p.ID]
		}
	} else {
		for _, p := range s.opts.Manifest.Packages {
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
	filtered := &manifest.Manifest{Settings: s.opts.Manifest.Settings}
	for _, p := range s.opts.Manifest.Packages {
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
