package backend

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// GithubBackend installs release binaries from GitHub repos (e.g.
// yetone/magpie) into ~/.local/bin. The package ID is "owner/repo"; the
// Asset field names the release asset (or substring) to download, with
// optional {os} and {arch} placeholders. A GitHub download proxy
// (SIS_GH_PROXY) is honoured when set.
type GithubBackend struct {
	installDir string
	ready      bool
}

func NewGithubBackend() *GithubBackend { return &GithubBackend{} }

func (g *GithubBackend) Name() string { return "github" }

func (g *GithubBackend) Detect() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建 %s 失败: %w", dir, err)
	}
	g.installDir = dir
	g.ready = true
	return nil
}

// IsInstalled checks whether the repo's binary (basename of repo) exists in
// the install dir and is executable.
func (g *GithubBackend) IsInstalled(ctx context.Context, id string) (bool, error) {
	if !g.ready {
		return false, errors.New("github backend not initialized")
	}
	bin := g.binName(id)
	p := filepath.Join(g.installDir, bin)
	info, err := os.Stat(p)
	if err != nil {
		return false, nil
	}
	return !info.IsDir(), nil
}

// InstalledSet reports repos whose binary is present.
func (g *GithubBackend) InstalledSet(ctx context.Context) (map[string]bool, error) {
	return map[string]bool{}, nil // not meaningful for github; per-repo check used
}

func (g *GithubBackend) Install(ctx context.Context, id string, opts InstallOptions) (*Output, error) {
	if !g.ready {
		return nil, errors.New("github backend not initialized")
	}
	if opts.Asset == "" {
		return nil, fmt.Errorf("github 包 %q 缺少 asset 字段（release 资源名）", id)
	}
	start := time.Now()

	asset := strings.ReplaceAll(opts.Asset, "{os}", runtime.GOOS)
	asset = strings.ReplaceAll(asset, "{arch}", runtime.GOARCH)

	// Resolve the asset download URL from the latest release.
	browserURL, err := g.resolveAssetURL(ctx, id, asset)
	if err != nil {
		return nil, err
	}
	dlURL := withProxy(browserURL)

	tmp, err := os.CreateTemp("", "sis-github-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 %s: %w", dlURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败: HTTP %d (%s)", resp.StatusCode, dlURL)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("写入临时文件: %w", err)
	}
	tmp.Close()

	bin := g.binName(id)
	dest := filepath.Join(g.installDir, bin)

	// Archive or single binary?
	switch {
	case strings.HasSuffix(asset, ".zip"):
		if err := extractBinaryFromZip(tmpPath, dest, bin); err != nil {
			return nil, err
		}
	case strings.HasSuffix(asset, ".tar.gz") || strings.HasSuffix(asset, ".tgz"):
		if err := extractBinaryFromTarGz(tmpPath, dest, bin); err != nil {
			return nil, err
		}
	default:
		// Plain binary: copy directly.
		if err := copyFile(tmpPath, dest); err != nil {
			return nil, err
		}
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return nil, err
	}
	return &Output{Stdout: dest, DurationMs: time.Since(start).Milliseconds()}, nil
}

func (g *GithubBackend) Uninstall(ctx context.Context, id string) (*Output, error) {
	if !g.ready {
		return nil, errors.New("github backend not initialized")
	}
	p := filepath.Join(g.installDir, g.binName(id))
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return &Output{}, nil
}

// binName is the binary filename for a repo id ("owner/repo" → repo basename,
// plus .exe on Windows).
func (g *GithubBackend) binName(id string) string {
	name := id
	if i := strings.LastIndex(id, "/"); i >= 0 {
		name = id[i+1:]
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// resolveAssetURL fetches the latest release and finds an asset whose name
// contains the requested substring (after placeholder expansion).
func (g *GithubBackend) resolveAssetURL(ctx context.Context, repo, substr string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("查询 release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("查询 release 失败: HTTP %d", resp.StatusCode)
	}
	var rel struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	for _, a := range rel.Assets {
		if strings.Contains(a.Name, substr) {
			return a.BrowserDownloadURL, nil
		}
	}
	if len(rel.Assets) > 0 {
		return rel.Assets[0].BrowserDownloadURL, nil
	}
	return "", fmt.Errorf("release 中没有匹配 %q 的资源", substr)
}

// Search queries GitHub for repositories matching query, returning the top
// results by stars. The hit Name is owner/repo, suitable for use as a
// github-source package id.
func (g *GithubBackend) Search(ctx context.Context, query string) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	u := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=15", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github search: HTTP %d", resp.StatusCode)
	}
	var res struct {
		Items []struct {
			FullName    string `json:"full_name"`
			Description string `json:"description"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("parse github search: %w", err)
	}
	hits := make([]SearchHit, 0, len(res.Items))
	for _, it := range res.Items {
		hits = append(hits, SearchHit{Name: it.FullName, Source: "github", Desc: it.Description})
	}
	return hits, nil
}

// withProxy prefixes the GitHub download proxy (SIS_GH_PROXY) when set, so
// release downloads are accelerated in China.
func withProxy(url string) string {
	proxy := os.Getenv("SIS_GH_PROXY")
	if proxy == "" || !strings.HasPrefix(url, "https://github.com/") {
		return url
	}
	return strings.TrimRight(proxy, "/") + "/" + url
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func extractBinaryFromZip(src, dest, bin string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("解压 zip: %w", err)
	}
	defer r.Close()
	// Prefer an entry matching the bin name; else the first executable-looking file.
	var match *zip.File
	for i := range r.File {
		f := r.File[i]
		name := filepath.Base(f.Name)
		if name == bin {
			match = r.File[i]
			break
		}
		if match == nil && !strings.Contains(f.Name, "/") {
			match = r.File[i]
		}
	}
	if match == nil {
		return fmt.Errorf("zip 中未找到二进制 %s", bin)
	}
	return saveZipEntry(match, dest)
}

func saveZipEntry(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func extractBinaryFromTarGz(src, dest, bin string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("解压 gzip: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("tar 中未找到二进制 %s", bin)
		}
		if err != nil {
			return err
		}
		if filepath.Base(h.Name) == bin {
			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			return err
		}
	}
}
