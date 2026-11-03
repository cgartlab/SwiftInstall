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
    console.error(e);
    $("list").innerHTML = `<div class="empty"><div class="empty-glyph">⚠</div><div>无法加载清单：${e}</div></div>`;
  }
}

async function loadStatus() {
  state.statusLoaded = false;
  render();
  try {
    const r = await fetch("/api/status").then((r) => r.json());
    if (r.error) throw new Error(r.error);
    state.installed = r.installed || {};
  } catch (e) {
    console.error("status:", e);
  }
  state.statusLoaded = true;
  render();
}

function statusOf(id) {
  if (!state.statusLoaded) return "unknown";
  return state.installed[id] ? "installed" : "missing";
}

/* ---------------- rendering ---------------- */

function render() {
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
    const h = document.createElement("div");
    h.className = "section-title";
    h.textContent = cat;
    sec.appendChild(h);
    const card = document.createElement("div");
    card.className = "card";
    for (const p of items) card.appendChild(renderRow(p));
    sec.appendChild(card);
    listEl.appendChild(sec);
  }

  $("empty").hidden = filtered.length > 0;
  syncSelectAll(filtered);
  updateCounts();
  updateActions();
}

function renderRow(p) {
  const row = document.createElement("div");
  row.className = "row";

  const cb = document.createElement("input");
  cb.type = "checkbox";
  cb.checked = state.selected.has(p.id);
  cb.addEventListener("change", () => {
    if (cb.checked) state.selected.add(p.id);
    else state.selected.delete(p.id);
    updateCounts();
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
    opt.textContent = "optional";
    main.appendChild(opt);
  }

  const kind = document.createElement("span");
  kind.className = "kind" + (p.kind === "cask" ? " cask" : "");
  kind.textContent = p.kind || "auto";

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
  return row;
}

function updateCounts() {
  const total = state.packages.length;
  const installed = state.statusLoaded
    ? state.packages.filter((p) => state.installed[p.id]).length
    : "…";
  $("counts").innerHTML =
    `共 <b>${total}</b> 个 · 已安装 <b>${installed}</b> · 已选 <b>${state.selected.size}</b>`;
  $("selInfo").innerHTML = `已选 <b>${state.selected.size}</b> 项`;
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
  const q = state.query.trim().toLowerCase();
  const filtered = state.packages.filter(
    (p) =>
      !q ||
      p.id.toLowerCase().includes(q) ||
      (p.category || "").toLowerCase().includes(q)
  );
  syncSelectAll(filtered);
}

/* ---------------- actions ---------------- */

function currentFiltered() {
  const q = state.query.trim().toLowerCase();
  return state.packages.filter(
    (p) =>
      !q ||
      p.id.toLowerCase().includes(q) ||
      (p.category || "").toLowerCase().includes(q)
  );
}

$("selectAll").addEventListener("change", (e) => {
  const filtered = currentFiltered();
  if (e.target.checked) filtered.forEach((p) => state.selected.add(p.id));
  else filtered.forEach((p) => state.selected.delete(p.id));
  render();
});

$("search").addEventListener("input", (e) => {
  state.query = e.target.value;
  render();
});

$("refresh").addEventListener("click", () => loadStatus());

$("install").addEventListener("click", () => run("install"));
$("uninstall").addEventListener("click", () => {
  let ids = [...state.selected];
  if (state.statusLoaded) {
    ids = ids.filter((id) => state.installed[id]);
  }
  if (ids.length === 0) {
    toast("所选软件里没有已安装项可卸载");
    return;
  }
  run("uninstall", ids);
});

function run(action, ids) {
  ids = ids && ids.length ? ids : [...state.selected];
  if (ids.length === 0) return;

  state.running = true;
  updateActions();

  const overlay = $("overlay");
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
  overlay.hidden = false;

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
    const cls =
      d.status === "success" ? "ok" : d.status === "failed" ? "fail" : "skip";
    li.classList.add(cls);
    const label =
      d.status === "success"
        ? "完成"
        : d.status === "failed"
        ? "失败"
        : "跳过";
    li.querySelector(".err").textContent = d.error || label;
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
      summary.innerHTML = parts.join(" · ") + `　用时 ${d.duration}`;
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

$("closeProgress").addEventListener("click", () => {
  $("overlay").hidden = true;
});

document.addEventListener("keydown", (e) => {
  if (e.key === "/" && document.activeElement !== $("search")) {
    e.preventDefault();
    $("search").focus();
  }
  if (e.key === "Escape" && !$("overlay").hidden && !state.running) {
    $("overlay").hidden = true;
  }
});

function toast(msg) {
  toast.el = toast.el || document.createElement("div");
  const el = toast.el;
  el.textContent = msg;
  el.style.cssText =
    "position:fixed;left:50%;bottom:80px;transform:translateX(-50%);background:var(--panel);border:1px solid var(--border);color:var(--text);padding:9px 16px;border-radius:999px;box-shadow:var(--shadow);font-size:13px;z-index:80;opacity:0;transition:opacity .2s";
  document.body.appendChild(el);
  requestAnimationFrame(() => (el.style.opacity = "1"));
  clearTimeout(toast.t);
  toast.t = setTimeout(() => (el.style.opacity = "0"), 1800);
}

loadState();
