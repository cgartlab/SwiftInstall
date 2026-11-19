package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/toollist"
)

// stubWinget 返回一个不触发真实 winget 子进程的 SystemLister（用于测试，
// 保证在 Windows 开机上 `go test` 既快速又不依赖本机装了哪些软件）。
func stubWinget(roots []string) SystemLister {
	return SystemLister{
		Roots:      roots,
		wingetList: func(context.Context) ([]byte, error) { return nil, nil },
	}
}

// winOverride 返回一个指向 true 的指针，用于在任意 OS 上确定性地启用 Windows
// 语义（Program Files 扫描 + winget token），使相关测试在 CI 的 ubuntu 上不被
// runtime.GOOS 门控旁路。
func winOverride() *bool {
	b := true
	return &b
}

func TestSystemLister_TempDir(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Foo.app"))          // app at root
	mustMkdir(t, filepath.Join(root, "Sub", "Bar.app"))   // nested one level
	mustWrite(t, filepath.Join(root, "notes.txt"))        // non-app file, ignored
	mustWrite(t, filepath.Join(root, "Tool", "bin", "x")) // deep non-app, ignored

	got, err := stubWinget([]string{root}).Installed(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, g := range got {
		set[g] = true
	}
	if !set["Foo"] || !set["Bar"] {
		t.Fatalf("expected Foo and Bar, got %v", got)
	}
	if set["notes"] || set["x"] {
		t.Fatalf("non-app entries should be ignored, got %v", got)
	}
}

func TestSystemLister_MissingRootIsIgnored(t *testing.T) {
	got, err := stubWinget([]string{"/definitely/not/here/xyz"}).Installed(context.Background())
	if err != nil {
		t.Fatalf("missing root should not error: %v", err)
	}
	_ = got // may still contain package-manager tokens on a dev machine
}

// TestSystemLister_WindowsProgramDirs 验证 Windows 安装目录扫描：顶层应用文件夹
// 被记录，非目录条目与缺失目录被忽略。（在所有 OS 上都能跑——扫描逻辑本身与
// OS 无关，Windows 之外也能用同样的临时目录验证。）
func TestSystemLister_WindowsProgramDirs(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Git"))
	mustMkdir(t, filepath.Join(root, "7-Zip"))
	mustMkdir(t, filepath.Join(root, "Node.js"))
	mustWrite(t, filepath.Join(root, "uninstall.exe")) // 文件，忽略
	mustWrite(t, filepath.Join(root, "Git", "bin", "git.exe"))

	l := SystemLister{
		Roots:             []string{root},
		isWindowsOverride: winOverride(),
		wingetList:        func(context.Context) ([]byte, error) { return nil, nil },
	}
	got, err := l.Installed(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, g := range got {
		set[g] = true
	}
	for _, want := range []string{"Git", "7-Zip", "Node.js"} {
		if !set[want] {
			t.Errorf("expected top-level app dir %q, got %v", want, got)
		}
	}
	if set["uninstall.exe"] || set["git.exe"] {
		t.Errorf("files/deep entries should be ignored, got %v", got)
	}
}

// fakeWingetTable 模拟 `winget list` 的表格输出（Name / Id / Version ...）。
const fakeWingetTable = `Name                       Id                       Version      Available Source
-----------------------------------------------------------------------------------
Git                        Git.Git                  2.45.0                 winget
7-Zip                      7zip.7zip                24.05                 winget
Tencent WeChat             Tencent.WeChat           3.9.10                 winget
`

// TestSystemLister_WingetTokens 验证 winget 表格解析：Id 与显示名都进入已装集合，
// 表头与分隔线被跳过。通过注入 wingetList，不依赖本机 winget。
func TestSystemLister_WingetTokens(t *testing.T) {
	l := SystemLister{
		isWindowsOverride: winOverride(),
		wingetList: func(context.Context) ([]byte, error) {
			return []byte(fakeWingetTable), nil
		},
	}
	got, err := l.Installed(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, g := range got {
		set[g] = true
	}
	// Id 列
	for _, want := range []string{"Git.Git", "7zip.7zip", "Tencent.WeChat"} {
		if !set[want] {
			t.Errorf("expected winget id %q in set, got %v", want, got)
		}
	}
	// 显示名列（用于按 Name 匹配）
	for _, want := range []string{"Git", "7-Zip", "Tencent WeChat"} {
		if !set[want] {
			t.Errorf("expected winget display name %q in set, got %v", want, got)
		}
	}
	// 表头不应被当成应用
	if set["Name"] || set["Id"] {
		t.Errorf("header should be skipped, got %v", got)
	}
}

// TestSystemLister_WingetUnavailable 测试 winget 不可用/出错时优雅降级（不报错）。
func TestSystemLister_WingetUnavailable(t *testing.T) {
	l := SystemLister{
		isWindowsOverride: winOverride(),
		wingetList:        func(context.Context) ([]byte, error) { return nil, context.DeadlineExceeded },
	}
	// winget 出错应被吞掉、不向上冒泡；目录扫描缺失也优雅降级。
	if _, err := l.Installed(context.Background()); err != nil {
		t.Fatalf("winget failure should not surface as error: %v", err)
	}
}

func TestRun_EmptyInstalledAllAdds(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Title: "T", Items: []toollist.Item{
		{Name: "A", License: "oss", Category: "设计", Why: "w"},
		{Name: "B", License: "free", Category: "效率", Why: "w"},
	}}
	rep, err := Run(context.Background(), fakeLister{}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Adds) != 2 || len(rep.Keeps) != 0 || len(rep.Replaces) != 0 {
		t.Fatalf("unexpected report: adds=%d keeps=%d replaces=%d", len(rep.Adds), len(rep.Keeps), len(rep.Replaces))
	}
	if rep.Adds[0].Category != "设计" {
		t.Errorf("category not propagated: %+v", rep.Adds[0])
	}
	if rep.InstalledCount != 0 {
		t.Errorf("installed_count = %d, want 0", rep.InstalledCount)
	}
}

func TestRun_EmptyCatalog(t *testing.T) {
	rep, err := Run(context.Background(), fakeLister{apps: []string{"Foo"}}, &toollist.Catalog{ID: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Adds) != 0 || len(rep.Keeps) != 0 || len(rep.Replaces) != 0 {
		t.Fatalf("empty catalog should yield empty report, got %+v", rep)
	}
	if rep.InstalledCount != 1 {
		t.Errorf("installed_count = %d, want 1", rep.InstalledCount)
	}
}

func TestRun_KeepMatchesByNameAndBrew(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Items: []toollist.Item{
		{Name: "Visual Studio Code", Brew: "visual-studio-code", License: "free"},
	}}
	// installed only via .app display name
	rep, _ := Run(context.Background(), fakeLister{apps: []string{"Visual Studio Code.app"}}, cat)
	if len(rep.Keeps) != 1 {
		t.Fatalf("expected keep by app name, got %+v", rep.Keeps)
	}
	// installed only via brew token
	rep2, _ := Run(context.Background(), fakeLister{apps: []string{"visual-studio-code"}}, cat)
	if len(rep2.Keeps) != 1 {
		t.Fatalf("expected keep by brew token, got %+v", rep2.Keeps)
	}
}

func TestNormalize_CJK(t *testing.T) {
	if got := Normalize("微信"); got != "微信" {
		t.Errorf("Normalize(微信) = %q", got)
	}
	if got := Normalize("  Raycast.app  "); got != "raycast" {
		t.Errorf("Normalize(Raycast.app) = %q", got)
	}
}

// windowsCatalog 模拟一个 Windows 清单：条目只带 winget id（无 brew），
// 平替目标也只带 winget id。
func windowsCatalog() *toollist.Catalog {
	return &toollist.Catalog{ID: "win", Items: []toollist.Item{
		{Name: "Git", Winget: "Git.Git", License: "oss"},
		{Name: "Steam", Winget: "Valve.Steam", License: "free"},
	}, Equivalences: []toollist.Equivalence{
		{From: []string{"WinRAR"}, To: toollist.To{Name: "7-Zip", Winget: "7zip.7zip", License: "oss"}, Why: "w"},
	}}
}

// TestRun_WingetIdMatch 验证已装集合里的 winget id 能让条目进入 keeps（而非 adds）。
func TestRun_WingetIdMatch(t *testing.T) {
	// 只装了 winget id（目录名/显示名都没匹配上），也应识别为已装。
	l := fakeLister{apps: []string{"Git.Git", "Valve.Steam"}}
	rep, err := Run(context.Background(), l, windowsCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Keeps) != 2 {
		t.Fatalf("winget id should match keeps, got keeps=%v adds=%v", rep.Keeps, rep.Adds)
	}
	if len(rep.Adds) != 0 {
		t.Errorf("expected no adds, got %v", rep.Adds)
	}
}

// TestRun_WingetIdSuppressesReplace 验证平替目标已通过 winget id 装好时不再推荐替换。
func TestRun_WingetIdSuppressesReplace(t *testing.T) {
	l := fakeLister{apps: []string{"WinRAR", "7zip.7zip"}}
	rep, err := Run(context.Background(), l, windowsCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Replaces) != 0 {
		t.Fatalf("winget-installed target should suppress replace, got %+v", rep.Replaces)
	}
}

// TestRun_WingetNotInstalledIsAdd 验证 winget 条目未装时进入 adds，且推荐带 winget id。
func TestRun_WingetNotInstalledIsAdd(t *testing.T) {
	l := fakeLister{apps: []string{"WinRAR"}}
	rep, err := Run(context.Background(), l, windowsCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Replaces) != 1 || rep.Replaces[0].Winget != "7zip.7zip" {
		t.Fatalf("expected 1 replace with winget id, got %+v", rep.Replaces)
	}
	var hasSteamAdd bool
	for _, a := range rep.Adds {
		if a.Name == "Steam" && a.Winget == "Valve.Steam" {
			hasSteamAdd = true
		}
	}
	if !hasSteamAdd {
		t.Errorf("expected Steam add carrying winget id, got %v", rep.Adds)
	}
	// Git/Steam 未装 → adds；WinRAR 已装但无对应 keeps（它只是 from）。
	if len(rep.Keeps) != 0 {
		t.Errorf("expected no keeps, got %v", rep.Keeps)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindEquivalence(t *testing.T) {
	cat := sampleCatalog()
	if eq := FindEquivalence(cat, "CleanShot X.app"); eq == nil || eq.To.Name != "Capso" {
		t.Fatalf("expected Capso for CleanShot X.app, got %+v", eq)
	}
	if eq := FindEquivalence(cat, "cleanshot"); eq == nil {
		t.Fatal("expected match by brew token cleanshot")
	}
	if eq := FindEquivalence(cat, "no-such-app"); eq != nil {
		t.Fatalf("expected nil, got %+v", eq)
	}
}

func TestRun_PlaybookPropagates(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Equivalences: []toollist.Equivalence{
		{From: []string{"X"}, To: toollist.To{Name: "Y", License: "oss"},
			Playbook: &toollist.Playbook{Summary: "s", DataFidelity: "d", VerifiedAt: "2026-11",
				Steps: []toollist.Step{{Title: "t", Detail: "d", SourceURL: "u"}}}},
	}}
	rep, err := Run(context.Background(), fakeLister{apps: []string{"X"}}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Replaces) != 1 {
		t.Fatalf("want 1 replace, got %d", len(rep.Replaces))
	}
	pb := rep.Replaces[0].Playbook
	if pb == nil || pb.Summary != "s" || len(pb.Steps) != 1 {
		t.Fatalf("playbook not propagated: %+v", pb)
	}
}

func TestRun_Unmatched(t *testing.T) {
	cat := &toollist.Catalog{ID: "t", Items: []toollist.Item{{Name: "IINA", Brew: "iina", License: "oss"}}}
	rep, err := Run(context.Background(), fakeLister{apps: []string{"IINA", "Foo", "Bar"}}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if rep.UnmatchedCount != 2 {
		t.Fatalf("unmatched = %d, want 2", rep.UnmatchedCount)
	}
	set := map[string]bool{}
	for _, n := range rep.UnmatchedSample {
		set[n] = true
	}
	if !set["Foo"] || !set["Bar"] {
		t.Errorf("sample should include Foo & Bar: %v", rep.UnmatchedSample)
	}
}
