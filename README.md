# SwiftInstall

<p align="center">
  <b>跨平台软件批量安装工具</b><br>
  <sub>基于 Windows <code>winget</code> 的一键自动化装机方案（macOS Homebrew 支持开发中）</sub>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/platform-Windows-blue?style=flat-square" alt="Platform">
  <img src="https://img.shields.io/badge/license-GPL--3.0-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/language-Go%20%7C%20Batch-orange?style=flat-square" alt="Language">
</p>

---

## 目录

- [项目概述](#项目概述)
- [核心功能特性](#核心功能特性)
- [快速开始](#快速开始)
  - [环境要求](#环境要求)
  - [一键安装](#一键安装)
  - [手动安装](#手动安装)
- [使用指南](#使用指南)
  - [CLI 命令总览](#cli-命令总览)
  - [install — 批量安装](#install--批量安装)
  - [uninstall — 批量卸载](#uninstall--批量卸载)
  - [list — 查看软件清单](#list--查看软件清单)
  - [status — 检查安装状态](#status--检查安装状态)
  - [version — 版本信息](#version--版本信息)
- [Manifest 文件格式](#manifest-文件格式)
  - [YAML 格式](#yaml-格式)
  - [TXT 格式](#txt-格式)
- [项目结构](#项目结构)
- [贡献指南](#贡献指南)
- [许可证](#许可证)

---

## 项目概述

**SwiftInstall** 是一个开源的批量软件安装工具，提供现代化的 Go CLI（`sis`）以及传统的脚本方案，帮助用户在新系统环境或重装系统后，通过简单的命令快速、自动化地安装日常开发所需的全部软件。

目前针对 **Windows** 提供了基于 `winget` 的完整解决方案，并内置了中国大陆网络环境的镜像加速与代理支持，无需复杂配置即可开箱即用。**macOS Homebrew 后端正在开发中。**

---

## 核心功能特性

| 特性 | 说明 |
|------|------|
| **Windows 完整支持** | 基于原生 `winget` 包管理器，覆盖安装/卸载/查询全生命周期 |
| **现代化 CLI** | 基于 Go + Cobra 的命令行工具，子命令 + 多种输出格式（table/json/silent） |
| **批量安装/卸载** | 基于 Manifest 文件自动遍历安装或卸载，失败不中断 |
| **镜像加速安装** | 安装脚本支持 `$env:SIS_MIRROR` 镜像加速下载；安装时可指定 `--proxy` |
| **代理支持** | 通过 `--proxy` 或清单中的 `proxy:` 指定 HTTP/HTTPS 代理 |
| **前置检查** | 安装前校验 Manifest 内容，并确认 `winget` 可用 |
| **去重与跳过** | 自动去重清单中的重复包，可跳过已安装软件 |
| **重试机制** | 安装失败自动重试，可配置重试次数与间隔 |
| **高度可定制** | 通过 YAML/TXT 清单调整要安装的软件与行为 |

---

## 快速开始

### 环境要求

#### Windows
- **操作系统**：Windows 10 版本 1809 或更高版本 / Windows 11
- **依赖工具**：[Windows Package Manager (winget)](https://learn.microsoft.com/zh-cn/windows/package-manager/winget/)（Windows 11 已预装，Windows 10 需手动安装）
- **权限要求**：部分操作需要**管理员权限**运行

#### macOS（开发中）
- **操作系统**：macOS 10.15 (Catalina) 或更高版本
- **依赖工具**：[Homebrew](https://brew.sh/)
- **状态**：Homebrew 后端尚未实现，敬请期待

---

### 一键安装

#### Windows（PowerShell）

在管理员 PowerShell 中执行以下命令，自动下载并安装最新版 `sis`：

```powershell
irm https://raw.githubusercontent.com/cgartlab/SwiftInstall/main/install.ps1 | iex
```

**中国大陆用户加速安装（任选其一）：**

```powershell
# 方式 1：使用 jsDelivr CDN（国内有 CDN 节点）
irm https://cdn.jsdelivr.net/gh/cgartlab/SwiftInstall@main/install.ps1 | iex

# 方式 2：使用 ghproxy 镜像
$env:SIS_MIRROR='ghproxy.com'; irm https://raw.githubusercontent.com/cgartlab/SwiftInstall/main/install.ps1 | iex

# 方式 3：镜像直接代理
irm https://ghproxy.com/https://raw.githubusercontent.com/cgartlab/SwiftInstall/main/install.ps1 | iex
```

安装完成后，重新打开终端即可使用 `sis` 命令。

#### 一键装机（带默认软件清单）

如果你已经准备好 `sis.yaml` 或 `software_list.txt`，可以一键完成全部软件安装：

```powershell
# 下载示例清单并执行安装
irm https://raw.githubusercontent.com/cgartlab/SwiftInstall/main/Windows/software_list.txt -OutFile software_list.txt
sis install
```

---

### 手动安装

#### 方式一：直接下载预编译二进制

从 [Releases](https://github.com/cgartlab/SwiftInstall/releases) 页面下载对应平台的 `sis.exe`，放入系统 `PATH` 中。

#### 方式二：从源码编译

```powershell
git clone https://github.com/cgartlab/SwiftInstall.git
cd SwiftInstall
go build -o sis.exe ./cmd/sis/
```

编译完成后，将生成的 `sis.exe` 放入系统 `PATH`。

#### 方式三：使用传统脚本（遗留方案）

项目仍保留原始的脚本，适用于无需 CLI 的场景：

- **Windows**：运行 `Windows/software_install.bat`（读取同目录的 `software_list.txt`）
- **macOS**：运行 `macOS/install_packages.sh`（读取同目录的 `packages.txt`）

两个脚本都已支持 YAML 清单与纯 TXT 列表两种格式。

---

## 使用指南

### CLI 命令总览

```text
sis install    从 Manifest 文件批量安装软件
sis uninstall  从 Manifest 文件批量卸载软件
sis list       查看 Manifest 中的软件清单
sis status     检查 Manifest 中各软件的安装状态
sis version    显示版本信息
```

全局选项：

```text
-f, --file string   清单文件路径（默认自动查找 sis.yaml / sis.yml / software_list.txt / packages.txt）
    --format string 输出格式：table, json, silent（默认 table）
    --no-color      禁用彩色输出
```

---

### install — 批量安装

```bash
# 使用默认 Manifest（自动查找 sis.yaml / sis.yml / software_list.txt / packages.txt）
sis install

# 指定 Manifest 文件
sis install -f software_list.txt

# 仅预览，不实际安装
sis install --dry-run

# 使用代理
sis install --proxy http://127.0.0.1:10809

# 跳过已安装软件
sis install --skip-existing

# 失败重试 3 次，间隔 5 秒
sis install --retry 3 --retry-delay 5
```

安装过程中会逐条输出进度，最后给出汇总：

```text
Installing 4 packages from software_list.txt

  [1/4] ✓ Microsoft.VisualStudioCode (dev)
  [2/4] ✓ Git.Git (dev)
  [3/4] ⚠ 7zip.7zip (utils) — winget exited with code -1978335189: No package found matching input criteria.
  [4/4] ✗ ObsProject.OBSStudio (utils) — winget exited with code -1978335189: No package found matching input criteria.

  ────────────────────────────────────────
  Total: 4  |  2 succeeded  1 skipped  1 failed (45.2s)

  Failed packages:
    ✗ ObsProject.OBSStudio — winget exited with code -1978335189: No package found matching input criteria.
```

> **注意**：安装失败不会中断整个批次，错误会在最后的汇总中报告。标记为 `optional: true` 的包失败时计入 `skipped`，不影响退出码。

---

### uninstall — 批量卸载

```bash
# 卸载清单中所有软件（必须加 --yes 确认）
sis uninstall -f software_list.txt --yes

# 预览卸载（不实际执行）
sis uninstall -f software_list.txt --yes --dry-run
```

---

### list — 查看软件清单

```bash
# 表格形式列出所有软件
sis list

# 指定 Manifest
sis list -f sis.yaml

# 按分类过滤
sis list --category dev

# JSON 输出
sis list --format json
```

示例输出：

```text
Packages from software_list.txt

  dev
    Microsoft.VisualStudioCode
    Git.Git

  utils
    7zip.7zip
    ObsProject.OBSStudio

  Total: 4 packages
```

---

### status — 检查安装状态

```bash
sis status
sis status -f software_list.txt
```

示例输出：

```text
Checking status of 4 packages from software_list.txt

  [1/4] ✓ Microsoft.VisualStudioCode
  [2/4] ✗ Git.Git
  [3/4] ✓ 7zip.7zip
  [4/4] ✗ ObsProject.OBSStudio

  2 installed, 2 missing
```

---

### version — 版本信息

```bash
sis version
```

示例输出：

```text
sis 1.0.0 (a1b2c3d, 2024-01-15)
```

---

## Manifest 文件格式

`sis` 支持 **YAML** 和 **TXT** 两种格式的 Manifest 文件，默认按以下顺序自动查找：

1. `sis.yaml`
2. `sis.yml`
3. `software_list.txt`
4. `packages.txt`

文件格式以**内容**为准：`.yaml` / `.yml` 一律按 YAML 解析；其余扩展名只要包含 `packages:` 或 `settings:` 键就按 YAML 解析，否则按 TXT 列表解析（所以 YAML 内容可以叫 `.txt`）。清单中的重复包 ID 会被自动去重。

### YAML 格式

推荐格式，支持分类、代理、重试等高级配置：

```yaml
proxy: http://127.0.0.1:10809

packages:
  - id: Microsoft.VisualStudioCode
    category: dev
  - id: Git.Git
    category: dev
  - id: 7zip.7zip
    category: utils
```

清单级可选设置（写在顶层或 `settings:` 块内均可）：

```yaml
settings:
  proxy: http://127.0.0.1:10809   # HTTP/HTTPS 代理
  skip_existing: true             # 跳过已安装软件
  retry_count: 2                  # 失败重试次数
  retry_delay: 3                  # 重试间隔（秒）
```

包级可选字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 包 ID（必填） |
| `category` | string | 分类，仅用于 `sis list` 分组展示 |
| `optional` | bool | 为 `true` 时，该包安装失败不计入 `failed`，也不影响退出码 |

### TXT 格式

兼容传统脚本，每行一个包 ID，`#` 开头的行为注释，可作为分类标题：

```text
# dev
Microsoft.VisualStudioCode
Git.Git

# utils
7zip.7zip
ObsProject.OBSStudio
```

---

## 项目结构

```text
SwiftInstall/
├── cmd/sis/
│   └── main.go                     # CLI 入口（ldflags 注入版本信息）
├── internal/
│   ├── cli/                        # Cobra 命令实现
│   │   ├── root.go                 # 根命令与全局标志
│   │   ├── install.go              # install 子命令
│   │   ├── uninstall.go            # uninstall 子命令
│   │   ├── list.go                 # list 子命令
│   │   ├── status.go               # status 子命令
│   │   ├── helpers.go              # 清单自动查找与 Renderer 选择
│   │   └── version.go              # version 子命令
│   ├── engine/                     # 批量安装/卸载编排（跳过、干跑、重试）
│   │   └── engine.go
│   ├── manifest/                   # YAML/TXT 清单解析、去重与校验
│   │   ├── manifest.go             # Manifest/Package/Settings 结构
│   │   ├── parser.go               # 格式识别与解析
│   │   └── validate.go             # 清单校验
│   ├── backend/                    # 后端接口 + winget 实现
│   │   ├── backend.go              # Backend 接口与 Output/InstallOptions
│   │   └── winget.go               # winget 后端实现
│   └── ui/                         # 输出渲染器
│       ├── renderer.go             # Renderer 接口
│       ├── terminal.go             # 终端彩色输出
│       ├── json.go                 # JSON 输出
│       └── silent.go               # 静默输出
├── Windows/
│   ├── software_install.bat        # Windows 安装脚本（遗留）
│   └── software_list.txt           # Windows 软件清单示例（YAML 内容）
├── macOS/
│   ├── install_packages.sh         # macOS 安装脚本（遗留，待迁移到 Go CLI）
│   └── packages.txt                # macOS 软件清单示例（YAML 内容）
├── install.ps1                     # Windows 一键安装脚本
├── README.md
├── LICENSE
└── AGENTS.md
```

---

## 贡献指南

欢迎提交 Issue 和 Pull Request！

1. **Fork** 本仓库。
2. 在您的分支上进行修改：`git checkout -b feature/YourFeature`。
3. 提交更改：`git commit -m 'Add some feature'`。
4. 推送分支：`git push origin feature/YourFeature`。
5. 新建一个 **Pull Request**。

### 提交规范

- 保持代码简洁，遵循 Go 官方代码规范。
- 修改 CLI 命令时，请同步更新 `README.md` 使用说明。
- 新增功能请补充必要的测试与文档。

---

## 许可证

本项目采用 [GNU General Public License v3.0](LICENSE) 开源许可证。

```text
SwiftInstall — Cross-platform batch software installer
Copyright (C) 2024 cgartlab

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```

---

<p align="center">如果本项目对您有帮助，欢迎点亮 ⭐ Star 支持我们！</p>
