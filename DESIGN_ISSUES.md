# SwiftInstall 设计缺陷与修复方案

本文档记录对 SwiftInstall Go CLI（`sis`）的架构审查发现，按严重程度优先级（P0→P3）排列。
每条问题附有：

- 缺陷描述与代码定位
- 详细修复方案（What & How）
- 风险评估与规避策略（防止修 A 坏 B）

---

## 勘误与更新记录

本文件所有修复方案均经过官方文档验证。以下列出修正记录：

| 版本 | 修正内容 | 依据 |
|------|---------|------|
| v1.1 | **#7.2 `list --output json` 方案移除** — winget `list` 命令不支持 `--output` 参数。替代方案改用 `winget list --id <id> --exact` 精确匹配 + 文本解析加固 | [winget list 官方文档](https://learn.microsoft.com/en-us/windows/package-manager/winget/list#options) |
| v1.1 | **#3.2 `source add` 命令格式校正** — 官方文档示例使用 `--name` 标志：`winget source add --name Contoso https://www.contoso.com/cache`。现有代码 `winget source add winget <url>` 缺少 `--name`，在严格模式下可能失败。需改为 `--name winget [-a] <url>` | [winget source 官方文档](https://learn.microsoft.com/en-us/windows/package-manager/winget/source#add) |
| v1.1 | **winget 退出码** — 官方文档未列出 `0x8A150011` 等具体退出码，这些来自社区经验。保留现有映射，但文档应注明来源 | [winget install 官方文档](https://learn.microsoft.com/en-us/windows/package-manager/winget/install) |
| v1.1 | **新增 `--no-upgrade` 发现** — winget `install` 支持 `--no-upgrade` 选项（"Skips upgrade if an installed version already exists"）。可作为 `--skip-existing` 的简化实现方案。见 #4 补充 | [winget install 官方文档](https://learn.microsoft.com/en-us/windows/package-manager/winget/install#options) |

---

## 目录

1. [P0 — 零测试覆盖](#p0--零测试覆盖)
2. [P0 — Config JSON 解析错误被静默吞掉](#p0--config-json-解析错误被静默吞掉)
3. [P1 — CLI 命令层过厚（God Handler）](#p1--cli-命令层过厚god-handler)
4. [P1 — exec.Command 无超时保护](#p1--execcommand-无超时保护)
5. [P1 — DRY 重复逻辑](#p1--dry-重复逻辑)
6. [P2 — 依赖倒置违背（DIP）](#p2--依赖倒置违背dip)
7. [P2 — 开闭原则违背（OCP）](#p2--开闭原则违背ocp)
8. [P2 — Winget 输出解析脆弱](#p2--winget-输出解析脆弱)
9. [P3 — 并发安全问题](#p3--并发安全问题)
10. [P3 — 僵尸代码（logger / --verbose）](#p3--僵尸代码logger---verbose)
11. [P3 — 串行安装性能](#p3--串行安装性能)

---

## P0 — 零测试覆盖

### 问题

仓库中没有任何 `*_test.go` 文件。核心逻辑——Engine Install/Uninstall/去重/重试、Manifest 解析、Config 合并/校验——全部无法通过自动化测试验证。重构或添加功能时只能靠手动测试，回归风险极高。

### 代码定位

- `internal/engine/engine.go` — 去重、重试、进度回调、Context 取消
- `internal/engine/manifest.go` — YAML + TXT 两种格式、BOM 处理、行内注释
- `internal/engine/check.go` — 预检套件
- `internal/config/config.go` — 两级 merge、Validate、sanitize、Save
- `internal/backend/winget.go` — parseWingetListOutput、IsInstalled

### 修复方案

#### Step 1：创建 FakeBackend（`internal/backend/fake.go`）

```go
package backend

type FakeBackend struct {
    NameFn         func() string
    DetectFn       func() error
    InstalledMap   map[string]bool      // package ID → installed?
    InstallFn      func(...) (*Output, error)
    UninstallFn    func(...) (*Output, error)
    ListInstalledFn func() ([]string, error)
    PermanentErrors []error
}
```

遵循 `Backend` 接口，所有字段可选默认行为。Install/Uninstall 支持注入错误以测试重试逻辑和永久错误短路。

#### Step 2：Engine 测试（`internal/engine/engine_test.go`）

| 测试用例 | 场景 |
|---------|------|
| TestInstall_Success | 正常安装列表，验证 Summary 字段 |
| TestInstall_DryRun | DryRun 模式不调用后端 |
| TestInstall_SkipExisting | 已安装包被跳过 |
| TestInstall_Dedup | 重复包只安装一次 |
| TestInstall_RetrySuccess | 前 2 次失败 → 第 3 次成功 |
| TestInstall_RetryExhausted | 达到最大重试次数后报告失败 |
| TestInstall_PermanentError | IsPermanent 返回 true → 立即停止重试 |
| TestInstall_ContextCancel | 取消后停止处理剩余包 |
| TestInstall_EmptyManifest | 空清单 |
| TestUninstall_DryRun | 同上 |
| TestCheckStatus | 对比 manifest 与 installedSet |
| TestProgressHook | 每个包触发一次回调 |
| TestConcurrentSetProgressHook | -race 下不会 data race（配合修复 #9） |

#### Step 3：Manifest 解析测试（`internal/engine/manifest_test.go`）

| 测试用例 | 场景 |
|---------|------|
| TestParseYAML_Normal | 标准 YAML：mirror、proxy、packages、category |
| TestParseYAML_Empty | 空 YAML → 0 个包 |
| TestParseYAML_Duplicates | 重复 ID 自动去重 |
| TestParseYAML_KnownFields | 未知字段报错 |
| TestParseYAML_BOM | UTF-8 BOM 正确处理 |
| TestParseTXT_Normal | 单行、注释作为 category、行内注释 |
| TestParseTXT_EmptyLines | 空行跳过 |
| TestParseTXT_Duplicates | 去重 |
| TestParseTXT_BOM | BOM 处理 |
| TestParse_UnknownExt | .json 等尝试 YAML 回退 TXT |
| TestParse_FileNotFound | 文件不存在返回错误 |

#### Step 4：Config 测试（`internal/config/config_test.go`）

| 测试用例 | 场景 |
|---------|------|
| TestLoad_OnlyGlobal | 只有全局配置 |
| TestLoad_OnlyLocal | 只有本地配置 |
| TestLoad_Both | 本地覆盖全局 |
| TestLoad_NoFiles | 返回默认值 |
| TestLoad_InvalidJSON_Global | 全局 JSON 破损 → 使用默认（配合修复 #2 输出警告） |
| TestMerge | 逐字段覆盖检查 |
| TestValidate_Valid | 各字段合法值通过 |
| TestValidate_Invalid | 非法值返回错误 |
| TestSanitize | 非法值重置为默认 |
| TestSave_Global | 写入 ~/.sis/config.json |
| TestSave_Local | 写入 .sis.json |
| TestSave_Then_Load_Roundtrip | 写后读一致性 |
| TestGetSet | 每个 Get/Set 键 |

#### Step 5：Winget 解析测试（`internal/backend/winget_test.go`）

| 测试用例 | 场景 |
|---------|------|
| TestParseWingetListOutput | 模拟 winget list 输出 |
| TestParseWingetListOutput_Empty | 空/只有表头 |
| TestParseWingetListOutput_MinWidth | 紧凑格式 |
| TestFormatWingetError | 所有已知 exit code |

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| FakeBackend 与真实后端行为不一致 | 中 | 增加集成测试（单独 tag `integration`，CI 中跑在有 winget 的机器上） |
| 测试框架依赖影响 go.mod | 低 | 测试只使用标准库 `testing`，无需额外依赖 |
| 不合理的测试 timeout 导致 CI 假阳性 | 低 | 测试都控制在毫秒级，FakeBackend 同步返回 |
| 测试覆盖率指标导致团队写无价值测试 | 低 | 文档明确：优先测逻辑路径（条件分支、错误处理），不追求行覆盖率的数字指标 |

---

## P0 — Config JSON 解析错误被静默吞掉

### 问题

`config.Load()` (config.go:75, 85) 在 `json.Unmarshal` 失败时，轻率跳过并继续使用默认值，**没有输出任何警告**。如果用户的 `~/.sis/config.json` 被手误编辑为非法 JSON，所有配置静默丢失，用户完全无感。

### 代码定位

```go
// config.go:72-80
if data, err := os.ReadFile(gp); err == nil {
    var gc Config
    if err := json.Unmarshal(data, &gc); err == nil {
        merge(cfg, &gc)
    }
}
// Unmarshal 失败 → 此处静默跳过
```

### 修复方案

```go
if data, err := os.ReadFile(gp); err == nil {
    var gc Config
    if err := json.Unmarshal(data, &gc); err != nil {
        // 输出警告到 stderr，内容包括文件路径和具体错误
        fmt.Fprintf(os.Stderr,
            "warning: %s contains invalid JSON (%v); using default values for this file\n",
            gp, err)
    } else {
        merge(cfg, &gc)
    }
}
```

同理处理 localPath `.sis.json`。

**关键设计决策：** 不直接返回错误退出——如果仅全局配置损坏但本地配置正常，允许部分恢复。如果两者都损坏/不存在，使用内置默认值。

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| 用户之前依赖静默吞错行为，warning 信息造成困扰 | 低 | warning 只输出到 stderr，不影响 stdout 结果 |
| 用户误将警告当作错误，以为程序故障 | 低 | 措辞明确使用 "warning:" 前缀，提示 "using default values" |
| fmt.Fprintf 在 config 包中引入了对 cli/engine 的输出耦合 | 低 | config 包已有底层输出需求，可统一使用自定义 `Warningf` 函数或返回警告列表由调用方处理 |

**推荐：** 在 `Load()` 中收集警告列表，通过一个 `type LoadResult struct { Config *Config; Warnings []string }` 传递，由调用方决定如何展示。但这会破坏当前 API。折中方案：先加 stderr 输出，后续引入 LoadResult。

---

## P1 — CLI 命令层过厚（God Handler）

### 问题

`cli/install.go` 的 `RunE` 函数（约 70 行实际逻辑）在一个闭包中完成了：
1. 加载配置
2. 解析命令行参数
3. 解析 manifest 路径
4. 代理检测（手动 + 自动 + manifest 级）
5. 镜像切换
6. 后端检测
7. 预检运行 + 输出
8. Manifest 解析
9. 引擎创建 + 参数注入
10. 安装执行
11. 结果汇总输出

这违背单一职责原则和分离关注点。新增命令（如 `update`、`upgrade`）时同样的 preamble 需要重复编写。`status.go` 和 `uninstall.go` 已经出现了大量重复的 preamble 代码。

### 修复方案

提取一个 `Orchestrator` 层（`internal/engine/orchestrator.go`），将安装/卸载/状态检查的公共 preamble 封装为方法：

```go
package engine

type InstallOrchestrator struct {
    // 只保留构造时注入的依赖，不依赖 CLI 层
}

// PrepareOptions 收集所有来自 CLI/Config 的参数
type PrepareOptions struct {
    ManifestPath    string
    Mirror          string
    Proxy           string
    ProxyAutoDetect bool
    SkipChecks      bool
    DryRun          bool
    SkipExisting    bool
    RetryCount      int
    RetryDelaySec   int
}

// Prepare 执行所有前置操作，返回就绪的 Engine + Manifest
func (o *InstallOrchestrator) Prepare(ctx context.Context, opts *PrepareOptions) (*Engine, *Manifest, error)
```

CLI 层的 `RunE` 缩减为：

```go
RunE: func(cmd *cobra.Command, args []string) error {
    opts, err := buildOptions(cmd.Flags())
    if err != nil { return err }

    result, err := orch.Install(ctx, opts)
    if err != nil { return err }

    renderSummary(result) // 只负责输出
    return nil
}
```

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| 提取过度导致抽象泄漏（Orchestrator 需要导入太多东西） | 中 | 限制 Orchestrator 只依赖 `backend.Backend` 接口 + `config.Config`，不依赖 cobra。所有 flag 参数在 CLI 层转换为 Plain Old Go Types |
| 破坏现有功能 | 中 | 分步进行：(1) 原地提取 PrepareOptions / 方法，不改变调用逻辑 (2) 确认测试通过后 (3) 将其移至独立的 orchestrator.go 文件 |
| 为"可能未来需要的命令"做了过度设计 | 低 | 不提前设计 update 等命令的抽象。只提取当前明确重复的部分，遵循 YAGNI |

---

## P1 — exec.Command 无超时保护

### 问题

多处 `exec.Command` 调用没有使用 context timeout，如果 winget/tasklist 进程挂起，CLI 将无限阻塞：

| 位置 | 调用 | 风险 |
|------|------|------|
| `mirror.go:42` | `exec.Command("winget", "source", "add", ...).Run()` | 网络不通时 hang |
| `mirror.go:60` | `exec.Command("winget", "source", "reset", ...)` | 同上 |
| `mirror.go:75` | `exec.Command("winget", "source", "remove", ...)` | 同上 |
| `mirror.go:111` | `exec.Command("winget", "source", "list").Output()` | 同上 |
| `proxy.go:23` | `exec.Command("tasklist", ...).Output()` | Windows 系统服务异常时 hang |

即使使用了 `exec.CommandContext` 的地方（winget.go），调用方传递的是 `context.Background()`，等同于没有 timeout。

### 修复方案

#### 3.1 在配置中添加全局 timeout

```go
type Config struct {
    // 新增字段
    InstallTimeoutSec *int `json:"install_timeout_sec,omitempty"`
    NetworkTimeoutSec *int `json:"network_timeout_sec,omitempty"`
}
```

默认值：InstallTimeoutSec=300（5分钟），NetworkTimeoutSec=30。

#### 3.2 修改所有 exec.Command 调用链路

- `Backend.Install(ctx)` / `Backend.Uninstall(ctx)` / `Backend.ListInstalled(ctx)` — 在调用方（Engine）创建带 timeout 的 context，而非在 Backend 内
- `mirror.Set()`, `mirror.Reset()`, `mirror.Current()` — 暴露为 `Set(ctx context.Context, ...)`，在 CLI 层注入 timeout context
- `proxy.Detect()` — 改为 `Detect(ctx context.Context)`，传入 5s timeout

#### 3.3 修改示例

**`source add` 命令格式校正：** [官方文档](https://learn.microsoft.com/en-us/windows/package-manager/winget/source#add) 示例使用 `--name` 标志：
```
winget source add --name Contoso https://www.contoso.com/cache
```
当前代码 `exec.Command("winget", "source", "add", "winget", url)` 没有传递 `--name`，需要修正。

```go
// mirror.go:40-50 当前：
func Set(name string) error {
    // ...
    removeSource("winget")

    if err := exec.Command("winget", "source", "add", "winget", def.WingetSourceURL).Run(); err != nil {
        //                                     ^^^^^^ 缺少 --name 标志

// 改为：
func Set(ctx context.Context, name string) error {
    // ...
    removeSource("winget")

    cmd := exec.CommandContext(ctx, "winget", "source", "add", "--name", "winget", def.WingetSourceURL)
    if err := cmd.Run(); err != nil { ... }
```

CLI 调用侧：

```go
ctx, cancel := context.WithTimeout(context.Background(), cfg.NetworkTimeoutSec())
defer cancel()
if err := mirror.Set(ctx, mirrorFlag); err != nil { ... }
```

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| timeout 太小导致正常慢安装中断 | 中 | 默认 300s 对 winget 安装足够宽松；用户可配置；日志记录实际耗时供调优参考 |
| 不兼容的 API 变更（mirror.Set 等现在需要 ctx） | 中 | 一次性更新所有调用方即可，不影响外部接口（mirror 包是 internal 的） |
| time.After 泄漏 goroutine | 低 | context.WithTimeout 会自动清理，time.After 用在 retry 中也是合理的 |
| winget source 操作在网络差时需要更长时间 | 低 | 统一通过 `cfg.NetworkTimeoutSec` 配置，用户可按需调整 |

---

## P1 — DRY 重复逻辑

### 问题

| 位置 | 重复内容 |
|------|---------|
| `config.go` - `Validate()` 与 `sanitize()` | LogLevel、Color、Mirror、Proxy 的校验列表完全重复，条目新增时需同时修改两处 |
| `engine.go` - `installWithRetry()` 与 `uninstallWithRetry()` | 重试循环结构完全一致，仅 fn 内容不同 |
| `engine.go` - `Install()` 与 `Uninstall()` | 包迭代、进度回调、Summary 构建结构高度相似 |
| `cli/install.go` / `status.go` / `uninstall.go` | preamble（加载配置 → 路径解析 → 后端创建 → manifest 解析 → 引擎创建）重复 3 次 |

### 修复方案

#### 4.1 Config Validate + Sanitize 去重

创建共享校验器：

```go
// config/validators.go
var validLogLevels = setOf("debug", "info", "warn", "error")
var validColors = setOf("auto", "always", "never")
var validMirrors = setOf("ustc", "official")

func validateString(value string, validSet map[string]bool, name string) bool {
    return validSet[strings.ToLower(value)]
}
```

`Validate()` 和 `sanitize()` 都调用 `validateString` 做校验，避免两份独立的条件分支。

#### 4.2 Engine 重循环去重

将 `Install()` 和 `Uninstall()` 的公共迭代逻辑提取为 `forEachPackage` 方法：

```go
type packageAction func(ctx context.Context, pkg Package) InstallResult

func (e *Engine) forEachPackage(ctx context.Context, packages []Package, action packageAction) *Summary {
    start := time.Now()
    summary := &Summary{StartTime: start, Total: len(packages)}

    for _, pkg := range packages {
        select {
        case <-ctx.Done():
            ...
        default:
        }
        result := action(ctx, pkg)
        summary.Results = append(summary.Results, result)
        e.fireProgress(result)
    }
    ...
}
```

`Install()` 和 `Uninstall()` 分别构造自己的 `packageAction` 闭包。

#### 4.3 重试逻辑提取

`installWithRetry()` 和 `uninstallWithRetry()` 合并为一个函数：

```go
func (e *Engine) retryOperation(ctx context.Context, id string, op string) error {
    return retry(ctx, e.cfg.RetryCount, e.cfg.RetryDelay, func() error {
        var err error
        switch op {
        case "install":
            _, err = e.backend.Install(ctx, id, backend.InstallOptions{...})
        case "uninstall":
            _, err = e.backend.Uninstall(ctx, id, backend.InstallOptions{...})
        }
        if err != nil && e.backend.IsPermanent(err) {
            return stop(err)
        }
        return err
    })
}
```

如果后续 install/uninstall 逻辑开始分化（不同的 install options），再拆分回来。

#### 4.4 CLI preamble 去重

配合 #3（Orchestrator），在提取的 `Prepare()` 方法中集中处理 preamble。

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| Validate/sanitize 逻辑统一后，一处 bug 影响两个路径 | 中 | 通过 Step 1 的 config 测试覆盖所有边界值，确保两者行为一致 |
| 合并重试逻辑后 install/uninstall 参数分化 | 低 | 保持 `retryOperation` 简单：接受 `op string` + 自定义 `InstallOptions` 参数；如果选项差异增大再拆分 |
| 提取 forEachPackage 改变 Summary 计算逻辑 | 中 | 确保 forEachPackage 前后计算 Succeeded/Skipped/Failed 的方式与原来完全一致，由 Step 2 的 engine 测试验证 |

---

## P2 — 依赖倒置违背（DIP）

### 问题

CLI 命令直接创建具体后端类型：

```go
// cli/install.go:83
winget := backend.NewWingetBackend()
```

未来添加 macOS Homebrew 后端时，需要在每个命令入口添加平台判断。违反"依赖抽象而非具体实现"原则。

### 修复方案

#### 5.1 添加 Backend 工厂

```go
// backend/factory.go
package backend

import "runtime"

func NewBackend() Backend {
    switch runtime.GOOS {
    case "windows":
        return NewWingetBackend()
    case "darwin":
        return NewHomebrewBackend()
    default:
        return NewWingetBackend() // fallback
    }
}
```

CLI 调用改为：

```go
be := backend.NewBackend()
if err := be.Detect(); err != nil {
    return fmt.Errorf("package manager: %w", err)
}
```

#### 5.2 支持通过环境变量或 flag 覆盖

```go
// 如果 SIS_BACKEND 设置了 "winget" 或 "homebrew"，强制使用
func NewBackend() Backend {
    if forced := os.Getenv("SIS_BACKEND"); forced != "" {
        switch forced {
        case "winget": return NewWingetBackend()
        case "homebrew": return NewHomebrewBackend()
        }
    }
    // auto-detect by platform
}
```

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| 自动选择后，Windows 上 ping winget 不在 PATH 时不会自动 fallback | 低 | `Detect()` 失败时返回明确错误，用户可手动指定 |
| HomebrewBackend 尚未实现，代码中存在 `NewHomebrewBackend()` 残桩可能导致编译问题 | 低 | TODO 标记 + 编译时 mock: 在 `build_darwin.go` / `build_windows.go` 中用 build tags 条件编译，不存在 Homebrew 的平台上不编译 |
| 现有代码在运行时才能发现后端不可用（之前是编译期通过调用 `NewWingetBackend` 明确知道） | 低 | `Detect()` 会在安装前执行并返回错误，不影响运行正确性 |

---

## P2 — 开闭原则违背（OCP）

### 问题

**Mirror 包** (`mirror.go:11`)：mirror 定义硬编码在包级 map 中：

```go
var mirrors = map[string]MirrorDef{
    "ustc":     {Name: "ustc", WingetSourceURL: "https://mirrors.ustc.edu.cn/winget-source"},
    "official": {Name: "official", WingetSourceURL: ""},
}
```

新增镜像源需要修改源码 + 重新编译。

**Config 包** (`config.go:170-246`)：`Get`/`Set` 使用 switch 穷举所有 key，新增配置项需要修改 struct + switch + `merge` + `validKeys` + `defaults`。

### 修复方案

#### 6.1 Mirror 注册机制

```go
// mirror/registry.go
var registry = map[string]MirrorDef{}

func Register(name string, def MirrorDef) {
    registry[name] = def
}

func init() {
    Register("ustc", MirrorDef{
        WingetSourceURL: "https://mirrors.ustc.edu.cn/winget-source",
    })
    Register("official", MirrorDef{})
}
```

后续新增镜像源只需在 `init()` 中注册，不修改已有文件。也可以从外部配置文件加载。

#### 6.2 Config Get/Set 去穷举

使用反射或值映射表：

```go
var configFields = map[string]configField{
    "mirror":           {path: "Mirror", kind: reflect.String},
    "proxy":            {path: "Proxy", kind: reflect.String},
    "proxy_auto_detect": {path: "ProxyAutoDetect", kind: reflect.Bool},
    // ...
}

func Get(cfg *Config, key string) (any, error) {
    field, ok := configFields[key]
    if !ok {
        return nil, fmt.Errorf("unknown config key: %s", key)
    }
    v := reflect.ValueOf(cfg).Elem().FieldByName(field.path)
    // 根据 kind 处理 pointer 或 value
    ...
}
```

Set 同理。

**注意：** 反射方案在 Go 中代价低，但可读性不如显式 switch。另一种方案：将每个字段的处理逻辑收集到一个 map 中：

```go
var configHandlers = map[string]struct{
    get func(*Config) any
    set func(*Config, string) error
    validate func(string) bool
}
```

但此方案的 boilerplate 反而更多。**推荐使用反射**，配合 `configFields` 表一目了然，新增配置项只需要添加一行表条目。

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| 反射 panic（字段名拼写错误、类型不匹配） | 中 | 在 `init()` 中做注册时自检：`mustRegister(field)` 在启动时反射验证路径有效 |
| 失去编译期类型检查 | 中 | 针对 `configFields` 表写一个 `TestConfigFieldsValid` 测试，确保所有条目 path 确实对应 struct field |
| Mirror 注册在 init() 中，测试时被污染 | 低 | 测试用 `t.Cleanup(func() { /* 恢复 registry */ })` |

---

## P2 — Winget 输出解析脆弱

### 问题

`parseWingetListOutput` (`winget.go:83`) 解析 winget list 输出的方式非常脆弱：

```go
lines := strings.Split(output, "\n")
if len(lines) < 3 { return nil }
lines = lines[2:] // 跳过前两行（header + 分隔线）
parts := strings.Fields(line)
if len(parts) >= 2 { id := parts[1] } // 硬编码取第二列
```

问题：
1. Winget 输出格式不保证列对齐 —— 包名含空格时 `strings.Fields` 会分错
2. `IsInstalled` (`winget.go:33`) 使用 `strings.Contains` 做子串匹配 —— 查 `Git` 会误匹配 `GitLFS`

### 修复方案

#### 7.1 IsInstalled 改用 --exact 和精确匹配

```go
func (w *WingetBackend) IsInstalled(ctx context.Context, id string) (bool, error) {
    cmd := exec.CommandContext(ctx, w.execPath, "list", "--id", id, "--exact", "--accept-source-agreements")
    out, err := cmd.Output()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
            return false, nil // winget list 返回非零表示未找到
        }
        return false, err
    }
    // 从 winget list 输出中逐行检查是否有精确匹配
    for _, line := range strings.Split(string(out), "\n") {
        fields := strings.Fields(line)
        for _, f := range fields {
            if strings.EqualFold(f, id) {
                return true, nil
            }
        }
    }
    return false, nil
}
```

#### 7.2 ListInstalled 文本解析加固（代替 JSON）

**勘误：** 此前假设 winget `list` 支持 `--output json`，但[官方文档](https://learn.microsoft.com/en-us/windows/package-manager/winget/list#options)中 `list` 命令没有 `--output` 参数。`winget export` 虽然输出 JSON，但它的 `-o` 参数接受的是文件路径（非 `-`/stdout），且 export 会扫描所有包并产生 warning，不适合频繁调用。

因此加固文本解析是更务实的选择：

```go
func parseWingetListOutput(output string) []string {
    lines := strings.Split(output, "\n")
    if len(lines) < 2 {
        return nil
    }

    // Step 1: 找到表头行（包含 "Id" 的行），确定 Id 列的索引
    headerIdx := -1
    idColIdx := -1
    for i, line := range lines {
        if strings.Contains(line, "Id") && strings.Contains(line, "Name") {
            headerIdx = i
            headers := strings.Fields(line)
            for j, h := range headers {
                if h == "Id" || h == "Id" {
                    idColIdx = j
                    break
                }
            }
            break
        }
    }
    if headerIdx == -1 || idColIdx == -1 {
        return nil
    }

    var ids []string
    for _, line := range lines[headerIdx+2:] { // 跳过表头 + 分隔线
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        fields := strings.Fields(line)
        if len(fields) > idColIdx {
            id := fields[idColIdx]
            if id != "" && id != "Id" {
                ids = append(ids, id)
            }
        }
    }
    return ids
}
```

**替代方案（备选）：** `winget export -o <tempfile>` + 删除临时文件。但会产生磁盘 I/O 和 package scanning 开销，仅当文本解析在实测中不可靠时考虑。

#### 7.3 IsPermanent 匹配加固

当前 `IsPermanent` 通过子串匹配错误信息判断。改用官方翼展错误码映射表（从 [winget-cli github issues](https://github.com/microsoft/winget-cli/issues) 中维护的已知退出码集合）：

```go
var permanentExitCodes = map[int]bool{
    0x8A150019: true, // 包未找到
    0x8A15003F: true, // 无可用包
}
```

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| winget 列格式随版本变化（列名更改、增加列） | 低 | 通过搜索 "Id" 和 "Name" 同时存在来识别表头，容忍列顺序变化 |
| 包 ID 含空格时 Fields 分词错误 | 低 | winget 的包 ID 规范不允许空格，格式为 `Publisher.Package`；若有此类包，通过 JSON 导出方案兜底 |
| 退出码映射不完整 | 中 | 初始使用社区已知集合，后续在 winget-cli 源码中搜索 `AppInstallerCliError` 枚举补充 |

---

## P3 — 并发安全问题

### 问题

`Engine.progressFn` 字段没有同步保护：

```go
// engine.go:30-32
type Engine struct {
    // ...
    progressFn func(InstallResult)
}

// engine.go:39-41 外部 setter
func (e *Engine) SetProgressHook(fn func(InstallResult)) {
    e.progressFn = fn
}

// engine.go:163-165 内部读
func (e *Engine) fireProgress(r InstallResult) {
    if e.progressFn != nil {
        e.progressFn(r)
    }
}
```

如果用户（或未来并行安装特性）在 `Install()` 尚未完成时调用 `SetProgressHook`，存在 data race。

### 修复方案

#### 8.1 最优方案：通过构造函数注入，移除 Setter

```go
func New(be backend.Backend, cfg *Config) *Engine
// 改为
func New(be backend.Backend, cfg *Config, progressFn func(InstallResult)) *Engine
```

**优势**：`progressFn` 在构造后不再可变，完全避免了 race。

**影响**：当前 CLI 调用 `New()` 后再 `SetProgressHook()`，需要将进度回调前移到构造函数。

#### 8.2 如果保留 Setter（渐进式，不改调用方）

```go
type Engine struct {
    // ...
    progressFn func(InstallResult)
    mu         sync.RWMutex   // 新增
}

func (e *Engine) SetProgressHook(fn func(InstallResult)) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.progressFn = fn
}

func (e *Engine) getProgressFn() func(InstallResult) {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.progressFn
}

func (e *Engine) fireProgress(r InstallResult) {
    fn := e.getProgressFn()  // 替代直接读 e.progressFn
    if fn != nil {
        fn(r)
    }
}
```

**影响**：调用方不需要修改，但每次 fireProgress 增加一次 RLock。

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| 构造函数改为接受 progressFn，破坏现有所有调用方 | 中 | 采用渐进方案（8.2），后续再统一转为构造函数注入 |
| RWMutex 增加的锁开销导致性能下降 | 低 | 非热点路径，每次安装→每个包触发一次，开销可忽略 |
| 调用方在 Install 期间调用 SetProgressHook 时，新的 progressFn 可能看不到已经完成的结果 | 低 | 文档约定：应只在 Install 前设置，但在 race-free 角度这是安全的 |

---

## P3 — 僵尸代码（logger / --verbose）

### 问题

- **`internal/log/logger.go`**（105 行）：完整的多 writer Logger + multiHandler，但 **全库没有一行 import 它**。所有输出通过 `fmt.Fprintf(os.Stderr, ...)` 直接写入。
- **`--verbose` 标志** (`root.go:32`)：在 root 命令注册了，但没有任何子命令读取或使用它。

这两个功能设计意图是好的（结构化日志、分级输出），但从未集成，形成死代码。

### 修复方案

#### 9.1 集成 Logger（推荐）

将 Logger 注入到 Engine 和 Orchestrator 中：

```go
// engine.go
type Engine struct {
    // ...
    log *log.Logger  // 新增
}

func (e *Engine) Install(ctx context.Context, manifest *Manifest) (*Summary, error) {
    if e.log != nil {
        e.log.Info("starting installation", "packages", len(manifest.Packages))
    }
    // ...
}
```

ClI 入口：

```go
// cli/install.go:55
logger, err := log.New(cfg.LogLevel, cfg.LogFile)
if err != nil {
    return fmt.Errorf("init logger: %w", err)
}
defer logger.Close()
eng := engine.New(winget, &cfg.EngineConfig, engine.WithLogger(logger))
```

#### 9.2 接入 --verbose 标志

```go
// cli/root.go:19
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    // 读取 verbose 标志，注入到 context
    verbose, _ := cmd.Flags().GetBool("verbose")
    if verbose {
        // 设置 log level 为 debug
    }
    return nil
},
```

实际由 `PersistentPreRunE` 在命令执行前统一读取 flag 并设置 log level。

#### 9.3 日志替换原则

`fmt.Fprintf(os.Stderr, ...)` 不全部替换为 `log.Info()`。区分两类输出：

- **用户可读的输出**（进度、结果汇总、使用说明）→ 保留 `fmt.Fprintf(os.Stderr, ...)`（或者使用专用的 `output` 对象）
- **诊断/调试信息**（后端命令耗时、配置加载详情、预检细节）→ 使用 Logger

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| 集成 Logger 导致大量 diff，影响 code review | 中 | 分步合并：(1) 接入 Logger 但不替换任何 fmt.Fprintf；(2) 逐步替换诊断输出。两篇独立 PR |
| 同时使用 Logger + fmt 导致输出混乱（顺序错乱） | 低 | Logger 输出到文件或 stderr；用户可见的输出到 stderr 但格式保持一致；不同时混用在同一函数 |
| --verbose 从未被使用，删除比集成更简单 | 取决于方向 | 如果团队认为 Logger 不是当前重点，删除 --verbose 标志和 logger.go 也是合理选择。YAGNI 原则同样适用 |

**建议**：保留 Logger 代码但先不集成。删除 `--verbose` 标志（已注册却无用会误导用户）。待确定了完整的输出策略后再做整合。

---

## P3 — 串行安装性能

### 问题

Engine 按顺序逐个安装包：

```go
for _, pkg := range packages {
    // 完全串行
    err := e.installWithRetry(ctx, pkg.ID)
}
```

对于包含 30-50 个包的 manifest，串行安装意味着总时间 = 所有包安装时间之和。

### 修复方案

#### 10.1 新增 --parallel 标志（非默认）

```go
// engine.go
type Config struct {
    // ...
    Parallel int  // 0 或 1 = 串行（默认）
}
```

#### 10.2 使用 errgroup 实现并发

```go
func (e *Engine) InstallParallel(ctx context.Context, manifest *Manifest, concurrency int) (*Summary, error) {
    g, ctx := errgroup.WithContext(ctx)
    sem := make(chan struct{}, concurrency)
    var mu sync.Mutex
    summary := &Summary{StartTime: time.Now()}

    for _, pkg := range manifest.Packages {
        pkg := pkg
        g.Go(func() error {
            select {
            case sem <- struct{}{}:
            case <-ctx.Done():
                return ctx.Err()
            }
            defer func() { <-sem }()

            result := e.installOne(ctx, pkg)
            mu.Lock()
            summary.Results = append(summary.Results, result)
            // 更新 Succeeded/Skipped/Failed 计数
            mu.Unlock()
            e.fireProgress(result)
            return nil
        })
    }

    err := g.Wait()
    // finalize summary
    return summary, err
}
```

#### 10.3 安全考虑

- 默认串行（`--parallel 1`），避免用户意外触发并发
- 并发数上限：推荐 `--parallel 3`，防止 winget 自带的安装服务过度负载
- progress hook 需做 goroutine-safe 处理（参考 #8）

### 风险评估与规避

| 风险 | 概率 | 规避 |
|------|------|------|
| 并行 winget 安装争抢写锁（AppX 包）导致随机失败 | 中 | 默认串行；文档提示并行仅适用于独立包；并行度上限设为 3 而非无限制 |
| 进度输出交错混乱 | 高 | 使用 `fireProgress` 串行化的回调处理；不要在 goroutine 中直接 `fmt.Fprintf` |
| errgroup 中一个包失败导致整个组取消 | 中 | 使用 `g.SetLimit(concurrency)` + 自定义错误收集（不返回错误给 errgroup），使一个失败不影响其他包 |
| 并发 + 重试逻辑交互复杂 | 中 | 保持 `installOne` 内部的串行重试逻辑不变，并发只在包级别进行 |

---

## 总结优先级

| 优先级 | 修复编号 | 描述 | 影响范围 | 预估工作量 |
|--------|---------|------|---------|-----------|
| **P0** | #1 | 添加测试覆盖 | engine, config, backend, manifest | 大 |
| **P0** | #2 | Config 静默吞错 | config | 小 |
| **P1** | #3 | CLI God Handler | cli, engine (新文件) | 中 |
| **P1** | #4 | exec 无超时 | mirror, proxy, backend | 中 |
| **P1** | #5 | DRY 重复 | config, engine, cli | 中 |
| **P2** | #6 | DIP 违背 | cli → backend 工厂 | 小 |
| **P2** | #7 | OCP 违背 | mirror, config | 小 |
| **P2** | #8 | Winget 解析脆弱 | backend/winget | 中 |
| **P3** | #9 | 并发安全 | engine | 小 |
| **P3** | #10 | 僵尸代码 | log, cli/root | 小 |
| **P3** | #11 | 串行安装 | engine | 中 |
