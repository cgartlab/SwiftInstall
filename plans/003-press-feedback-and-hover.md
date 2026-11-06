# 003 — 可点元素加按压反馈；hover 位移仅在精细指针下生效

- **Status**: DONE (round 40)
- **Commit**: 14ba7d7
- **Severity**: MEDIUM
- **Category**: Physicality / Accessibility
- **Estimated scope**: styles.css + audit/toollists/swap, ~8 lines

## Problem

全站 `.btn` 等可点元素**没有 `:active` 按压反馈**（grep 全库 0 个 `:active`），点下去只有 hover 的位移；且 hover 位移未用 `@media (hover:hover)` 门控，触屏点按会触发"假 hover"。

```css
/* internal/gui/assets/styles.css:242 — current */
.btn:hover { border-color: var(--moss); }
```

## Target

```css
@media (hover: hover) and (pointer: fine) {
  .btn:hover { transform: translateY(-1px); }
}
.btn:active { transform: scale(0.97); }
.btn { transition: transform 160ms cubic-bezier(0.23,1,0.32,1), border-color 160ms, background 160ms; }
```

## Repo conventions to follow

- 主 GUI 的按钮在 `internal/gui/assets/styles.css:235-247`（`.btn`/`.btn.primary`/`.btn.danger`）；页面内按钮（audit/toollists/swap 的 `.btn`）各自有本地定义，逐一补 `:active` 与 hover 门控。

## Steps

1. `styles.css` 的 `.btn` 增加 `:active{transform:scale(0.97)}` 与 `transition: transform 160ms cubic-bezier(0.23,1,0.32,1), border-color .16s, background .16s`；hover 位移包进 `@media (hover:hover) and (pointer:fine)`。
2. `audit.html` / `toollists.html` / `swap.html` 的 `.btn` 同样补 `:active`（这些页已有 hover 位移的，包进门控）。

## Boundaries

- 只加 `:active` 与 hover 门控，不改布局/配色。
- 不改禁用态（`:disabled` 保持 opacity 处理）。

## Verification

- **Feel check**：按下按钮有轻微"被按下去"的 0.97 缩放（160ms）；松手回弹；触屏设备（或 DevTools 设备模拟）不出现 hover 位移卡住。
- **Done when**: 所有主要按钮按压有反馈，触屏无假 hover。
