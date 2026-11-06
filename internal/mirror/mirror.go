// Package mirror configures China-mainland mirrors for the development toolchain
// (Homebrew, npm, pip) and a GitHub download proxy, so software installs at full
// speed. Configuration is idempotent and clean: it writes one managed env file
// and sources it from the shell rc with a marked block, rather than appending
// duplicate lines on every run. `Restore` removes everything it set.
package mirror

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Mirror endpoints (USTC for Homebrew, npmmirror for npm, Tuna for pip).
const (
	brewAPI     = "https://mirrors.ustc.edu.cn/homebrew-bottles/api"
	brewBottle  = "https://mirrors.ustc.edu.cn/homebrew-bottles"
	brewGit     = "https://mirrors.ustc.edu.cn/brew.git"
	coreGit     = "https://mirrors.ustc.edu.cn/homebrew-core.git"
	caskGit     = "https://mirrors.ustc.edu.cn/homebrew-cask.git"
	npmRegistry = "https://registry.npmmirror.com"
	pipIndex    = "https://pypi.tuna.tsinghua.edu.cn/simple"
	ghProxy     = "https://ghproxy.com"
)

const beginMarker = "# >>> sis mirror >>>"
const endMarker = "# <<< sis mirror <<<"

// ConfigDir returns ~/.config/sis (created).
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		base = filepath.Join(os.Getenv("HOME"), ".config")
	}
	dir := filepath.Join(base, "sis")
	return dir, os.MkdirAll(dir, 0o755)
}

// EnvFile is the managed shell env file sourced from the rc.
func EnvFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mirror.env"), nil
}

func envContent() string {
	return strings.Join([]string{
		"# Managed by `sis mirror apply`. Do not edit; use `sis mirror restore`.",
		fmt.Sprintf("export HOMEBREW_API_DOMAIN=%q", brewAPI),
		fmt.Sprintf("export HOMEBREW_BOTTLE_DOMAIN=%q", brewBottle),
		fmt.Sprintf("export HOMEBREW_BREW_GIT_REMOTE=%q", brewGit),
		fmt.Sprintf("export HOMEBREW_CORE_GIT_REMOTE=%q", coreGit),
		fmt.Sprintf("export HOMEBREW_CASK_GIT_REMOTE=%q", caskGit),
		fmt.Sprintf("export SIS_GH_PROXY=%q", ghProxy),
		"",
	}, "\n")
}

// rcFiles returns the shell startup files that should source the env.
func rcFiles() []string {
	home := os.Getenv("HOME")
	return []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bash_profile"),
	}
}

// ensureSourced makes rc file source the env file inside a marked block.
func ensureSourced(rc, env string) error {
	data, _ := os.ReadFile(rc)
	content := string(data)
	block := fmt.Sprintf("%s\n[ -f %q ] && source %q\n%s\n", beginMarker, env, env, endMarker)

	if strings.Contains(content, beginMarker) {
		return nil // already sourced
	}
	// Append a blank line + block.
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n" + block
	return os.WriteFile(rc, []byte(content), 0o644)
}

