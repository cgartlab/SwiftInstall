# SwiftInstall 测试报告

**测试日期**: 2026-05-11
**测试版本**: sis dev (none, unknown)
**测试环境**: Windows (PowerShell)

---

## 一、测试用例执行情况

### 1.1 命令可用性测试

| 用例 ID | 测试内容 | 命令 | 预期结果 | 实际结果 | 状态 |
|---------|---------|------|---------|---------|------|
| CMD-001 | 显示帮助信息 | `sis --help` | 显示可用命令列表 | 显示 install、status、version 等命令 | ✅ 通过 |
| CMD-002 | 显示版本信息 | `sis version` | 显示版本号 | `sis dev (none, unknown)` | ✅ 通过 |
| CMD-003 | install 子命令帮助 | `sis install --help` | 显示 install 用法和 flags | 显示 --file, --dry-run, --format 等 flags | ✅ 通过 |
| CMD-004 | status 子命令帮助 | `sis status --help` | 显示 status 用法和 flags | 显示 --file, --format 等 flags | ✅ 通过 |

### 1.2 sis install 测试

| 用例 ID | 测试内容 | 命令 | 预期结果 | 实际结果 | 状态 |
|---------|---------|------|---------|---------|------|
| INS-001 | 无 manifest 文件 | `sis install` | 报错提示找不到 manifest | `no manifest file found; use --file or create software_list.txt or sis.yaml` | ✅ 通过 |
| INS-002 | 指定 YAML manifest + dry-run | `sis install -f test_manifest_valid.yaml --dry-run` | 显示 3 个包预览安装 | 正确显示 3 个包，全部 success | ✅ 通过 |
| INS-003 | JSON 格式输出 | `sis install -f test_manifest_valid.yaml --dry-run --format json` | 输出 JSON 格式结果 | 输出合法 JSON，包含 total/succeeded/skipped/failed | ✅ 通过 |
| INS-004 | silent 格式输出 | `sis install -f test_manifest_valid.yaml --dry-run --format silent` | 仅显示 DRY-RUN 提示 | 仅显示 `DRY-RUN — no changes will be made` | ✅ 通过 |
| INS-005 | 空包列表 manifest | `sis install -f test_manifest_empty.yaml --dry-run` | 显示 0 个包，正常完成 | `Installing 0 packages... Summary: 0 total...` | ✅ 通过 |
| INS-006 | 重复包 ID 去重 | `sis install -f test_manifest_dup.yaml --dry-run` | 去重后安装 2 个包 | 正确去重，安装 2 个包 | ✅ 通过 |
| INS-007 | 空包 ID 过滤 | `sis install -f test_manifest_invalid.yaml --dry-run` | 过滤空 ID，安装 1 个包 | 过滤空 ID，只安装 Git.Git | ✅ 通过 |
| INS-008 | 损坏的 YAML | `sis install -f test_manifest_broken.yaml --dry-run` | 解析错误退出 | `parse manifest: parse YAML: yaml: line 2...` | ✅ 通过 |
| INS-009 | 不存在的文件 | `sis install -f nonexistent.yaml --dry-run` | 文件不存在错误 | `parse manifest: read manifest: open nonexistent.yaml...` | ✅ 通过 |
| INS-010 | TXT 格式 manifest | `sis install -f test_manifest_txt.txt --dry-run` | 正确解析并安装 3 个包 | 正确解析 TXT 格式，安装 3 个包 | ✅ 通过 |
| INS-011 | 自动查找 sis.yaml | `sis install --dry-run` (存在 sis.yaml) | 自动找到并执行 | 自动找到 sis.yaml，安装 1 个包 | ✅ 通过 |
| INS-012 | 自动查找失败 | `sis install --dry-run` (不存在 sis.yaml) | 提示找不到 manifest | `no manifest file found...` | ✅ 通过 |

### 1.3 sis status 测试

| 用例 ID | 测试内容 | 命令 | 预期结果 | 实际结果 | 状态 |
|---------|---------|------|---------|---------|------|
| STA-001 | 无 manifest 文件 | `sis status` | 报错提示找不到 manifest | `no manifest file found` | ✅ 通过 |
| STA-002 | 检查已安装包状态 | `sis status -f test_manifest_valid.yaml` | 显示各包安装状态 | 显示 3 个包状态（根据实际安装情况） | ✅ 通过 |
| STA-003 | JSON 格式输出 | `sis status -f test_manifest_valid.yaml --format json` | 输出 JSON 格式结果 | 输出合法 JSON | ⚠️ 发现 Bug (BUG-003) |
| STA-004 | 混合状态包 | `sis status -f test_manifest_optional.yaml` | 显示已安装和未安装包 | 正确显示 ✓ 和 ⚠ 状态 | ✅ 通过 |

### 1.4 异常输入和边界条件

| 用例 ID | 测试内容 | 命令 | 预期结果 | 实际结果 | 状态 |
|---------|---------|------|---------|---------|------|
| BND-001 | 无效 format 值 | `sis install -f test_manifest_valid.yaml --format invalid_format --dry-run` | 报错或回退到默认格式 | 回退到 terminal 格式，正常执行 | ⚠️ 发现 Bug (BUG-002) |
| BND-002 | 未知 YAML 字段 | `sis install -f test_manifest_unknown_fields.yaml --dry-run` | 报错提示未知字段 | 严格报错，拒绝未知字段 | ✅ 通过（设计如此） |

