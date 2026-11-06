"use strict";

const $ = (id) => document.getElementById(id);

const state = {
  packages: [],
  installed: {},
  selected: new Set(),
  query: "",
  statusLoaded: false,
  running: false,
};

/* Package id -> icon slug in the embedded /icons/*.svg set. Packages absent
   from this map (or whose icon fails to load) get a coloured letter tile. */
const ICONS = {
  git: "git",
  gh: "github",
  "python@3.12": "python",
  node: "nodedotjs",
  pnpm: "pnpm",
  go: "go",
  bat: "bat",
  htop: "htop",
  neovim: "neovim",
  tmux: "tmux",
  zsh: "zsh",
  starship: "starship",
  "visual-studio-code": "vscode",
  iterm2: "iterm2",
  figma: "figma",
  krita: "krita",
  blender: "blender",
  "unity-hub": "unity",
  telegram: "telegram",
  wechat: "wechat",
  notion: "notion",
  obsidian: "obsidian",
  raycast: "raycast",
  loop: "loop",
  steam: "steam",
  scrcpy: "android",
  "squirrel-app": "rime",
  mas: "appstore",
};

const MONO_HUES = [255, 210, 152, 12, 340, 280, 190, 96, 30];
function monoColor(id) {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0;
  return `hsl(${MONO_HUES[h % MONO_HUES.length]}, 52%, 48%)`;
}
function monoText(id) {
  if (id.startsWith("font-")) return "Aa";
  const c = id.replace(/[^a-z0-9]/gi, "");
  return (c[0] || "?").toUpperCase();
}

function makeIcon(p) {
  const icon = document.createElement("span");
  icon.className = "pkg-icon";
  const slug = ICONS[p.id];
  if (slug) {
    const img = document.createElement("img");
    img.src = `/icons/${slug}.svg`;
    img.alt = "";
    img.loading = "lazy";
    img.addEventListener("error", () => {
      icon.removeChild(img);
      icon.classList.add("mono");
      icon.textContent = monoText(p.id);
      icon.style.background = monoColor(p.id);
    });
    icon.appendChild(img);
  } else {
    icon.classList.add("mono");
    icon.textContent = monoText(p.id);
    icon.style.background = monoColor(p.id);
  }
  return icon;
}

/* ---------------- data ---------------- */

async function loadState() {
  try {
    const s = await fetch("/api/state").then((r) => r.json());
    state.packages = s.packages || [];
    $("backend").textContent = s.backend || "backend";
    $("source").textContent = s.source || "(未指定清单)";
    render();
    loadStatus();
  } catch (e) {
    $("list").innerHTML = `<div class="empty"><div>无法加载清单：${e}</div></div>`;
  }
}

async function loadStatus() {
  state.statusLoaded = false;
  updateStats();
  render();
  try {
    const r = await fetch("/api/status").then((r) => r.json());
    if (r.error) throw new Error(r.error);
    state.installed = r.installed || {};
  } catch (e) {
    console.error("status:", e);
  }
  state.statusLoaded = true;
  updateStats();
  render();
}

function statusOf(id) {
  if (!state.statusLoaded) return "unknown";
  return state.installed[id] ? "installed" : "missing";
}

/* ---------------- rendering ---------------- */

var kbIndex = -1;
var lastG = 0;
function visibleRows() { return Array.prototype.slice.call(document.querySelectorAll("#list .row")); }
function setKb(i) {
  var rows = visibleRows();
  if (!rows.length) { kbIndex = -1; return; }
  kbIndex = Math.max(0, Math.min(rows.length - 1, i));
  rows.forEach(function (r, j) { r.classList.toggle("kb", j === kbIndex); });
  rows[kbIndex].scrollIntoView({ block: "nearest" });
}