// removeSourced strips the marked block from rc.
func removeSourced(rc string) error {
	data, err := os.ReadFile(rc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := string(data)
	for {
		start := strings.Index(content, beginMarker)
		if start < 0 {
			break
		}
		end := strings.Index(content[start:], endMarker)
		if end < 0 {
			break
		}
		end += start + len(endMarker)
		// trim a trailing newline if present
		if end < len(content) && content[end] == '\n' {
			end++
		}
		// trim a leading blank line before the block
		s := start
		if s > 0 && content[s-1] == '\n' {
			s--
		}
		content = content[:s] + content[end:]
	}
	return os.WriteFile(rc, []byte(content), 0o644)
}

// run executes a command silently; returns error message.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// has reports whether an executable is on PATH.
func has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Apply configures all mirrors (Homebrew env, npm, pip, GitHub proxy).
// Missing tools (e.g. no pip) are skipped, not fatal.
func Apply() ([]string, error) {
	var done []string

	dir, err := ConfigDir()
	if err != nil {
		return done, err
	}
	env := filepath.Join(dir, "mirror.env")
	if err := os.WriteFile(env, []byte(envContent()), 0o644); err != nil {
		return done, fmt.Errorf("写入 %s: %w", env, err)
	}
	done = append(done, "Homebrew 镜像环境 → "+env)
	for _, rc := range rcFiles() {
		// Only touch rc files that already exist, to avoid creating unused ones.
		if _, err := os.Stat(rc); err == nil {
			if err := ensureSourced(rc, env); err != nil {
				return done, fmt.Errorf("更新 %s: %w", rc, err)
			}
			done = append(done, "已注入到 "+filepath.Base(rc))
		}
	}

	if has("npm") {
		if err := run("npm", "config", "set", "registry", npmRegistry); err != nil {
			return done, err
		}
		done = append(done, "npm registry → "+npmRegistry)
	} else {
		done = append(done, "npm: 未安装，跳过")
	}

	if has("pip3") {
		if err := run("pip3", "config", "set", "global.index-url", pipIndex); err != nil {
			return done, err
		}
		done = append(done, "pip index → "+pipIndex)
	} else if has("pip") {
		if err := run("pip", "config", "set", "global.index-url", pipIndex); err != nil {
			return done, err
		}
		done = append(done, "pip index → "+pipIndex)
	} else {
		done = append(done, "pip: 未安装，跳过")
	}

	done = append(done, "GitHub 下载代理 → "+ghProxy+" (由 sis 在下载 release 时使用)")
	return done, nil
}

// Restore reverts everything Apply set.
func Restore() ([]string, error) {
	var done []string

	env, err := EnvFile()
	if err != nil {
		return done, err
	}
	if err := os.Remove(env); err != nil && !os.IsNotExist(err) {
		return done, err
	}
	done = append(done, "已删除 "+filepath.Base(env))
	for _, rc := range rcFiles() {
		if err := removeSourced(rc); err != nil {
			return done, err
		}
	}
	done = append(done, "已从 shell rc 移除注入")

	if has("npm") {
		_ = run("npm", "config", "delete", "registry")
		done = append(done, "npm registry 已恢复默认")
	}
	if has("pip3") {
		_ = run("pip3", "config", "unset", "global.index-url")
		done = append(done, "pip index 已恢复默认")
	} else if has("pip") {
		_ = run("pip", "config", "unset", "global.index-url")
		done = append(done, "pip index 已恢复默认")
	}
	return done, nil
}

// Show prints the current mirror state (from env file + tool configs).
func Show() ([]string, error) {
	var lines []string

	env, _ := EnvFile()
	if data, err := os.ReadFile(env); err == nil {
		lines = append(lines, "— Homebrew 镜像环境（"+env+"）—")
		for _, l := range strings.Split(string(data), "\n") {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "#") {
				continue
			}
			lines = append(lines, "  "+l)
		}
	} else {
		lines = append(lines, "— Homebrew 镜像环境：未配置（sis mirror apply）—")
	}

	if has("npm") {
		out, _ := exec.Command("npm", "config", "get", "registry").Output()
		lines = append(lines, "— npm registry: "+strings.TrimSpace(string(out)))
	} else {
		lines = append(lines, "— npm: 未安装")
	}
	if has("pip3") {
		out, _ := exec.Command("pip3", "config", "get", "global.index-url").Output()
		lines = append(lines, "— pip index: "+strings.TrimSpace(string(out)))
	}
	if v := os.Getenv("SIS_GH_PROXY"); v != "" {
		lines = append(lines, "— GitHub 代理 (SIS_GH_PROXY): "+v)
	} else {
		lines = append(lines, "— GitHub 代理: 未设置（apply 后写入 mirror.env）")
	}
	return lines, nil
}
