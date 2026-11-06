#!/bin/sh
# SwiftInstall macOS 一键安装脚本
#
# 标准用法：
#   curl -fsSL https://raw.githubusercontent.com/cgartlab/SwiftInstall/main/install.sh | sh
#
# 指定版本：
#   SIS_VERSION=v0.1.0 sh install.sh
#
# 中国大陆加速（镜像）：
#   SIS_MIRROR=https://ghproxy.com sh install.sh
#
# 自定义安装目录：
#   SIS_INSTALL_DIR="$HOME/.local/bin" sh install.sh

set -eu

REPO_OWNER="cgartlab"
REPO_NAME="SwiftInstall"
BIN_NAME="sis"

info() { printf '\033[36m[SwiftInstall]\033[0m %s\n' "$*"; }
ok()   { printf '\033[32m[SwiftInstall]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[SwiftInstall]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[31m[SwiftInstall]\033[0m %s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

# 取回 URL 内容到标准输出（用于 JSON 元数据）。
fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --connect-timeout 15 "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$1"
  else
    die "需要 curl 或 wget 才能下载，请先安装其中之一。"
  fi
}

# 下载 URL 到指定文件。
download() {
  if command -v curl >/dev/null 2>&1; then
    curl -fL --retry 3 --connect-timeout 15 -o "$2" "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    die "需要 curl 或 wget 才能下载，请先安装其中之一。"
  fi
}

# 1. 操作系统检测（本脚本仅面向 macOS）
os="$(uname -s)"
if [ "$os" != "Darwin" ]; then
  die "此脚本仅支持 macOS（检测到 $os）。Windows 请使用 install.ps1。"
fi
info "检测到 macOS ($(sw_vers -productVersion 2>/dev/null || echo '未知版本'))"

# 2. CPU 架构检测
machine="$(uname -m)"
case "$machine" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64)  arch="amd64" ;;
  *) die "不支持的 CPU 架构：$machine（仅支持 Apple Silicon 的 arm64 与 Intel 的 amd64）。" ;;
esac
info "CPU 架构: $arch"

# 3. 解析目标版本：显式 SIS_VERSION 优先，否则查询 latest release。
version="${SIS_VERSION:-}"
if [ -z "$version" ]; then
  info "正在查询最新版本..."
  api="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
  json="$(fetch "$api")" || die "无法获取最新版本信息，请检查网络，或用 SIS_MIRROR / SIS_VERSION 指定。"
  version="$(printf '%s' "$json" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1)"
  [ -n "$version" ] || die "无法解析最新版本号，请用 SIS_VERSION 指定，例如 SIS_VERSION=v0.1.0。"
fi
# 允许省略 v 前缀
case "$version" in
  v*) ;;
  *) version="v${version}" ;;
esac
info "目标版本: $version"

# 4. 拼装下载地址（可选镜像加速）
asset="sis-darwin-${arch}"
url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${asset}"
if [ -n "${SIS_MIRROR:-}" ]; then
  mirror="$SIS_MIRROR"
  case "$mirror" in
    http://*|https://*) ;;
    *) mirror="https://${mirror}" ;;
  esac
  mirror="${mirror%/}"
  url="${mirror}/${url}"
  info "使用镜像加速: $mirror"
fi

# 5. 选择安装目录：优先 /usr/local/bin（可写时），否则回退到 ~/bin。
install_dir="${SIS_INSTALL_DIR:-}"
if [ -z "$install_dir" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir="/usr/local/bin"
  elif [ ! -e /usr/local/bin ] && [ -w /usr/local ]; then
    install_dir="/usr/local/bin"
  else
    install_dir="${HOME}/bin"
    warn "/usr/local/bin 不可写，回退到 ${install_dir}（无需 sudo）。"
  fi
fi
mkdir -p "$install_dir" 2>/dev/null || die "无法创建安装目录: $install_dir"
[ -w "$install_dir" ] || die "安装目录不可写: $install_dir。请设置 SIS_INSTALL_DIR 为可写目录。"
info "安装目录: $install_dir"

# 6. 下载到临时文件
tmp="$(mktemp "${TMPDIR:-/tmp}/sis.XXXXXX")"
trap 'rm -f "$tmp"' EXIT INT TERM
info "正在下载 ${asset} ..."
if ! download "$url" "$tmp"; then
  die "下载失败: $url"
fi
chmod +x "$tmp"

# 7. 安装（覆盖旧版本前保留原文件备份）
target="${install_dir}/${BIN_NAME}"
if [ -e "$target" ]; then
  warn "检测到已安装的 ${target}，将覆盖。"
fi
if ! mv -f "$tmp" "$target" 2>/dev/null; then
  cp -f "$tmp" "$target" || die "写入 ${target} 失败"
  rm -f "$tmp"
fi
chmod +x "$target"
trap - EXIT INT TERM
ok "已安装到: ${target}"

# 8. 验证版本
info "正在验证安装..."
if out="$("$target" version 2>/dev/null)"; then
  ok "验证成功: ${out}"
else
  warn "文件已安装，但暂时无法执行 sis version（可能需要重新打开终端）。"
fi

# 9. PATH 提示
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *)
    warn "提示: ${install_dir} 不在 PATH 中。请将下面一行加入 ~/.zshrc 或 ~/.bash_profile："
    printf '\n  export PATH="%s:$PATH"\n\n' "$install_dir"
    ;;
esac

# 10. 完成
printf '\n'
ok "SwiftInstall 安装完成！"
printf '\n'
printf '  sis install     # 按清单批量安装\n'
printf '  sis list        # 查看清单\n'
printf '  sis status      # 检查安装状态\n'
printf '  sis tui         # 交互式界面\n'
printf '\n'
info "如果 sis 命令未生效，请重新打开终端。"