function render() {
  kbIndex = -1;
  const q = state.query.trim().toLowerCase();
  const filtered = state.packages.filter(
    (p) =>
      !q ||
      p.id.toLowerCase().includes(q) ||
      (p.category || "").toLowerCase().includes(q)
  );

  const listEl = $("list");
  listEl.innerHTML = "";

  const groups = new Map();
  for (const p of filtered) {
    const c = p.category || "未分类";
    if (!groups.has(c)) groups.set(c, []);
    groups.get(c).push(p);
  }

  for (const [cat, items] of groups) {
    const sec = document.createElement("section");
    sec.className = "section";
    const head = document.createElement("div");
    head.className = "section-head";
    const title = document.createElement("span");
    title.className = "section-title";
    title.textContent = cat;
    const count = document.createElement("span");
    count.className = "section-count";
    count.textContent = items.length + " 项";
    head.append(title, count);
    sec.appendChild(head);
    for (const p of items) sec.appendChild(renderRow(p));
    listEl.appendChild(sec);
  }

  const emptyEl = $("empty");
  if (filtered.length === 0) {
    if (state.packages.length === 0) {
      emptyEl.innerHTML = '<div>当前清单还没有软件</div>' +
        '<div class="empty-sub">去「软件清单」挑一批只收免费/开源的工具，或另存一份自己的清单。</div>' +
        '<a class="btn" href="/toollists" style="margin-top:14px;display:inline-flex">去软件清单 →</a>';
    } else {
      emptyEl.innerHTML = '<div>没有匹配的软件</div>' +
        '<div class="empty-sub">试试换个关键词，或清空过滤条件。</div>';
    }
    emptyEl.hidden = false;
  } else {
    emptyEl.hidden = true;
  }
  syncSelectAll(filtered);
  updateActions();
}

function renderRow(p) {
  const wrap = document.createElement("div");
  wrap.className = "row-wrap";

  const row = document.createElement("div");
  row.className = "row";
  row.dataset.id = p.id;

  const cb = document.createElement("input");
  cb.type = "checkbox";
  cb.checked = state.selected.has(p.id);
  cb.addEventListener("change", () => {
    if (cb.checked) state.selected.add(p.id);
    else state.selected.delete(p.id);
    updateStats();
    updateActions();
    syncSelectAllCurrent();
  });

  const main = document.createElement("div");
  main.className = "pkg-main";
  const idEl = document.createElement("span");
  idEl.className = "pkg-id";
  idEl.textContent = p.id;
  idEl.title = p.id;
  main.appendChild(idEl);
  if (p.optional) {
    const opt = document.createElement("span");
    opt.className = "pkg-opt";
    opt.textContent = "可选";
    main.appendChild(opt);
  }
  const chev = document.createElement("span");
  chev.className = "chev";
  chev.textContent = "›";
  main.appendChild(chev);

  const kind = document.createElement("span");
  // For non-brew sources, show the source (npm/github) instead of the
  // Homebrew channel; brew packages show formula/cask as before.
  if (p.source === "npm" || p.source === "github") {
    kind.className = "kind src-" + p.source;
    kind.textContent = p.source;
  } else {
    kind.className = "kind" + (p.kind === "cask" ? " cask" : "");
    kind.textContent = p.kind === "cask" ? "应用" : p.kind === "formula" ? "命令行" : "自动";
  }

  const st = statusOf(p.id);
  const status = document.createElement("span");
  status.className = "status " + st;
  status.innerHTML = `<span class="led"></span>`;
  status.appendChild(
    document.createTextNode(
      st === "installed" ? "已安装" : st === "missing" ? "未安装" : "检测中"
    )
  );

  row.append(cb, makeIcon(p), main, kind, status);
  // Clicking anywhere on the row except the checkbox toggles details.
  row.addEventListener("click", (e) => {
    if (e.target === cb) return;
    toggleDetail(wrap, row, p);
  });

  wrap.appendChild(row);
  return wrap;
}

function toggleDetail(wrap, row, p) {
  const open = wrap.classList.contains("open");
  if (open) {
    wrap.classList.remove("open");
    wrap.querySelector(".detail")?.remove();
    return;
  }
  const detail = document.createElement("div");
  detail.className = "detail loading";
  detail.textContent = "加载中…";
  wrap.appendChild(detail);
  wrap.classList.add("open");
  loadDetail(wrap, detail, p);
}

