# SwiftInstall

<p align="center">
  <b>跨平台软件批量安装工具</b><br>
  <sub>基于 Windows <code>winget</code> 与 macOS <code>Homebrew</code> 的一键自动化装机方案</sub>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-blue?style=flat-square" alt="Platform">
  <img src="https://img.shields.io/badge/license-GPL--3.0-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/language-Bash%20%7C%20Batch-orange?style=flat-square" alt="Language">
</p>

---

## 目录

- [项目概述](#项目概述)
- [核心功能特性](#核心功能特性)
- [快速开始](#快速开始)
  - [环境要求](#环境要求)
  - [安装与配置](#安装与配置)
- [使用指南](#使用指南)
  - [Windows](#windows)
  - [macOS](#macos)
- [自定义软件列表](#自定义软件列表)
  - [Windows 软件清单](#windows-软件清单)
  - [macOS 软件清单](#macos-软件清单)
- [脚本说明](#脚本说明)
- [项目结构](#项目结构)
- [贡献指南](#贡献指南)
- [许可证](#许可证)
- [联系方式](#联系方式)

---

## 项目概述

**SwiftInstall** 是一个开源的跨平台批量软件安装脚本集合，旨在帮助用户在新系统环境或重装系统后，通过简单的命令或双击操作，快速、自动化地安装日常开发与设计所需的全部软件。

项目针对 **Windows** 和 **macOS** 两大主流操作系统分别提供了基于原生包管理器（`winget` / `Homebrew`）的解决方案，并内置了中国大陆网络环境的镜像加速支持，无需复杂配置即可开箱即用。

## 核心功能特性

| 特性 | 说明 |
|------|------|
| **跨平台支持** | 同时覆盖 Windows 与 macOS 双平台，统一的使用体验 |
| **一键批量安装** | 基于清单文件自动遍历安装，无需手动逐一下载 |
| **国内镜像加速** | 内置 USTC（中科大）镜像源切换脚本，解决中国大陆访问缓慢问题 |
| **代理安装支持** | Windows 平台提供 v2rayN 代理安装模式，适应不同网络环境 |
| **去重与跳过** | macOS 脚本自动检测已安装软件并跳过，避免重复安装 |
| **零依赖运行** | 除系统自带包管理器外，无需安装任何额外运行时或编译环境 |
| **高度可定制** | 通过简单的文本编辑即可自由增删软件清单 |

## 快速开始

### 环境要求

#### Windows
- **操作系统**：Windows 10 版本 1809 或更高版本 / Windows 11
- **依赖工具**：[Windows Package Manager (winget)](https://learn.microsoft.com/zh-cn/windows/package-manager/winget/)（Windows 11 已预装，Windows 10 需手动安装）
- **权限要求**：部分脚本需要**管理员权限**运行

#### macOS
- **操作系统**：macOS 10.15 (Catalina) 或更高版本
- **依赖工具**：[Homebrew](https://brew.sh/)（若未安装，请先在终端执行安装命令）
- **权限要求**：标准用户权限即可

### 安装与配置

1. **克隆或下载仓库**

   ```bash
   git clone https://github.com/cgartlab/Software_Install_Script.git
   cd Software_Install_Script
   ```

   或直接下载 [ZIP 压缩包](https://github.com/cgartlab/Software_Install_Script/archive/refs/heads/main.zip) 并解压。

2. **（中国大陆用户推荐）切换国内镜像源**

   - **Windows**：右键以管理员身份运行 `Windows/switch_winget_to_USTCsource.bat`，按提示选择 `[Y]` 切换至 USTC 源。
   - **macOS**：`macOS/install_packages.sh` 脚本已内置 Homebrew 中科大源切换逻辑，无需手动操作。

## 使用指南

### Windows

#### 标准安装流程

1. 将项目文件夹解压至同一目录下。
2. （可选）运行 `switch_winget_to_USTCsource.bat` 切换国内源以提升下载速度。
3. 根据自身需求编辑 `Windows/software_list.txt`，增删软件包 ID。
4. 双击运行 `software_install.bat`，脚本将自动读取清单并逐一下载安装。

#### 代理安装流程（需 v2rayN）

若你本地已运行 [v2rayN](https://github.com/2dust/v2rayN) 且监听在 `127.0.0.1:10809`，可直接运行 `software_install_proxy.bat`，脚本会通过 HTTP 代理加速下载。

```batch
REM 代理脚本内部核心逻辑示例
winget install <PackageId> --proxy http://127.0.0.1:10809
```

#### 管理员权限说明

`switch_winget_to_USTCsource.bat` 在启动时会自动检测当前是否具备管理员权限；若未提升，将弹出提示并终止运行，需右键选择“以管理员身份运行”。

### macOS

1. 将项目文件夹解压至任意目录。
2. 打开**终端 (Terminal)**。
3. 将 `macOS/install_packages.sh` 文件拖入终端窗口，按 **回车键** 执行。

```bash
# 或直接通过命令行执行
bash /path/to/macOS/install_packages.sh
```

脚本会自动完成以下操作：
- 切换 Homebrew 至中科大镜像源
- 读取 `packages.txt` 软件清单
- 检查每个软件是否已安装，若未安装则自动执行 `brew install`

## 自定义软件列表

### Windows 软件清单

编辑 [`Windows/software_list.txt`](./Windows/software_list.txt)，每行填写一个 `winget` 包 ID（以 `#` 开头的行为注释）。

```text
# Dev 开发
Microsoft.VisualStudioCode
Git.Git
Python.Python

# 常用工具
7zip.7zip
ObsProject.OBSStudio
```

**查找包 ID 的方法**：

```powershell
winget search <关键词>
# 示例
winget search vscode
```

### macOS 软件清单

编辑 [`macOS/packages.txt`](./macOS/packages.txt)，每行填写一个 Homebrew Formula 或 Cask 名称。

```text
# 编程语言
Python@3
node.js

# 代码编辑器
Visual-Studio-Code
```

**查找包名的方法**：

```bash
brew search <关键词>
# 示例
brew search vscode
```

## 脚本说明

| 平台 | 脚本文件 | 功能描述 |
|------|----------|----------|
| Windows | `software_install.bat` | 标准模式批量安装，读取 `software_list.txt` |
| Windows | `software_install_proxy.bat` | 代理模式批量安装，依赖本地 v2rayN (`127.0.0.1:10809`) |
| Windows | `switch_winget_to_USTCsource.bat` | 切换 winget 源为中科大镜像，支持回滚 |
| macOS | `install_packages.sh` | 批量安装并自动切换 Homebrew 至中科大源 |

## 项目结构

```text
Software_Install_Script/
├── Windows/
│   ├── software_install.bat          # Windows 标准安装脚本
│   ├── software_install_proxy.bat    # Windows 代理安装脚本
│   ├── switch_winget_to_USTCsource.bat # 切换 winget 国内源
│   └── software_list.txt             # Windows 软件清单
├── macOS/
│   ├── install_packages.sh           # macOS 安装脚本
│   └── packages.txt                  # macOS 软件清单
├── bin/
│   └── sis-windows-amd64.exe         # 预编译 Windows 辅助工具
├── README.md                         # 项目说明文档
├── LICENSE-2.0                       # Apache License 2.0
└── AGENTS.md                         # 项目元数据与规范
```

## 贡献指南

欢迎提交 Issue 和 Pull Request！

1. **Fork** 本仓库。
2. 在您的分支上进行修改：`git checkout -b feature/YourFeature`。
3. 提交更改：`git commit -m 'Add some feature'`。
4. 推送分支：`git push origin feature/YourFeature`。
5. 新建一个 **Pull Request**。

### 提交规范

- 保持脚本简洁，不引入外部运行时依赖。
- 修改清单文件时，请在注释中注明软件类别。
- 若新增脚本，请在 `README.md` 中同步更新说明。

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
