# SwiftInstall 设计系统（Design System）

> 目标：把散落在各页面的样式收敛成**一套可执行的 token + 组件规范**，保证一致、可维护、可访问，并守住"《Cabinet of Equivalents》"的品牌调性。
> 交互与动效策略见 [`INTERACTION-DESIGN.md`](INTERACTION-DESIGN.md)；定位与差异见 [`POSITIONING.md`](POSITIONING.md)。

## 一、设计原则（一句话版）

1. **纸与墨的编辑气质**：骨纸底、铁墨字、氧化绿/焦琥珀做唯一强调色；画面像一本标本图鉴，不是 SaaS 卡片墙。
2. **结构即信息**：分类、发丝线、标签、版本日期都是"有意义的结构"，不是装饰。
3. **动效分级**：高频处近乎无感（120–180ms），低频仪式才允许编排（520ms）；永远有 `reduced-motion` 降级。
4. **克制与聚焦**：一屏只有一个记忆点（如书单页的左侧索引、滑卡的卡堆），其余安静。

## 二、设计 Token

### 颜色（CSS 变量，亮/暗两套）

> 暗色实现采用 `@media (prefers-color-scheme: dark)`（跟随系统），而非 `[data-theme="dark"]`。审查/Argus 报告若出现「dark 未覆盖」告警，属误报。
| Token | 亮色 | 暗色 | 用途 |
|---|---|---|---|
| `--paper` | `#F4EFE6` | `#16140F` | 页面底色（骨纸） |
| `--panel` / `--card` | `#FBF8F1` | `#1D1A13` | 面板 / 卡片面 |
| `--panel-2` | `#EFEADE` | `#242016` | 次级面 / 选中底 |
| `--ink` | `#1C1B17` | `#ECE7DC` | 正文墨色 |
| `--muted` | `#6E6B60` | `#A39B89` | 次要文字 |
| `--faint` | `#9A9384` | `#6F6858` | 标签 / 最弱信息 |
| `--line` / `--line-2` | `#DDD6C7` / `#E8E2D5` | `#2E2A20` / `#252118` | 发丝线 / 更淡分隔 |
| `--moss`（=氧化绿） | `#2F5D46` | `#8FB89B` | 主强调 / 收益 / 开源 |
| `--moss-ink` | `#234636` | `#A7C9B1` | 强调悬停 |
| `--moss-soft` | `#E7EEE8` | `#1E2A22` | 强调浅底 / 选中 |
| `--copper`（=焦琥珀） | `#A9762A` | `#D6A45E` | 次强调 / 代价 / 付费 |
| `--copper-soft` | `#F4EAD8` | `#3A2C18` | 琥珀浅底 |
| `--red` | `#A24B3B` | `#D97757` | 危险 / 错误 |
| `--tile` | `#FFFFFF` | `#232016` | 图标底块 |

**用色规则**：绿=收益/开源/免费；琥珀=代价/付费/需复核；红=危险。颜色只做**辅助编码**，文字必须能独立表意（不靠颜色单独传达）。

### 字体
- `--serif`：`"Iowan Old Style","New York","Palatino","Songti SC","Noto Serif SC",Georgia,serif` —— 标题、工具名、书单名。
- `--sans`：`-apple-system,BlinkMacSystemFont,"PingFang SC","Noto Sans SC",sans-serif` —— 正文、按钮、标签。
- `--mono`：`ui-monospace,"SF Mono",Menlo,monospace` —— 代码、命令行、编号。

### 间距与圆角
- 间距节奏：4 / 8 / 12 / 16 / 20 / 24 / 32（px）。
- 圆角：输入/按钮 9–11；卡片 14–18；胶囊 20 / 999。
- 阴影克制：内容层仅 1px 暖色描边 + 极淡暖阴影；只有"正在决策/浮层"抬升一层。

### 运动
| Token | 值 | 用途 |
|---|---|---|
| `--dur-1` | 120ms | hover / 焦点 |
| `--dur-2` | 200ms | 常规过渡 |
| `--dur-3` | 320ms | 抽屉 / 模态 |
| `--dur-4` | 520ms | 首次仪式（仅一次） |
| `--ease` | `cubic-bezier(.22,.61,.36,1)` | 主力缓动（ease-out-expo 近似） |

