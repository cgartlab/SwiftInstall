# 贡献指南

感谢贡献！SwiftInstall (`sis`) 是 Go 编写的跨平台批量软件安装器（Go 1.25.0）。

## 开发环境

- Go 1.25.0
- 提 PR 前：

```bash
go test ./...
go build ./...
```

## 目录约定

- `cmd/sis/` — 入口
- `internal/` — 核心包（cli / engine / manifest / backend / ui）
- `Windows/`、`macOS/` — 平台脚本
- `docs/` — 使用文档

## 提交规范

Conventional Commits：`feat` / `fix` / `docs` / `style` / `refactor` / `perf` / `test` / `chore` / `ci`

## 提 PR 前

- **单批次内某个软件包安装失败时不要中断整个批次**，错误在最后汇总中报告 —— 这是既有约定
- 涉及设计取舍 → 更新 `DESIGN_ISSUES.md`
- 涉及用户可见行为 → 更新 `README.md`
- 涉及特定平台 → 在对应平台实测

## 安全问题

不要在公开 issue 报告漏洞。
