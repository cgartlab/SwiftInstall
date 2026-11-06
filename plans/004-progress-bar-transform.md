# 004 — 扫描进度条改用 transform: scaleX（避免动画 width 触发布局）

- **Status**: DONE (round 40)
- **Commit**: 14ba7d7
- **Severity**: MEDIUM
- **Category**: Performance
- **Estimated scope**: 1 file, ~4 lines

## Problem

体检页扫描进度条动画 `width`（布局属性，触发布局+绘制，非合成层）：

```css
/* internal/gui/assets/audit.html:56 — current */
.track i{display:block;height:100%;width:0;background:var(--green);transition:width .35s ease}
```
```js
/* audit.html:204 — current（每帧写 width） */
bar.style.width = (e*100)+"%";
```

## Target

用 `transform: scaleX` 驱动，`transform-origin:left`，`will-change: transform`；JS 改 `bar.style.transform = "scaleX("+e+")"`。

```css
.track i{display:block;height:100%;width:100%;transform:scaleX(0);transform-origin:left;background:var(--green);transition:transform .35s cubic-bezier(0.23,1,0.32,1);will-change:transform}
```

## Repo conventions to follow

- `audit.html` 内联 CSS（:56 与 :204）；沿用本页既有 `--green` 与 350ms 观感。

## Steps

1. 改 `.track i` 样式为 `scaleX` 版本（见上）。
2. 把 `bar.style.width=(e*100)+"%"` 改为 `bar.style.transform="scaleX("+e+")"`（e 已是 0..1）。

## Boundaries

- 只改 `audit.html` 的进度条；不动扫描其余逻辑。

## Verification

- **Mechanical**: `node --check` 通过；页面加载显示进度仍从 0 涨到满。
- **Performance**：DevTools Performance 录制扫描过程，`Rendering → Paint flashing` 不应每帧重绘整条轨道；无 Layout 每帧。
- **Done when**: 进度条视觉一致、无每帧 Layout。