async function loadDetail(wrap, detail, p) {
  try {
    const r = await fetch(`/api/info?name=${encodeURIComponent(p.id)}`).then((r) => r.json());
    if (r.error) { detail.className = "detail"; detail.textContent = "无法获取详情：" + r.error; return; }
    detail.className = "detail";
    const meta = document.createElement("div");
    meta.className = "detail-meta";
    meta.innerHTML =
      `<span class="dm"><span class="dm-l">版本</span><span class="dm-v">${r.version || "—"}</span></span>` +
      `<span class="dm"><span class="dm-l">最近更新</span><span class="dm-v">${r.updated || "—"}</span></span>` +
      `<span class="dm"><span class="dm-l">类型</span><span class="dm-v">${r.kind || "—"}</span></span>`;
    const desc = document.createElement("div");
    desc.className = "detail-desc";
    desc.textContent = r.desc || "（无简介）";
    if (r.ai_desc) {
      const badge = document.createElement("span");
      badge.className = "ai-badge";
      badge.textContent = "AI 补全";
      desc.appendChild(badge);
    }
    const links = document.createElement("div");
    links.className = "detail-links";
    if (r.homepage) {
      const a = document.createElement("a");
      a.href = r.homepage; a.target = "_blank"; a.rel = "noreferrer"; a.textContent = "主页";
      links.appendChild(a);
    }
    if (r.history_url) {
      const a = document.createElement("a");
      a.href = r.history_url; a.target = "_blank"; a.rel = "noreferrer"; a.textContent = "更新历史";
      links.appendChild(a);
    }
    detail.append(meta, desc, links);

    // Collapsed update log
    const updates = r.updates || [];
    if (updates.length) {
      const upd = document.createElement("details");
      upd.className = "detail-updates";
      const sum = document.createElement("summary");
      sum.textContent = `更新信息（${updates.length}）`;
      upd.appendChild(sum);
      const ul = document.createElement("ul");
      for (const u of updates) {
        const li = document.createElement("li");
        li.innerHTML = `<span class="upd-d">${u.date}</span><span class="upd-m"></span>`;
        li.querySelector(".upd-m").textContent = u.message;
        ul.appendChild(li);
      }
      upd.appendChild(ul);
      detail.appendChild(upd);
    }
  } catch (e) {
    detail.className = "detail";
    detail.textContent = "获取详情失败：" + e;
  }
}

function updateStats() {
  const total = state.packages.length;
  const installed = state.statusLoaded
    ? state.packages.filter((p) => state.installed[p.id]).length
    : "–";
  $("statTotal").textContent = total;
  $("statInstalled").textContent = installed;
  $("statSelected").textContent = state.selected.size;
  $("selInfo").innerHTML = `已选 <b>${state.selected.size}</b>`;
}

function updateActions() {
  const n = state.selected.size;
  $("install").disabled = state.running || n === 0;
  $("uninstall").disabled = state.running || n === 0;
}

function syncSelectAll(filtered) {
  const all = filtered.length > 0 && filtered.every((p) => state.selected.has(p.id));
  $("selectAll").checked = all;
  $("selectAll").indeterminate =
    !all && filtered.some((p) => state.selected.has(p.id));
}

function syncSelectAllCurrent() {
  syncSelectAll(currentFiltered());
}

function currentFiltered() {
  const q = state.query.trim().toLowerCase();
  return state.packages.filter(
    (p) =>
      !q ||
      p.id.toLowerCase().includes(q) ||
      (p.category || "").toLowerCase().includes(q)
  );
}

/* ---------------- selection / filter ---------------- */

$("selectAll").addEventListener("change", (e) => {
  const filtered = currentFiltered();
  if (e.target.checked) filtered.forEach((p) => state.selected.add(p.id));
  else filtered.forEach((p) => state.selected.delete(p.id));
  updateStats();
  render();
});