---

## 二、发现的 Bug 清单

### BUG-001: status 命令使用 install 的提示文本

- **严重程度**: 🟡 中
- **复现步骤**:
  1. 运行 `sis status -f test_manifest_valid.yaml`
- **预期结果**: 提示文本应为 "Checking status of N packages from ..."
- **实际结果**: 提示文本显示 "Installing N packages from ..."
- **根因分析**: `status.go` 使用了 `TerminalRenderer`，其 `Start()` 方法硬编码了 "Installing" 文本，没有区分 install 和 status 场景
- **影响范围**: 仅影响 status 命令的终端输出，不影响功能正确性
- **修复建议**: 修改 `Renderer` 接口的 `Start` 方法，增加操作类型参数；或创建独立的 StatusRenderer

### BUG-002: 无效的 --format 值未报错

- **严重程度**: 🟡 中
- **复现步骤**:
  1. 运行 `sis install -f test_manifest_valid.yaml --format invalid_format --dry-run`
- **预期结果**: 应报错提示无效的 format 值，并列出支持的格式
- **实际结果**: 回退到默认的 terminal 格式，静默执行
- **根因分析**: `install.go` 和 `status.go` 中的 format switch 使用 default case 回退到 terminal，没有校验输入值
- **影响范围**: 用户输入错误格式时得不到反馈，可能导致困惑
- **修复建议**: 在 switch 前增加 format 值校验，无效时返回错误

### BUG-003: status 命令 JSON 输出中时间戳为零值

- **严重程度**: 🟢 低
- **复现步骤**:
  1. 运行 `sis status -f test_manifest_valid.yaml --format json`
- **预期结果**: `start_time` 和 `end_time` 应为当前时间或合理值
- **实际结果**: `"start_time": "0001-01-01T00:00:00Z", "end_time": "0001-01-01T00:00:00Z"`
- **根因分析**: `status.go` 手动构造 `engine.Summary` 时没有设置 `StartTime` 和 `EndTime` 字段
- **影响范围**: 仅影响 status 命令的 JSON 输出中的时间字段
- **修复建议**: 在 `status.go` 中构造 Summary 时设置 `StartTime: time.Now()` 和 `EndTime`

### BUG-004: install 命令空包 ID 未在 Validate 中拦截

- **严重程度**: 🟢 低
- **复现步骤**:
  1. 创建包含 `id: ""` 的 manifest
  2. 运行 `sis install -f test_manifest_invalid.yaml --dry-run`
- **预期结果**: `manifest.Validate()` 应报错 "package ID cannot be empty"
- **实际结果**: 空 ID 包被静默过滤，只安装非空 ID 的包
- **根因分析**: `parser.go` 的 `parseYAML` 在解析时过滤了空 ID，导致 Validate 看不到空 ID
- **影响范围**: 用户可能不知道 manifest 中有空 ID 包被忽略
- **修复建议**: 统一校验逻辑——要么在 parser 中保留空 ID 让 Validate 报错，要么在 parser 中记录警告

### BUG-005: optional 字段未被 engine 使用

- **严重程度**: 🟢 低
- **复现步骤**:
  1. 创建包含 `optional: true` 的 manifest
  2. 运行 `sis install -f test_manifest_optional.yaml --dry-run`
- **预期结果**: optional 包安装失败时不应影响整体退出码
- **实际结果**: dry-run 模式下所有包都显示 success，无法验证 optional 行为；且 engine 中没有读取 `pkg.Optional` 字段
- **根因分析**: `engine.go` 的 `Install` 方法没有使用 `pkg.Optional` 字段来决定失败策略
- **影响范围**: optional 字段声明了但无实际效果
- **修复建议**: 在 engine 中根据 `pkg.Optional` 决定是否将失败计入 summary.Failed 或影响退出码

---

## 三、测试总结

### 通过率统计

| 类别 | 总用例 | 通过 | 失败 | 通过率 |
|------|--------|------|------|--------|
| 命令可用性 | 4 | 4 | 0 | 100% |
| install 功能 | 12 | 12 | 0 | 100% |
| status 功能 | 4 | 3 | 1 | 75% |
| 边界条件 | 2 | 1 | 1 | 50% |
| **总计** | **22** | **20** | **2** | **90.9%** |

### Bug 严重程度分布

| 严重程度 | 数量 |
|---------|------|
| 🔴 高 | 0 |
| 🟡 中 | 2 |
| 🟢 低 | 3 |
| **总计** | **5** |

### 结论

SwiftInstall 重构后的核心功能（install、status、version）基本可用。主要问题集中在：

1. **UI 文本不准确**（BUG-001）: status 命令错误显示 "Installing"
2. **输入校验不完善**（BUG-002）: 无效 format 值未报错
3. **JSON 输出不完整**（BUG-003）: status JSON 时间戳为零值
4. **校验逻辑不一致**（BUG-004）: 空包 ID 在 parser 中被过滤而非 Validate 拦截
5. **声明未实现**（BUG-005）: optional 字段无实际效果

建议优先修复 BUG-001 和 BUG-002，这两个问题直接影响用户体验。
