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
  - [mirror — 镜像源管理](#mirror--镜像源管理)
  - [config — 配置管理](#config--配置管理)
  - [version — 版本信息](#version--版本信息)
- [Manifest 文件格式](#manifest-文件格式)
  - [YAML 格式](#yaml-格式)
  - [TXT 格式](#txt-格式)
- [配置文件](#配置文件)
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
| **现代化 CLI** | 基于 Go + Cobra 的命令行工具，支持子命令、配置管理与丰富输出 |
| **批量安装/卸载** | 基于 Manifest 文件自动遍历安装或卸载，失败不中断 |
| **国内镜像加速** | 内置 USTC（中科大）镜像源切换，解决中国大陆访问缓慢问题 |
| **代理自动检测** | 自动检测本地 v2rayN 代理（`127.0.0.1:10809`），支持手动指定代理地址 |
| **预检机制** | 安装前自动检查包管理器、Manifest 文件、管理员权限等 |
| **去重与跳过** | 自动跳过重复包与已安装软件 |
| **重试机制** | 安装失败自动重试，可配置重试次数与间隔 |
| **高度可定制** | 通过 YAML/TXT 清单与 JSON 配置文件自由调整行为 |

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
irm https://raw.githubusercontent.com/cgartlab/Software_Install_Script/main/install.ps1 | iex
```

安装完成后，重新打开终端即可使用 `sis` 命令。

#### 一键装机（带默认软件清单）

如果你已经准备好 `sis.yaml` 或 `software_list.txt`，可以一键完成全部软件安装：

```powershell
# 下载示例清单并执行安装
irm https://raw.githubusercontent.com/cgartlab/Software_Install_Script/main/Windows/software_list.txt -OutFile software_list.txt
sis install
```

---

### 手动安装

#### 方式一：直接下载预编译二进制

从 [Releases](https://github.com/cgartlab/Software_Install_Script/releases) 页面下载对应平台的 `sis.exe`，放入系统 `PATH` 中。

#### 方式二：从源码编译

```powershell
git clone https://github.com/cgartlab/Software_Install_Script.git
cd Software_Install_Script
go build -o sis.exe ./cmd/sis/
```

编译完成后，将生成的 `sis.exe` 放入系统 `PATH`。

#### 方式三：使用传统脚本（遗留方案）

项目仍保留原始的 Batch 脚本，适用于无需 CLI 的场景：

- **Windows**：运行 `Windows/software_install.bat`
- **Windows（代理版）**：运行 `Windows/software_install_proxy.bat`（需 v2rayN 在 `127.0.0.1:10809`）
- **Windows（切换镜像源）**：运行 `Windows/switch_winget_to_USTCsource.bat`

---

## 使用指南

### CLI 命令总览

```text
sis install    从 Manifest 文件批量安装软件
sis uninstall  从 Manifest 文件批量卸载软件
sis list       查看 Manifest 中的软件清单
sis status     检查 Manifest 中各软件的安装状态
sis mirror     管理 winget 镜像源（Windows）
sis config     查看和修改配置
sis version    显示版本信息
```

全局选项：

```text
-v, --verbose   输出详细日志（当前版本标志存在，详细日志功能开发中）
```

---

### install — 批量安装

```bash
# 使用默认 Manifest（自动查找 sis.yaml / software_list.txt / packages.txt）
sis install

# 指定 Manifest 文件
sis install -f software_list.txt

# 仅预览，不实际安装
sis install --dry-run

# 使用代理
sis install --proxy http://127.0.0.1:10809

# 切换镜像源（Windows）
sis install --mirror ustc

# 跳过已安装软件
sis install --skip-existing

# 跳过预检
sis install --skip-checks
```

安装过程中会逐条输出进度，最后给出汇总：

```text
Installing 4 packages from software_list.txt
  ✓ Microsoft.VisualStudioCode
  ✓ Git.Git
  ⚠ 7zip.7zip
  ✗ ObsProject.OBSStudio

Summary: 4 total, 2 succeeded, 1 skipped, 1 failed (45.2s)
```

> **注意**：安装失败不会中断整个批次，错误会在最后的汇总中报告。

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

  Microsoft.VisualStudioCode      dev
  Git.Git                         dev
  7zip.7zip                       utils
  ObsProject.OBSStudio            utils

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
Status for 4 packages from software_list.txt

  ✓ Microsoft.VisualStudioCode
  ✗ Git.Git
  ✓ 7zip.7zip
  ✗ ObsProject.OBSStudio

2 installed, 2 missing
```

---

### mirror — 镜像源管理

> 仅 Windows 平台支持。用于切换 `winget` 的源地址，解决中国大陆下载缓慢问题。

```bash
# 查看当前镜像源
sis mirror --status

# 切换到中科大 USTC 镜像
sis mirror ustc

# 切换回官方源
sis mirror official

# 重置为官方源
sis mirror --reset
```

支持的镜像源：

| 名称 | 说明 |
|------|------|
| `official` | 微软官方源 |
| `ustc` | 中科大镜像源 |

---

### config — 配置管理

配置文件采用两级 JSON 结构：

- **全局配置**：`~/.sis/config.json`
- **本地配置**：当前目录下的 `.sis.json`

优先级：**CLI 参数 > 本地配置 > 全局配置 > 内置默认值**

```bash
# 查看某项配置
sis config get mirror
sis config get log_level

# 设置配置（默认保存到全局配置）
sis config set mirror ustc
sis config set proxy http://127.0.0.1:10809
sis config set log_level debug

# 保存到本地配置
sis config set default_manifest ./sis.yaml --local
```

可配置项：

| 键 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `mirror` | string | `""` | 默认镜像源（`ustc` / `official`） |
| `proxy` | string | `""` | HTTP/HTTPS 代理地址 |
| `proxy_auto_detect` | bool | `false` | 是否自动检测 v2rayN 代理 |
| `log_level` | string | `"info"` | 日志级别：`debug` / `info` / `warn` / `error` |
| `log_file` | string | `""` | 日志文件路径 |
| `default_manifest` | string | `""` | 默认 Manifest 文件路径 |
| `skip_existing` | bool | `false` | 默认跳过已安装软件 |
| `skip_checks` | bool | `false` | 默认跳过预检 |
| `color` | string | `"auto"` | 终端颜色：`auto` / `always` / `never` |
| `retry_count` | int | `2` | 安装失败重试次数 |
| `retry_delay_sec` | int | `3` | 每次重试间隔（秒） |

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
2. `software_list.txt`
3. `packages.txt`

### YAML 格式

推荐格式，支持分类、镜像、代理等高级配置：

```yaml
mirror: ustc
proxy: http://127.0.0.1:10809

packages:
  - id: Microsoft.VisualStudioCode
    category: dev
  - id: Git.Git
    category: dev
  - id: 7zip.7zip
    category: utils
```

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

## 配置文件

### 全局配置示例（`~/.sis/config.json`）

```json
{
  "mirror": "ustc",
  "proxy_auto_detect": true,
  "log_level": "info",
  "default_manifest": "./software_list.txt",
  "skip_existing": true,
  "retry_count": 3,
  "retry_delay_sec": 5
}
```

### 本地配置示例（`.sis.json`）

```json
{
  "mirror": "official",
  "default_manifest": "sis.yaml"
}
```

---

## 项目结构

```text
Software_Install_Script/
├── cmd/sis/
│   └── main.go                     # CLI 入口（ldflags 注入版本信息）
├── internal/
│   ├── cli/                        # Cobra 命令实现
│   │   ├── root.go                 # 根命令与全局标志
│   │   ├── install.go              # install 子命令
│   │   ├── uninstall.go            # uninstall 子命令
│   │   ├── list.go                 # list 子命令
│   │   ├── status.go               # status 子命令
│   │   ├── mirror.go               # mirror 子命令
│   │   ├── config.go               # config 子命令（get/set）
│   │   └── version.go              # version 子命令
│   ├── engine/                     # 安装引擎 + Manifest 解析 + 预检
│   │   ├── engine.go               # 核心安装/卸载/状态检查逻辑
│   │   ├── manifest.go             # YAML/TXT Manifest 解析
│   │   └── check.go                # 预检（winget/Manifest/管理员权限）
│   ├── backend/                    # 后端接口 + winget 实现
│   │   ├── backend.go              # Backend 接口与错误处理
│   │   └── winget.go               # winget 后端实现
│   ├── config/                     # 两级 JSON 配置管理
│   │   └── config.go               # 配置加载/合并/保存/校验
│   ├── mirror/                     # USTC 镜像源切换
│   │   └── mirror.go               # winget source 管理
│   ├── proxy/                      # v2rayN 代理检测
│   │   └── proxy.go                # 进程检测与代理信息
│   └── log/                        # 分级日志（slog，终端 + JSON 文件）
│       └── logger.go               # Logger 实现
├── Windows/
│   ├── software_install.bat        # Windows 标准安装脚本（遗留）
│   ├── software_install_proxy.bat  # Windows 代理安装脚本（遗留）
│   ├── switch_winget_to_USTCsource.bat # 切换 winget 国内源（遗留）
│   └── software_list.txt           # Windows 软件清单示例
├── macOS/
│   ├── install_packages.sh         # macOS 安装脚本（遗留，待迁移到 Go CLI）
│   └── packages.txt                # macOS 软件清单示例
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
