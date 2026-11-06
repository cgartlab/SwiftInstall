# Animation improvement plans

来源：`improve-animations` 审计（commit 14ba7d7）。执行者零上下文——每份计划自带精确取值与验收。

| # | 标题 | 严重度 | 状态 |
|---|---|---|---|
| 001 | 滑卡：按速度判定 + 飞出与换牌同步 | HIGH | DONE |
| 002 | reduced-motion 只去位移、保留颜色/透明度 | MEDIUM | DONE |
| 003 | 按压反馈 + hover 门控 | MEDIUM | DONE |
| 004 | 进度条改用 transform: scaleX | MEDIUM | DONE |

**建议顺序**：001 → 003 → 002 → 004。
- 001 是招牌交互手感，最高杠杆，独立。
- 003 与 002 都触及 hover/反馈，可相邻做；002 依赖 003 的按钮结构无关，仅需同一批 CSS。
- 004 独立、低风险。

**未列入本批（LOW）**：audit/toollists 里硬编码时长（.2s/.35s/.3s/.4s）与 swap 的 `--dur-*` token 不统一——待 002/003 落地后再做 token 收敛。
