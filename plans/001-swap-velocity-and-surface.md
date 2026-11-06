# 001 — 滑卡：按速度判定 + 飞出与换牌同步（招牌交互手感）

- **Status**: DONE (round 40)
- **Commit**: 14ba7d7
- **Severity**: HIGH
- **Category**: Interruptibility / Physicality
- **Estimated scope**: 1 file, ~25 lines

## Problem

`/swap` 的平替滑卡（招牌交互）有两处手感问题，都在 `internal/gui/assets/swap.html`：

1. 松手判定只看**固定距离**，不看速度（快甩反而可能不过阈值）：
```js
/* internal/gui/assets/swap.html:175 — current */
if(dx>110) go(el,"swap"); else if(dx<-110) go(el,"keep");
```
2. 飞出动画时长是 `var(--dur-3)`=320ms，但换牌 `render()` 在 **190ms** 就执行，把正在飞出的卡片直接清掉重绘 → 飞出被截断、下一张"啪"地出现，没有承接：
```js
/* internal/gui/assets/swap.html:183 — current */
deck.shift(); setTimeout(render,190);
```
```css
/* internal/gui/assets/swap.html:37 — current */
.card.leaving{transition:transform var(--dur-3) var(--ease),opacity var(--dur-3) var(--ease)}
```

## Target

- 松手用**速度**判定：`|dx| / elapsedMs > 0.11` 即判方向（距离仍作为兜底 `>110`）。
- 飞出的过渡时长与换牌时机对齐：`--dur-3`=320ms，则 `render()` 在 **320ms** 后再执行（或监听 `transitionend`）。
- 新卡出现时有**轻微弹簧感**：`c0` 从 `translateY(10px) scale(0.98)` 过渡到原位，`transform 260ms cubic-bezier(0.23,1,0.32,1)`（强 ease-out）。

## Repo conventions to follow

- 本页已定义运动 token：`--dur-2:200ms; --dur-3:320ms; --ease:cubic-bezier(.22,.61,.36,1)`（`swap.html:13`）。新增强 ease-out 用 `--ease-out: cubic-bezier(0.23,1,0.32,1)`，与既有风格一致。
- 参考 `swap.html:61` 的按钮 `transition:transform var(--dur-2) var(--ease),...` 的写法。

## Steps

1. 在 `#150` 附近记录按下时间：`pointerdown` 时 `sx=e.clientX; st=performance.now();`。
2. 改 `pointerup` 判定：
   ```js
   const dt = Math.max(1, performance.now()-st);
   const v = Math.abs(dx)/dt; // px/ms
   if(dx>110 || (v>0.11 && dx>0)) go(el,"swap");
   else if(dx<-110 || (v>0.11 && dx<0)) go(el,"keep");
   else { el.style.transform=""; }
   ```
3. 把 `setTimeout(render,190)` 改为 `setTimeout(render,320)`（与 `--dur-3` 对齐）。
4. 在 CSS 里给新卡入场：`.card.c0{transition:transform 260ms cubic-bezier(0.23,1,0.32,1)}` 并在 `render()` 里先把 `c0` 设为 `translateY(10px) scale(.98)`，下一帧再置空（用 `requestAnimationFrame`）。

## Boundaries

- 只改 `internal/gui/assets/swap.html`；不要动 `/api` 或其它页面。
- 不改卡片 HTML 结构，只改事件与 CSS 过渡。
- 不引入依赖。

## Verification

- **Mechanical**: `node --check` 提取的脚本应通过；`go build ./... && go test ./...` 仍绿。
- **Feel check**（DevTools Animations 面板 10% 慢放）：
  - 快甩一下 → 即使没到 110px 也应判定飞出；
  - 飞出完整 320ms 后下一张才出现，衔接自然、不"啪"；
  - 连甩两张不卡顿、不回弹错方向；
  - 打开 `prefers-reduced-motion` → 位移消失但换牌逻辑仍正常（配合计划 002）。
- **Done when**: 上述四条肉眼可验。