$("search").addEventListener("input", (e) => {
  state.query = e.target.value;
  render();
});

$("refresh").addEventListener("click", () => loadStatus());

$("checkUpdates").addEventListener("click", async () => {
  const btn = $("checkUpdates");
  btn.disabled = true;
  const orig = btn.innerHTML;
  btn.innerHTML = `<svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-2.6-6.4"/><path d="M21 4v5h-5"/></svg> 检查中…`;
  try {
    const r = await fetch("/api/outdated").then((r) => r.json());
    if (r.error) { toast("检查更新失败：" + r.error); return; }
    const list = r.outdated || [];
    if (list.length === 0) {
      toast("所有已装软件均为最新");
    } else {
      toast(`${list.length} 个软件有更新：${list.slice(0, 6).join("、")}${list.length > 6 ? "…" : ""}`);
    }
  } catch (e) {
    toast("检查更新出错：" + e);
  } finally {
    btn.disabled = false;
    btn.innerHTML = orig;
  }
});

$("install").addEventListener("click", () => run("install"));
$("uninstall").addEventListener("click", () => {
  let ids = [...state.selected];
  if (state.statusLoaded) ids = ids.filter((id) => state.installed[id]);
  if (ids.length === 0) { toast("所选软件里没有已安装项可卸载"); return; }
  run("uninstall", ids);
});

/* ---------------- batch run ---------------- */

function run(action, ids) {
  ids = ids && ids.length ? ids : [...state.selected];
  if (ids.length === 0) return;

  state.running = true;
  updateActions();

  const progList = $("progList");
  const barFill = $("barFill");
  const title = $("progTitle");
  const sub = $("progSub");
  const badge = $("progBadge");
  const summary = $("progSummary");
  const closeBtn = $("closeProgress");

  title.textContent = action === "install" ? "正在安装" : "正在卸载";
  sub.textContent = `0 / ${ids.length}`;
  badge.textContent = "";
  summary.textContent = "准备中…";
  barFill.style.width = "0%";
  progList.innerHTML = "";
  closeBtn.hidden = true;
  $("overlay").hidden = false;

  const items = new Map();
  const ensure = (id) => {
    if (!items.has(id)) {
      const li = document.createElement("li");
      li.innerHTML = `<span class="led"></span><span class="pid"></span><span class="err"></span>`;
      li.querySelector(".pid").textContent = id;
      progList.appendChild(li);
      items.set(id, li);
    }
    return items.get(id);
  };

  let done = 0;
  const url = `/api/run?action=${action}&ids=${encodeURIComponent(ids.join(","))}`;
  const es = new EventSource(url);

  es.addEventListener("start", (ev) => {
    const d = JSON.parse(ev.data);
    sub.textContent = `0 / ${d.total}`;
    summary.textContent = "进行中…";
  });

  es.addEventListener("progress", (ev) => {
    const d = JSON.parse(ev.data);
    done += 1;
    const li = ensure(d.id);
    li.classList.remove("ok", "fail", "skip");
    li.classList.add(d.status === "success" ? "ok" : d.status === "failed" ? "fail" : "skip");
    li.querySelector(".err").textContent =
      d.error ||
      (d.status === "success" ? "完成" : d.status === "failed" ? "失败" : "跳过");
    sub.textContent = `${done} / ${d.total}`;
    barFill.style.width = `${Math.round((done / d.total) * 100)}%`;
  });

  es.addEventListener("done", (ev) => {
    const d = JSON.parse(ev.data);
    es.close();
    state.running = false;
    updateActions();
    if (d.error) {
      summary.textContent = "错误：" + d.error;
    } else {
      barFill.style.width = "100%";
      const parts = [`<b>${d.succeeded}</b> 成功`];
      if (d.skipped) parts.push(`<b>${d.skipped}</b> 跳过`);
      if (d.failed) parts.push(`<b>${d.failed}</b> 失败`);
      summary.innerHTML = parts.join("　") + `　用时 ${d.duration}`;
    }
    closeBtn.hidden = false;
    loadStatus();
  });

  es.onerror = () => {
    es.close();
    state.running = false;
    updateActions();
    summary.textContent = "连接中断或执行失败";
    closeBtn.hidden = false;
  };
}

