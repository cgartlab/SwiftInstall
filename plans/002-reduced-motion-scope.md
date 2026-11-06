# 002 — reduced-motion 只去位移，保留颜色/透明度反馈

- **Status**: DONE (round 40)
- **Commit**: 14ba7d7
- **Severity**: MEDIUM
- **Category**: Accessibility
- **Estimated scope**: 2+1 files, ~10 lines

## Problem

多处用 `*{transition:none!important}` 一刀切，把颜色/透明度等"帮助理解"的反馈也砍了（违反 reduced-motion 应"少而稳"而非"全无"）：

```css
/* internal/gui/assets/swap.html:72 — current */
@media (prefers-reduced-motion:reduce){*{transition:none!important}.card{box-shadow:none}}
```
```css
/* internal/gui/assets/toollists.html:116 — current */
@media (prefers-reduced-motion:reduce){*{transition:none!important}}
```
```css
/* internal/gui/assets/styles.css:369 — current */
@media (prefers-reduced-motion: reduce) { * { animation: none !important; transition: none !important; } }
```

## Target

在 reduced-motion 下：**去掉位移/缩放，保留 opacity 与颜色过渡**。示例：

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: .01ms !important; animation-iteration-count: 1 !important; }
  .card, .inner, .rail-item, .btn, .gdrawer, .cmdk { transition-property: opacity, background-color, border-color, color !important; }
  /* 位移类的 transform 过渡在 reduced-motion 下不生效 */
}
```
（`styles.css` 同理：不要 `transition:none`，改为把 transform/animation 关掉，保留 opacity/color。）

## Repo conventions to follow

- 各页已有 `@media (prefers-reduced-motion:reduce)` 块（swap:72、toollists:116、audit:144、styles.css:369），就地改写，不新增位置。

## Steps

1. 改 `swap.html:72` 的块：去掉 `*{transition:none!important}`，改为上面的"只保留 opacity/color"规则。
2. 改 `toollists.html:116` 同理。
3. 改 `styles.css:369`：把 `transition:none` 换成只禁用 transform 位移与 animation，保留 opacity/color。

## Boundaries

- 只动这三个 `@media (prefers-reduced-motion:reduce)` 块；不动正常态样式。
- 不改动效值。

## Verification

- **Feel check**：Rendering 面板开启 `prefers-reduced-motion` → hover 仍能看到颜色/透明度变化，但不再有位移/缩放；页面切换无滑动。
- **Done when**: reduced-motion 下"无位移、有反馈"。
