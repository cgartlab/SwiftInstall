# LESSONS — SwiftInstall

> 跨平台批量软件安装器：Go CLI `sis`（Cobra，flag-driven，无配置文件）逐步取代遗留 winget/brew 脚本；Windows 优先，仅 winget backend，CI 发布 Windows amd64 binary。
> 本仓库从真实失误沉淀的「教训→规则」清单。**会话开始先全文阅读并遵循其中所有规则**；出现新失误按 §维护 追加。
> 权威依据：Anthropic Claude Code memory/best practices（<200 行、具体可验证、第二次犯同样错即写入）；社区 lessons.md 惯例（Boris Cherny，Anthropic）。
> 来源：本仓库 AGENTS.md 的 Gotchas + 已验证 fix 提交；预填为种子，后续由 Agent 在真实工作中增长。

## 维护
- **触发**：用户纠正 / Agent 自查到的失误 / 非显而易见的 gotcha。
- **格式**：`### <主题>` 下，每条 = 失误（具体场景）→ 规则（最小可执行指令），一行可验证。
- **只记项目特有、不可从代码推断的**；通用最佳实践不写；含糊措辞（"小心""注意"）删掉。
- **重复出现** → 升级进 `AGENTS.md` 硬规则；**月度审阅**裁剪过期项；目标 < 200 行。

## 清单解析

- 失误：按扩展名判定 manifest 格式（`.txt` 清单其实是 YAML）→ 规则：格式由内容决定，`manifest.ParseManifest` 嗅探 `packages:`/`settings:` 键；不要"修正"改名而不同步遗留脚本期望（见 AGENTS.md「Gotchas」）
- 失误：遗留安装器只解析 TXT，读不到 YAML 清单 → 规则：遗留脚本必须保持格式无关，同时接受 `- id:` 行与纯 TXT 列表（commit d401bc7；见 AGENTS.md「Gotchas」）
- 失误：重复 id 被当成两个独立包 → 规则：解析时折叠重复 ID；改解析器必须过根目录 `test_manifest_*.yaml` 回归样本（commit 0550216）

## 架构边界

- 失误：重新引入 config/mirror/proxy/log 子命令 → 规则：已作为死重删除，`sis` 是纯 flag-driven，无配置文件无持久化；无具体需求不要复活（见 AGENTS.md「Architecture」）
- 失误：把 `bin/` 预编译二进制当源码保留 → 规则：`bin/` 已删除（无源可溯），`.gitignore` 覆盖 `*.exe`；不要重新提交二进制（见 AGENTS.md「Gotchas」）
- 失误：DESIGN_ISSUES.md / TEST_REPORT.md 声称有 preflight/admin 检测 → 规则：实际不存在，唯一环境检查是 `backend.Detect()` 找 winget；两份文档视为历史（见 AGENTS.md「Gotchas」）
- 失误：给 sis 加配置文件 → 规则：flags 优先于 manifest 设置，优先于内置默认值（`cmd.Flags().Changed`），没有 config 文件（见 AGENTS.md「Config precedence」）
- 失误：list/status 忽略 `--no-color` 与 `--format silent` → 规则：所有命令都要尊重输出开关（commit ca07283）

## CI 与发布

- 失误：CI runner 名 / argus review label 写错 → 规则：核对 GitHub Actions 允许的 runner 名与 composite action label（commit 2887604、6425663）
- 失误：install.ps1 指向旧仓库名 → 规则：改名后同步 install 脚本的仓库 URL（commit 750126a）
- 失误：README / AGENTS.md / CLI_USAGE 与实际 CLI 脱节 → 规则：文档必须与真实 CLI 命令同步（commit cab23d7）
- 失误：CI 里 git 身份未配 + CRLF 损坏 → 规则：Actions 显式配 git identity 并处理行尾（commit 29683cf）
- 失误：构建产物与死配置文件入库 → 规则：清理 tracked 构建产物与死配置（commit d95abf3）
- 失误：假设 push 触发 CI 已覆盖 vet/测试 → 规则：当前 CI 只跑 build + version/help smoke，`go vet` 与测试尚未接入，不要假设已覆盖（见 AGENTS.md「CI/CD」）
- 失误：跳过本地校验 → 规则：改 Go 代码后 `go build -o sis.exe ./cmd/sis/` + `go vet ./...` 双跑（见 AGENTS.md「Build」）