$("closeProgress").addEventListener("click", () => ($("overlay").hidden = true));

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") {
    if (!$("overlay").hidden && !state.running) $("overlay").hidden = true;
    else if (!$("saveOverlay").hidden) $("saveOverlay").hidden = true;
    else if ($("kbdHelp") && !$("kbdHelp").hidden) $("kbdHelp").hidden = true;
  }
  // vim-style list navigation: j/k move, Space toggles, Enter opens detail.
  const tag = (e.target && e.target.tagName || "").toLowerCase();
  const typing = tag === "input" || tag === "textarea" || tag === "select" || (e.target && e.target.isContentEditable);
  const cm = document.getElementById("cmdk");
  const cmOpen = cm && !cm.hidden;
  const active = document.activeElement;
  const onControl = active && active !== document.body && !(active.classList && active.classList.contains("row"));
  const helpOpen = $("kbdHelp") && !$("kbdHelp").hidden;
  if (e.key === "?" && !typing && !cmOpen) { e.preventDefault(); if ($("kbdHelp")) $("kbdHelp").hidden = helpOpen; return; }
  if (typing || cmOpen || helpOpen) return; // never act behind an open help/palette
  if (kbIndex < 0 && onControl) return; // let buttons/links keep their native keys
  if (e.key === "g") { const now = Date.now(); if (now - lastG < 600) { e.preventDefault(); setKb(0); lastG = 0; return; } lastG = now; return; }
  if (e.key === "G") { e.preventDefault(); const rows = visibleRows(); if (rows.length) setKb(rows.length - 1); return; }
  if (e.key === "j") { e.preventDefault(); setKb(kbIndex < 0 ? 0 : kbIndex + 1); }
  else if (e.key === "k") { e.preventDefault(); setKb(kbIndex < 0 ? 0 : kbIndex - 1); }
  else if (e.key === " " || e.key === "Spacebar") {
    const rows = visibleRows();
    if (kbIndex >= 0 && rows[kbIndex]) {
      e.preventDefault();
      const cb = rows[kbIndex].querySelector('input[type=checkbox]');
      if (cb) { cb.checked = !cb.checked; cb.dispatchEvent(new Event("change")); }
    }
  } else if (e.key === "Enter") {
    const rows = visibleRows();
    if (kbIndex >= 0 && rows[kbIndex]) { e.preventDefault(); rows[kbIndex].click(); }
  }
});

function toast(msg) {
  toast.el = toast.el || document.createElement("div");
  const el = toast.el;
  el.textContent = msg;
  el.style.cssText =
    "position:fixed;left:50%;bottom:80px;transform:translateX(-50%);background:var(--panel);border:1px solid var(--line);color:var(--ink);padding:9px 16px;border-radius:8px;box-shadow:var(--shadow);font-size:13px;z-index:80;opacity:0;transition:opacity .2s";
  document.body.appendChild(el);
  requestAnimationFrame(() => (el.style.opacity = "1"));
  clearTimeout(toast.t);
  toast.t = setTimeout(() => (el.style.opacity = "0"), 1800);
}

/* ---------------- LLM settings ---------------- */

if ($("kbdBtn")) $("kbdBtn").addEventListener("click", () => { $("kbdHelp").hidden = false; });
if ($("kbdHelp")) $("kbdHelp").addEventListener("click", (e) => { if (e.target === $("kbdHelp")) $("kbdHelp").hidden = true; });
$("openLLM").addEventListener("click", openLLM);
$("llmCancel").addEventListener("click", () => ($("llmOverlay").hidden = true));
$("llmSave").addEventListener("click", saveLLM);

