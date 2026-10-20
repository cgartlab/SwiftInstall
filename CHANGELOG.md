# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Added

- CI 门禁接入 `go vet ./...`（修 #27）。

## [0.2.3] - 2026-02-20

### 说明

v0.2.0 ~ v0.2.2 的发布未维护 CHANGELOG 条目，相应变更见 [GitHub Releases](https://github.com/cgartlab/SwiftInstall/releases) 与 git tag。

## [v0.1.7] - 2026-02-19

### Fixed
- 修复 `install` 和 `search` 命令的 `help` 参数处理，现在正确显示帮助文档
- 修复 `search` 命令无参数时的行为，现在启动交互式搜索界面
- 修复搜索界面表格列标题不一致问题（Category → Publisher）
- 修复主菜单快捷键执行后状态混乱的问题
- 修复安装界面完成后只能按 Enter 退出的问题
- 修复 About 页面无交互的问题

### Added
- 搜索界面添加清晰的快捷键提示
- 安装界面改进帮助文本显示
- About 页面改为 TUI 交互模式，支持任意键返回
- 主菜单快捷键执行后正确退出并返回终端

### Changed
- 改进 `hasHelpArg()` 函数，支持在参数任意位置检测 help
- 优化搜索界面帮助文本，显示 "Add: Enter | Refine: / | Back: Esc | Quit: q"
- 优化安装界面完成后的退出提示 "Exit: Enter/Esc | About: a | Quit: q"
- 更新 README.md，添加完整的快捷键说明表格

## [v0.1.6] - 2026-02-19

### Added
- 新增自动更新检查功能，启动时可选择是否启用
- 新增环境预检功能，安装/搜索前自动检测包管理器状态
- 新增 Linux 平台支持（见 `internal/installer/linux.go`）
- 新增 `sis setup` 命令，一键自动完成环境检测、依赖准备与验证

### Changed
- 优化 setup 流程，支持 `--auto-install-deps` 和 `--dry-run` 参数
- 增强 release 日志记录模块
- 改进软件接口特性和用户体验

### Fixed
- 修复配置保存错误处理
- 修复安装模型指针接收器问题
- 修复并发安装中的 panic 恢复和错误处理问题

## [v0.1.5] - 2026-02-18

### Changed
- 合并多个特性分支，统一代码结构
- 优化软件接口特性

## [v0.1.4] - 2026-02-18

### Added
- 增强菜单导航系统
- 改进 help 系统，支持更完整的帮助文档

### Changed
- 更新项目文档和许可证信息

## [v0.1.3] - 2026-02-18

### Fixed
- 修复 `InstallModel` 并发安全问题，添加 `sync.Mutex` 保护共享数据
- 合并 `cmd/root.go` 中的重复 `init()` 函数
- 修复 `config.save()` 方法，保存所有配置项（language, theme, parallel_install 等）
- 修复 `getOSName()` 和 `getArch()` 使用 `runtime.GOOS` 和 `runtime.GOARCH` 动态获取系统信息
- 修复 Makefile 中 ldflags 包路径错误
- 添加 `validatePackageID()` 函数验证包 ID 格式，防止潜在命令注入
- 改进 `parseWingetLine()` 函数，使用正则表达式更好地解析含空格的软件名
- 修复 `IsInstalled()` 方法错误返回值，正确返回错误信息
- 修复 `Install()` 方法中 `IsInstalled` 错误被忽略的问题
- 修复 `BatchInstall()` 并发安装时缺少 panic 恢复机制
- 修复 `install.go` 中非并行模式实际仍是并发执行的问题
- 修复 `ShowSpinner()` goroutine 缺少 panic 恢复机制
- 修复 `installPackage()` 中 `Install` 错误被忽略的问题
- 添加数组越界检查防止 table 更新崩溃

### Added
- 添加 `internal/installer/installer_test.go` 测试文件
  - TestValidatePackageID: 包 ID 验证测试
  - TestParseWingetLine: winget 输出解析测试
  - TestCheckPackageManager: 包管理器检测测试
- 添加 `internal/config/config_test.go` 测试文件
  - TestConfigInit: 配置初始化测试
  - TestConfigSave: 配置保存测试
  - TestConfigPersistence: 配置持久化测试
  - TestConcurrentAccess: 并发访问测试
- 添加 `internal/i18n/i18n_test.go` 测试文件
  - TestTranslation: 翻译功能测试
  - TestLanguageSwitch: 语言切换测试
  - TestCompleteness: 翻译完整性测试
- 添加包级正则表达式缓存，优化性能

### Changed
- 优化并发安装时的线程安全性
- 改进 winget 输出解析的健壮性
- 统一错误处理模式，添加日志记录
- 改进并发安装的 semaphore 获取方式，避免主 goroutine 阻塞

## [0.1.2] - 2024-01-01

### Added
- 初始版本发布
- 支持 Windows (Winget) 和 macOS (Homebrew)
- 交互式 TUI 界面
- 多语言支持（中文/英文）
- 批量安装功能
- 配置文件管理