拖拽交互用弹簧（跟随手势速度）；**高频展开禁用带 overshoot 的弹簧**（连按会"卡在反弹里"）。所有编排在 `prefers-reduced-motion:reduce` 下降级为直切/交叉淡入。

## 三、组件规格

### 按钮 `.btn`
- 变体：`.btn`（默认）/`.btn.primary`（绿实心，唯一主动作）/`.btn.ghost`（轻边界）/`.btn.danger`（红）。
- 高度 ≥ 36px，触控目标 ≥ 44px；hover 位移 -1px 或描边变绿；`:focus-visible` 2px 绿环。

### 标签 `.chip`
- `oss`/`free`=绿浅底；`freemium`=琥珀浅底；`warn`（需复核）=红浅底。用于授权/状态，**不做装饰**。

### 账本行 `.row`（列表主体）
- 发丝线分隔、不套圆角卡片；左图标 tile + 名称 + 角色 + 授权 chip + 状态点；行高一致、数字 `tabular-nums`。

### 卡片 `.card` / `.item`
- 仅在"需要成块包裹"时使用（滑卡、对比、抽屉面板）；内容层不要满屏圆角卡 + 统一灰影（AI 生成感）。

### 抽屉 `.drawer` / `.gdrawer`
- 右滑入，320ms；遮罩 `rgba(28,27,23,.30)`；Esc/点遮罩关闭；内部标题用 `--serif`。

### 命令面板 `.cmdk`（⌘K）
- 自注入（`cmdk.js`）；输入过滤 + `↑↓` 选择 + `↵` 执行 + `Esc` 关闭；面板宽 ≤560px，半透明遮罩，`role=dialog`。

### 平替滑卡 `.card.c0/.c1/.c2`
- 三张层叠；顶层可拖拽（跟手 + 轻旋转），松手按位移/速度判定左右；翻面用 `rotateY(180deg)` 展示"数据保真 + 第一步"。

### 对比 `.cols`（并排）
- 两列 + 中间发丝分隔；同一维度左右对齐；结论条用 `--moss-soft`；**文字维度用并排，滑块留截图对比**。

### 步进 `.steps`（迁移指导）
- 仅当内容是"真的步骤序列"才编号；每步可"复制该步命令"，只读不代办。

### 提示 `.toast`
- 底部居中、2s 自动消失；只做"已复制/已保存"这类确认。

## 四、无障碍与质量门槛

- 正文对比度 ≥ 4.5:1（在**最差背景**——纯白/高饱和壁纸上验）。
- 焦点可见（2px 绿环、offset 2）；键盘可达所有核心动作；拖拽必须有按钮等价。
- 触控目标 ≥ 44px；`prefers-reduced-motion` / `prefers-reduced-transparency` 均有降级（半透明面配实心兜底）。
- 只动 `transform/opacity/clip-path`；拖拽走 `requestAnimationFrame`，不阻塞主线程。

## 五、要避免的"AI 生成感"（Do / Don't）

- ✅ 克制的单一强调色 + 结构化的分类与日期；❌ 每个区块都 fade-up、满屏同款圆角卡 + 灰影。
- ✅ 文字 sentence case、主动语态、按钮动词一致（"保存"→"已保存"）；❌ 堆 ALL-CAPS 眉题、`A · B · C` 串、无意义 01/02/03 编号。
- ✅ 对比用并排、动效花在低频仪式；❌ 用滑块做纯文字对比、高频处加花哨动画。

## 六、代码映射

| 规范 | 落地文件 |
|---|---|
| 颜色/字体/组件基础 | `internal/gui/assets/styles.css`（主 GUI） |
| 体检页 | `internal/gui/assets/audit.html` |
| 清单浏览 / 过滤 | `internal/gui/assets/toollists.html` |
| 平替滑卡 `/swap` | `internal/gui/assets/swap.html` |
| 对比原型 | `docs/prototype/compare.html` |
| 命令面板 | `internal/gui/assets/cmdk.js` |
| 系统自检 `/health` | `internal/gui/assets/health.html` |