async function openLLM() {
  $("llmError").textContent = "";
  try {
    const r = await fetch("/api/llm").then((r) => r.json());
    $("llmBase").value = r.base_url || "";
    $("llmModel").value = r.model || "";
    $("llmKey").value = r.api_key || "";
    $("llmKey").placeholder = r.configured ? "（已配置，留空保持不变）" : "sk-…";
  } catch (e) {}
  $("llmOverlay").hidden = false;
  setTimeout(() => $("llmBase").focus(), 30);
}

async function saveLLM() {
  $("llmError").textContent = "";
  const body = {
    base_url: $("llmBase").value.trim(),
    model: $("llmModel").value.trim(),
    api_key: $("llmKey").value,
  };
  try {
    const r = await fetch("/api/llm", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then((r) => r.json());
    if (r.error) { $("llmError").textContent = r.error; return; }
    $("llmOverlay").hidden = true;
    toast(r.configured ? "LLM 已配置，软件详情将自动补全简介" : "已保存（配置不完整）");
  } catch (e) {
    $("llmError").textContent = "保存失败：" + e;
  }
}

/* ---------------- presets ---------------- */

async function loadPresetList(selected) {
  try {
    const r = await fetch("/api/presets").then((r) => r.json());
    const ul = $("presetList");
    ul.innerHTML = "";
    const names = r.presets || [];
    if (names.length === 0) {
      ul.innerHTML = '<li class="muted">尚无预设</li>';
      return;
    }
    for (const n of names) {
      const li = document.createElement("li");
      li.className = "preset-item" + (n === selected ? " active" : "");
      li.dataset.name = n;
      const name = document.createElement("span");
      name.className = "preset-name";
      name.textContent = n;
      const del = document.createElement("button");
      del.className = "preset-del";
      del.textContent = "×";
      del.title = "删除预设";
      del.addEventListener("click", async (e) => {
        e.stopPropagation();
        if (!confirm(`删除预设 “${n}”？`)) return;
        const rr = await fetch("/api/preset?action=delete", {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name: n }),
        }).then((r) => r.json());
        if (rr.error) { toast(rr.error); return; }
        toast(`已删除 “${n}”`);
        loadPresetList();
      });
      li.addEventListener("click", () => loadPresetByName(n));
      li.append(name, del);
      ul.appendChild(li);
    }
  } catch (e) {
    console.error("presets:", e);
  }
}

async function loadPresetByName(name) {
  try {
    const r = await fetch(`/api/preset?name=${encodeURIComponent(name)}`).then((r) => r.json());
    if (r.error) { toast(r.error); return; }
    state.packages = r.packages || [];
    state.selected.clear();
    state.installed = {};
    state.statusLoaded = false;
    $("source").textContent = "preset: " + name;
    render();
    updateStats();
    loadStatus();
    loadPresetList(name);
    toast(`已加载预设 “${name}”`);
  } catch (e) {
    toast("加载失败：" + e);
  }
}

