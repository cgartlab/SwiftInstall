// Shared ⌘K command palette for every SwiftInstall page.
// Self-injects its styles and overlay; binds ⌘K / Ctrl-K. Generic navigation
// commands work on any page, so the keyboard-first experience is consistent.
(function () {
  if (window.__cmdkInit) return;
  window.__cmdkInit = true;

  var css = `
  .cmdk{position:fixed;inset:0;z-index:2000;display:grid;place-items:start center;padding-top:14vh;
    background:color-mix(in srgb,#0b0b0a 42%,transparent);backdrop-filter:blur(6px)}
  .cmdk[hidden]{display:none}
  .cmdk-box{width:min(560px,92vw);background:var(--card,var(--panel,#FBF8F1));
    border:1px solid var(--line,#DDD6C7);border-radius:14px;box-shadow:inset 0 1px 0 rgba(255,255,255,.55),0 24px 70px rgba(0,0,0,.35);overflow:hidden}
  .cmdk-input{width:100%;border:0;border-bottom:1px solid var(--line,#DDD6C7);background:transparent;
    color:var(--ink,#1C1B17);font-size:15px;padding:15px 18px;outline:0;
    font-family:-apple-system,BlinkMacSystemFont,"PingFang SC",sans-serif}
  .cmdk-list{list-style:none;max-height:46vh;overflow-y:auto;padding:6px}
  .cmdk-item{display:flex;justify-content:space-between;align-items:center;gap:12px;padding:10px 12px;
    border-radius:9px;cursor:pointer;color:var(--ink,#1C1B17)}
  .cmdk-item.on{background:var(--green-soft,#E7EEE8)}
  .cmdk-item .ct{font-size:13.5px;font-weight:600}
  .cmdk-item .cd{font-size:12px;color:var(--muted,#6E6B60);margin-top:1px}
  .cmdk-item .ck{font-size:11px;color:var(--faint,#9A9384)}
  .cmdk-empty{padding:16px 12px;color:var(--faint,#9A9384);font-size:13px}
  .cmdk-grp{list-style:none;font-size:11px;letter-spacing:.04em;color:var(--faint,#9A9384);padding:9px 12px 3px}
  @media (prefers-reduced-transparency:reduce){.cmdk{background:rgba(11,11,10,.92);backdrop-filter:none}}
  .cmdk-foot{font-size:11px;color:var(--faint,#9A9384);padding:9px 16px;
    border-top:1px solid var(--line,#DDD6C7);text-align:right}
  `;
  var style = document.createElement("style");
  style.textContent = css;
  document.head.appendChild(style);

  var ov = document.createElement("div");
  ov.className = "cmdk";
  ov.hidden = true;
  ov.innerHTML =
    '<div class="cmdk-box" role="dialog" aria-modal="true" aria-label="命令面板">' +
    '<input id="cmdkInput" class="cmdk-input" type="text" placeholder="搜索命令…（体检 / 清单 / 逐条决策 / 返回）" autocomplete="off" spellcheck="false">' +
    '<ul id="cmdkList" class="cmdk-list"></ul>' +
    '<div class="cmdk-foot">↑↓ 选择 · ↵ 执行 · Esc 关闭</div></div>';
  document.body.appendChild(ov);

  var CMDS = [
    { t: "环境体检", d: "扫描已装软件，看该装 / 该换什么", run: function () { location.href = "/audit"; } },
    { t: "软件清单", d: "浏览精选免费 / 开源清单", run: function () { location.href = "/toollists"; } },
    { t: "逐条决策 · 设计师清单", d: "用平替滑卡逐条决定", run: function () { location.href = "/swap?toollist=mac-designers"; } },
    { t: "逐条决策 · 开发者清单", d: "用平替滑卡逐条决定", run: function () { location.href = "/swap?toollist=developers"; } },
    { t: "逐条决策 · 创作者清单", d: "用平替滑卡逐条决定", run: function () { location.href = "/swap?toollist=creators"; } },
    { t: "系统自检", d: "看包管理器 / 清单新鲜度 / AI 配置", run: function () { location.href = "/health"; } },
    { t: "回到主界面", d: "返回清单管理", run: function () { location.href = "/"; } },
    { t: "回到顶部", d: "滚动到页面顶部", run: function () { close(); window.scrollTo({ top: 0, behavior: "smooth" }); } },
    { t: "刷新", d: "重新加载页面", run: function () { location.reload(); } }
  ];

  var inp = ov.querySelector("#cmdkInput");
  var list = ov.querySelector("#cmdkList");
  var recent = []; // in-memory only (no persistence), most-recently-run first
  var shown = CMDS.slice();
  var sel = 0;

  function ordered() {
    var r = [];
    recent.forEach(function (c) { if (CMDS.indexOf(c) >= 0 && r.indexOf(c) < 0) r.push(c); });
    CMDS.forEach(function (c) { if (r.indexOf(c) < 0) r.push(c); });
    return r;
  }
  function open() { ov.hidden = false; inp.value = ""; filter(""); inp.focus(); }
  function close() { ov.hidden = true; }
  function fscore(hay, q) {
    var h = (hay || "").toLowerCase();
    var i = h.indexOf(q);
    if (i >= 0) return 1000 - i; // substring: strongest, earlier is better
    var hi = 0, s = 0;
    for (var k = 0; k < q.length; k++) {
      var p = h.indexOf(q[k], hi);
      if (p < 0) return -1; // not a subsequence
      s += (p === hi ? 2 : 1); // reward contiguity
      hi = p + 1;
    }
    return s; // weaker but valid
  }
  function filter(q) {
    q = (q || "").trim().toLowerCase();
    if (!q) { shown = ordered(); sel = 0; paint(); return; }
    var scored = [];
    for (var i = 0; i < CMDS.length; i++) {
      var c = CMDS[i], st = fscore(c.t, q), sd = fscore(c.d, q);
      var sc = st >= 0 ? st : (sd >= 0 ? sd - 300 : -1);
      if (sc >= 0) scored.push({ c: c, s: sc });
    }
    scored.sort(function (a, b) { return b.s - a.s; });
    shown = scored.map(function (x) { return x.c; });
    sel = 0; paint();
  }
  function groupOf(c) { return (c.t === "回到顶部" || c.t === "刷新") ? "动作" : "导航"; }
  function paint() {
    if (!shown.length) { list.innerHTML = '<li class="cmdk-empty">没有匹配的命令</li>'; return; }
    var html = "", lastG = null;
    shown.forEach(function (c, i) {
      var g = groupOf(c);
      if (g !== lastG) { html += '<li class="cmdk-grp">' + g + '</li>'; lastG = g; }
      html += '<li class="cmdk-item ' + (i === sel ? "on" : "") + '" data-i="' + i + '">' +
        '<div><div class="ct">' + c.t + '</div><div class="cd">' + c.d + '</div></div>' +
        '<span class="ck">↵</span></li>';
    });
    list.innerHTML = html;
    Array.prototype.forEach.call(list.querySelectorAll(".cmdk-item"), function (li) {
      li.onclick = function () { sel = +li.dataset.i; exec(); };
    });
  }
  function exec() { var c = shown[sel]; if (c) { recent = [c].concat(recent.filter(function (x) { return x !== c; })).slice(0, 4); close(); c.run(); } }

  inp.addEventListener("input", function (e) { filter(e.target.value); });
  inp.addEventListener("keydown", function (e) {
    if (e.key === "ArrowDown") { e.preventDefault(); sel = Math.min(shown.length - 1, sel + 1); paint(); }
    else if (e.key === "ArrowUp") { e.preventDefault(); sel = Math.max(0, sel - 1); paint(); }
    else if (e.key === "Enter") { e.preventDefault(); exec(); }
    else if (e.key === "Escape") { e.preventDefault(); close(); }
  });
  ov.addEventListener("click", function (e) { if (e.target === ov) close(); });
  document.addEventListener("keydown", function (e) {
    if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) {
      e.preventDefault(); ov.hidden ? open() : close();
    } else if (e.key === "Escape" && !ov.hidden) { close(); }
  });
  window.__cmdkOpen = open;
})();