$("presetSave").addEventListener("click", () => {
  $("saveError").textContent = "";
  $("saveName").value = "";
  $("saveOverlay").hidden = false;
  setTimeout(() => $("saveName").focus(), 30);
});
$("saveCancel").addEventListener("click", () => ($("saveOverlay").hidden = true));
$("saveName").addEventListener("keydown", (e) => { if (e.key === "Enter") $("saveConfirm").click(); });
$("saveConfirm").addEventListener("click", async () => {
  const name = $("saveName").value.trim();
  if (!name) { $("saveError").textContent = "请输入名称"; return; }
  const body = {
    name,
    packages: state.packages.map((p) => ({ id: p.id, category: p.category, kind: p.kind, optional: p.optional })),
  };
  try {
    const r = await fetch("/api/preset?action=save", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then((r) => r.json());
    if (r.error) { $("saveError").textContent = r.error; return; }
    $("saveOverlay").hidden = true;
    toast(`已保存 “${name}”（${r.total} 个包）`);
    await loadPresetList(name);
  } catch (e) {
    $("saveError").textContent = "保存失败：" + e;
  }
});

/* ---------------- cross-source repository search ---------------- */
// The single search box both filters the loaded manifest and searches the
// Homebrew / npm / GitHub repositories. Manifest matches show in the list;
// repository hits show in a section below, each with an "添加" button.

const repoState = { timer: null, req: 0 };

$("search").addEventListener("input", (e) => {
  state.query = e.target.value;
  render(); // filter the loaded list immediately
  clearTimeout(repoState.timer);
  const q = state.query.trim();
  if (!q) {
    $("repoResults").hidden = true;
    $("repoResults").innerHTML = "";
    return;
  }
  repoState.timer = setTimeout(() => runRepoSearch(q), 350);
});

async function runRepoSearch(q) {
  const my = ++repoState.req;
  const el = $("repoResults");
  el.hidden = false;
  el.innerHTML = `<div class="repo-head">搜索仓库中…</div>`;
  try {
    const r = await fetch(`/api/search?q=${encodeURIComponent(q)}`).then((r) => r.json());
    if (my !== repoState.req) return;
    if (r.error) { el.innerHTML = `<div class="repo-head">搜索失败：${r.error}</div>`; return; }
    const hits = r.hits || [];
    const installed = r.installed || {};
    const inManifest = r.inManifest || {};
    // Group by source for the heading.
    const bySrc = {};
    for (const h of hits) (bySrc[h.source] ||= []).push(h);
    if (hits.length === 0) {
      el.innerHTML = `<div class="repo-head">仓库中没有与 “${q}” 匹配的结果</div>`;
      return;
    }
    el.innerHTML = "";
    const head = document.createElement("div");
    head.className = "repo-head";
    head.textContent = `仓库搜索结果（${hits.length}）— 点击添加到清单`;
    el.appendChild(head);
    for (const h of hits) {
      const row = document.createElement("div");
      row.className = "srow";
      const icon = document.createElement("span");
      icon.className = "pkg-icon mono";
      icon.textContent = (h.name[0] || "?").toUpperCase();
      icon.style.background = monoColor(h.name);
      const main = document.createElement("div");
      main.className = "srow-main";
      const name = document.createElement("span");
      name.className = "sname";
      name.textContent = h.name;
      name.title = h.name;
      const desc = document.createElement("span");
      desc.className = "sdesc";
      desc.textContent = h.desc || "";
      main.append(name, desc);
      const tag = document.createElement("span");
      tag.className = "kind src-" + h.source;
      tag.textContent = h.source;
      const tail = document.createElement("div");
      tail.className = "srow-tail";
      if (installed[h.name]) {
        const st = document.createElement("span");
        st.className = "sstatus installed";
        st.textContent = "已安装";
        tail.appendChild(st);
      }
      const add = document.createElement("button");
      add.className = "add-btn";
      if (inManifest[h.name]) {
        add.classList.add("added");
        add.textContent = "已添加";
        add.disabled = true;
      } else {
        add.textContent = "添加";
        add.addEventListener("click", () => {
          state.packages.push({
            id: h.name,
            source: h.source,
            kind: h.source === "brew" ? (h.kind || "") : "",
            category: sourceCategory(h.source),
            optional: false,
          });
          add.classList.add("added");
          add.textContent = "已添加";
          add.disabled = true;
          updateStats();
          render();
          toast(`已添加 ${h.name}`);
        });
      }
      tail.appendChild(add);
      row.append(icon, main, tag, tail);
      el.appendChild(row);
    }
  } catch (e) {
    if (my === repoState.req) el.innerHTML = `<div class="repo-head">搜索出错：${e}</div>`;
  }
}

function sourceCategory(src) {
  if (src === "npm") return "npm 包";
  if (src === "github") return "GitHub 工具";
  return h.kind === "cask" ? "Apps" : "Dev";
}

loadState();
loadPresetList();
