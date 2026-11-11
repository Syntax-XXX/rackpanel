/* RackPanel – web UI (no framework, no build step) */
'use strict';

const REPO = 'https://github.com/Syntax-XXX/rackpanel';
const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const store = {
  get(k) { try { return localStorage.getItem(k); } catch { return null; } },
  set(k, v) { try { v == null ? localStorage.removeItem(k) : localStorage.setItem(k, v); } catch { /* private window */ } },
};

// ── Icons (line style) ──
const P = {
  grid: '<rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/>',
  box: '<path d="M21 7.5 12 3 3 7.5v9L12 21l9-4.5z"/><path d="M3 7.5 12 12l9-4.5M12 12v9"/>',
  home: '<path d="M3 10.5 12 3l9 7.5"/><path d="M5 9.5V21h14V9.5"/><path d="M10 21v-6h4v6"/>',
  disk: '<rect x="3" y="4" width="18" height="16" rx="3"/><circle cx="12" cy="12" r="4"/><path d="M12 12h.01"/>',
  folder: '<path d="M3 6.5A2.5 2.5 0 0 1 5.5 4H10l2 2.5h6.5A2.5 2.5 0 0 1 21 9v8.5a2.5 2.5 0 0 1-2.5 2.5h-13A2.5 2.5 0 0 1 3 17.5z"/>',
  share: '<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="m8.6 13.5 6.8 4M15.4 6.5l-6.8 4"/>',
  users: '<circle cx="9" cy="8" r="4"/><path d="M2 21a7 7 0 0 1 14 0"/><path d="M16 4a4 4 0 0 1 0 8M22 21a6 6 0 0 0-4-5.6"/>',
  net: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"/>',
  layers: '<path d="m12 2 10 5-10 5L2 7z"/><path d="m2 17 10 5 10-5M2 12l10 5 10-5"/>',
  update: '<path d="M21 12a9 9 0 1 1-3-6.7L21 8"/><path d="M21 3v5h-5"/><path d="M12 8v5l3 2"/>',
  logs: '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6M8 13h8M8 17h5"/>',
  sliders: '<path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6"/>',
  power: '<path d="M12 2v10"/><path d="M18.4 6.6a9 9 0 1 1-12.8 0"/>',
  server: '<rect x="2" y="3" width="20" height="8" rx="2"/><rect x="2" y="13" width="20" height="8" rx="2"/><path d="M6 7h.01M6 17h.01"/>',
  cpu: '<rect x="5" y="5" width="14" height="14" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M9 2v3M15 2v3M9 19v3M15 19v3M2 9h3M2 15h3M19 9h3M19 15h3"/>',
  temp: '<path d="M14 14.8V4a2 2 0 0 0-4 0v10.8a4 4 0 1 0 4 0z"/>',
  fan: '<circle cx="12" cy="12" r="2"/><path d="M12 10c-1-4 0-7 3-7 2 0 3 2 2 4-1 1.5-3 2.3-5 3M14 12c4-1 7 0 7 3 0 2-2 3-4 2-1.5-1-2.3-3-3-5M12 14c1 4 0 7-3 7-2 0-3-2-2-4 1-1.5 3-2.3 5-3M10 12c-4 1-7 0-7-3 0-2 2-3 4-2 1.5 1 2.3 3 3 5"/>',
  file: '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/>',
  image: '<rect x="3" y="3" width="18" height="18" rx="3"/><circle cx="9" cy="9" r="2"/><path d="m21 15-5-5L5 21"/>',
  film: '<rect x="2" y="4" width="20" height="16" rx="3"/><path d="M7 4v16M17 4v16M2 9h5M2 15h5M17 9h5M17 15h5"/>',
  music: '<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>',
  archive: '<rect x="2" y="3" width="20" height="5" rx="1.5"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8M10 12h4"/>',
  upload: '<path d="M12 16V4M6 10l6-6 6 6"/><path d="M4 20h16"/>',
  download: '<path d="M12 4v12M6 10l6 6 6-6"/><path d="M4 20h16"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  trash: '<path d="M3 6h18M8 6V4h8v2M6 6l1 15h10l1-15"/>',
  edit: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>',
  copy: '<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1"/>',
  cut: '<circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M20 4 8.1 15.9M14.5 14.5 20 20M8.1 8.1 12 12"/>',
  paste: '<rect x="6" y="4" width="12" height="17" rx="2"/><path d="M9 4V2h6v2"/>',
  logout: '<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/>',
  play: '<path d="m6 4 14 8-14 8z"/>',
  stop: '<rect x="5" y="5" width="14" height="14" rx="2"/>',
  rotate: '<path d="M3 12a9 9 0 0 1 15-6.7L21 8M21 3v5h-5M21 12a9 9 0 0 1-15 6.7L3 16M3 21v-5h5"/>',
  key: '<circle cx="7.5" cy="15.5" r="4.5"/><path d="m10.7 12.3 9.3-9.3M17 6l3 3M14 9l2 2"/>',
  moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 3"/>',
  shield: '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>',
  up: '<path d="M12 19V5M5 12l7-7 7 7"/>',
  link: '<path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7"/>',
};
const ic = (n) => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${P[n] || ''}</svg>`;

// ── Formatting ──
function fmtBytes(b, d = 1) {
  b = Number(b) || 0;
  const u = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let i = 0;
  while (b >= 1024 && i < u.length - 1) { b /= 1024; i++; }
  return `${b.toFixed(i ? d : 0)} ${u[i]}`;
}
function fmtUptime(s) {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
  return d ? `${d}d ${h}h` : h ? `${h}h ${m}m` : `${m}m`;
}
const fmtDate = (t) => new Date(t).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' });
const pct = (a, b) => (b ? Math.round((a / b) * 100) : 0);
const level = (p) => (p >= 90 ? 'bad' : p >= 75 ? 'warn' : '');

// ── API ──
let tok = null;
try { tok = JSON.parse(store.get('rp.tok') || 'null'); } catch { tok = null; }
let refreshing = null;

async function refresh() {
  if (!refreshing) {
    refreshing = fetch('/api/refresh', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: tok?.refresh_token }) })
      .then((r) => r.json())
      .then((j) => {
        if (j.success !== 200) throw new Error('expired');
        tok = j.data; store.set('rp.tok', JSON.stringify(tok));
      })
      .finally(() => { refreshing = null; });
  }
  return refreshing;
}

async function api(method, path, body) {
  if (tok && tok.expires_at * 1000 < Date.now() + 60000 && path !== 'login') {
    try { await refresh(); } catch { logout(); throw new Error('Please sign in again'); }
  }
  const headers = {};
  if (tok) headers.Authorization = tok.access_token;
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const r = await fetch('/api/' + path, { method, headers, body: body !== undefined ? JSON.stringify(body) : undefined });
  let j;
  try { j = await r.json(); } catch { throw new Error(`Keine Antwort vom NAS (${r.status})`); }
  if (r.status === 401 && path !== 'login') { logout(); throw new Error('Please sign in again'); }
  if (j.success !== 200) throw new Error(j.message || 'Error');
  return j.data;
}
const authQ = () => 'token=' + encodeURIComponent(tok?.access_token || '');

// ── Feedback, dialogs ──
function toast(msg, kind = '') {
  const t = document.createElement('div');
  t.className = 'toast ' + kind;
  t.textContent = msg;
  $('#toasts').append(t);
  setTimeout(() => t.remove(), kind === 'bad' ? 7000 : 3500);
}
const fail = (e) => toast(e.message || String(e), 'bad');

function modal({ title, body = '', wide = false, buttons = [{ label: 'Close' }], onOpen }) {
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = `<div class="modal glass ${wide ? 'wide' : ''}" role="dialog" aria-modal="true" aria-label="${esc(title)}">
    <h3><span class="grow">${esc(title)}</span><button class="btn icon sm" data-x aria-label="Close">✕</button></h3>
    <div class="mb">${body}</div><div class="foot"></div></div>`;
  const close = () => { bg.remove(); document.removeEventListener('keydown', onKey); };
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  document.addEventListener('keydown', onKey);
  bg.addEventListener('mousedown', (e) => { if (e.target === bg) close(); });
  $('[data-x]', bg).onclick = close;
  for (const b of buttons) {
    const el = document.createElement('button');
    el.className = 'btn ' + (b.cls || '');
    el.textContent = b.label;
    el.onclick = async () => {
      if (!b.onClick) return close();
      el.disabled = true;
      try { if ((await b.onClick(bg)) !== false) close(); } catch (e) { fail(e); } finally { el.disabled = false; }
    };
    $('.foot', bg).append(el);
  }
  document.body.append(bg);
  (bg.querySelector('input,select,textarea') || $('.foot .btn:last-child', bg))?.focus();
  onOpen?.(bg, close);
  return { el: bg, close };
}

const confirmDlg = (text, label = 'OK', cls = 'danger') => new Promise((res) => {
  modal({ title: 'Please confirm', body: `<p>${esc(text)}</p>`, buttons: [
    { label: 'Cancel', onClick: () => res(false) },
    { label, cls, onClick: () => res(true) },
  ] });
  // closing via ✕/Esc counts as cancel
  setTimeout(() => { const bg = $$('.modal-bg').pop(); new MutationObserver((_, o) => { if (!bg.isConnected) { res(false); o.disconnect(); } }).observe(document.body, { childList: true }); });
});

const promptDlg = (title, label, value = '', type = 'text') => new Promise((res) => {
  modal({ title, body: `<div class="field"><label>${esc(label)}</label><input class="input" type="${type}" value="${esc(value)}"></div>`,
    buttons: [{ label: 'Cancel', onClick: () => res(null) }, { label: 'OK', cls: 'primary', onClick: (m) => res($('input', m).value) }],
    onOpen: (m, close) => { const i = $('input', m); i.select(); i.onkeydown = (e) => { if (e.key === 'Enter') { res(i.value); close(); } }; } });
});

// background job (apt …) with live output
function runJob(id, title, onDone) {
  const m = modal({ title, wide: true, body: '<pre class="code" data-o>Starting …</pre><p class="muted" data-s></p>' });
  const tick = async () => {
    if (!m.el.isConnected) return;
    try {
      const j = await api('GET', 'jobs/' + id);
      const pre = $('[data-o]', m.el);
      const atEnd = pre.scrollTop + pre.clientHeight >= pre.scrollHeight - 30;
      pre.textContent = j.output || '…';
      if (atEnd) pre.scrollTop = pre.scrollHeight;
      if (j.running) { $('[data-s]', m.el).textContent = 'Running …'; setTimeout(tick, 1500); return; }
      $('[data-s]', m.el).textContent = j.exit_code === 0 ? 'Done.' : `Finished with exit code ${j.exit_code}.`;
      toast(j.exit_code === 0 ? title + ': done' : title + ': failed', j.exit_code === 0 ? 'ok' : 'bad');
      onDone?.(j.exit_code === 0);
    } catch (e) {
      // a RackPanel update restarts the panel, which forgets running jobs: wait for it and reload
      if (!/unknown|fetch|network|load failed/i.test(e.message)) return fail(e);
      $('[data-s]', m.el).textContent = 'RackPanel was updated and is restarting …';
      const back = () => fetch('/api/me', { headers: { Authorization: tok?.access_token || '' } }).then((r) => (r.status < 500 ? location.reload() : setTimeout(back, 2000))).catch(() => setTimeout(back, 2000));
      setTimeout(back, 3000);
    }
  };
  tick();
}
const installPkgs = (pkgs, onDone) => api('POST', 'packages/install', { packages: pkgs }).then((id) => runJob(id, 'Install: ' + pkgs, onDone)).catch(fail);

function gaugeSVG(p, color = 'url(#gg)') {
  const r = 46, c = 2 * Math.PI * r;
  return `<svg viewBox="0 0 108 108"><defs><linearGradient id="gg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#5eead4"/><stop offset="1" stop-color="#818cf8"/></linearGradient></defs>
    <circle cx="54" cy="54" r="${r}" fill="none" stroke="rgba(255,255,255,.08)" stroke-width="10"/>
    <circle data-arc cx="54" cy="54" r="${r}" fill="none" stroke="${color}" stroke-width="10" stroke-linecap="round" transform="rotate(-90 54 54)"
      stroke-dasharray="${c}" stroke-dashoffset="${c * (1 - p / 100)}" style="transition:stroke-dashoffset .6s"/></svg>`;
}
function setGauge(el, p) { const c = 2 * Math.PI * 46; $('[data-arc]', el).setAttribute('stroke-dashoffset', c * (1 - Math.min(100, p) / 100)); }

// ── Pages ──
const basePages = [
  { id: 'overview', title: 'Overview', icon: 'home', render: pageOverview },
  { id: 'storage', title: 'Storage', icon: 'disk', render: pageStorage },
  { id: 'files', title: 'Files', icon: 'folder', render: pageFiles },
  { id: 'shares', title: 'Shares', icon: 'share', render: pageShares },
  { id: 'users', title: 'Users', icon: 'users', render: pageUsers },
  { id: 'apps', title: 'Apps', icon: 'grid', render: pageApps },
  { id: 'network', title: 'Network', icon: 'net', render: pageNetwork },
  { id: 'services', title: 'Services', icon: 'layers', render: pageServices },
  { id: 'updates', title: 'Updates & Packages', icon: 'update', render: pageUpdates },
  { id: 'logs', title: 'Logs', icon: 'logs', render: pageLogs },
  { id: 'system', title: 'System', icon: 'sliders', render: pageSystem },
];
let pages = basePages;
let cleanup = null;
let hostInfo = { hostname: '' };

// ── Add-ons: /usr/share/rackpanel/addons/<id>/page.js calls RackPanel.addon(id, render) ──
const RackPanel = (window.RackPanel = { addons: {}, addon(id, render) { this.addons[id] = render; } });
function addonCtx(a) {
  const run = (args) => api('POST', `addons/${a.id}/run`, { args });
  return {
    api, toast, fail, modal, confirmDlg, promptDlg, esc, ic, fmtBytes, fmtDate, setSub, reload: () => route(), host: () => hostInfo,
    read: (...args) => api('GET', `addons/${a.id}/run?` + args.map((x) => 'a=' + encodeURIComponent(x)).join('&')).then((r) => r.output),
    run: (...args) => run(args).then((r) => r.output),
    job: (title, ...args) => run(args).then((r) => new Promise((res) => (r.job ? runJob(r.job, title, res) : res(true)))),
  };
}
async function pageAddon(a, v) {
  if (!RackPanel.addons[a.id]) {
    await new Promise((res, rej) => {
      const s = document.createElement('script');
      s.src = `/addons/${encodeURIComponent(a.id)}/${a.page || 'page.js'}?v=${encodeURIComponent(a.version || '')}`;
      s.onload = res; s.onerror = () => rej(new Error(`Add-on “${a.title}” could not be loaded`));
      document.head.append(s);
    });
  }
  const render = RackPanel.addons[a.id];
  if (!render) throw new Error(`Add-on “${a.title}” did not register`);
  setSub(a.description || '');
  return render(v, addonCtx(a));
}

async function shell() {
  const addons = await api('GET', 'addons').catch(() => []);
  const ap = addons.map((a) => ({ id: 'x-' + a.id, title: a.title, icon: P[a.icon] ? a.icon : 'grid', render: (v) => pageAddon(a, v) }));
  const at = basePages.findIndex((p) => p.id === 'network');
  pages = [...basePages.slice(0, at), ...ap, ...basePages.slice(at)];
  $('#root').innerHTML = `<div class="app">
    <aside class="side glass">
      <div class="brand"><span class="brand-mark">${ic('server')}</span><div>RackPanel<small>NAS management</small></div></div>
      <nav class="nav" aria-label="Sections">${pages.map((p) => `<a href="#${p.id}" data-p="${p.id}">${ic(p.icon)}<span>${p.title}</span></a>`).join('')}</nav>
      <div class="side-foot"><a href="${REPO}" target="_blank" rel="noopener">RackPanel</a> · <span data-ver></span></div>
    </aside>
    <section class="main">
      <header class="top glass">
        <div><h1 data-title></h1><div class="sub" data-sub></div></div>
        <div class="grow"></div>
        <span class="chip hide-sm"><span class="dot" data-hdot></span><span data-host>…</span></span>
        <span class="clock hide-sm" data-clock></span>
        <button class="btn icon" data-logout title="Sign out" aria-label="Sign out">${ic('logout')}</button>
      </header>
      <div class="view" data-view></div>
    </section></div>`;
  $('[data-logout]').onclick = logout;
  const clock = () => { const c = $('[data-clock]'); if (c) c.textContent = new Date().toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' }); };
  clock(); setInterval(clock, 10000);
  api('GET', 'overview').then((o) => {
    hostInfo = o;
    $('[data-host]').textContent = o.hostname;
    $('[data-ver]').textContent = 'v' + o.version;
    document.title = o.hostname + ' · RackPanel';
  }).catch(() => {});
  window.onhashchange = route;
  route();
}

function route() {
  const id = location.hash.slice(1).split('/')[0] || 'overview';
  const p = pages.find((x) => x.id === id) || pages[0];
  $$('.nav a').forEach((a) => a.classList.toggle('on', a.dataset.p === p.id));
  $('[data-title]').textContent = p.title;
  $('[data-sub]').textContent = '';
  if (cleanup) { try { cleanup(); } catch { /* ignore */ } cleanup = null; }
  const v = $('[data-view]');
  v.innerHTML = '<div class="empty">Loading …</div>';
  v.scrollTop = 0;
  Promise.resolve(p.render(v)).then((c) => { cleanup = c || null; }).catch((e) => { v.innerHTML = `<div class="card glass empty">${esc(e.message)}</div>`; });
}
const setSub = (t) => { const s = $('[data-sub]'); if (s) s.textContent = t; };

// ── Overview ──
async function pageOverview(v) {
  const o = await api('GET', 'overview');
  hostInfo = o;
  const raidBad = o.raid.filter((r) => /_/.test(r.status) || r.sync);
  const diskBad = o.disks.filter((d) => !d.health || d.reallocated > 0 || d.pending > 0);
  const health = raidBad.length || diskBad.length ? 'warn' : '';
  setSub(`${o.model || 'Server'} · ${o.os}`);
  v.innerHTML = `<div class="grid">
    <div class="card glass c12"><div class="hero">
      <div class="hero-icon">${ic('server')}</div>
      <div><h3>${esc(o.hostname)}</h3><div class="muted">${esc(o.model || '')} · ${esc(o.os)} · Kernel ${esc(o.kernel)}</div></div>
      <div class="facts">
        <div class="fact"><b data-up>${fmtUptime(o.uptime)}</b><span>Uptime</span></div>
        <div class="fact"><b>${o.cpus} × ${esc(o.arch)}</b><span>CPU</span></div>
        <div class="fact"><b>${fmtBytes(o.mem.total, 0)}</b><span>Memory</span></div>
        <div class="fact"><b><span class="badge ${health}">${health ? 'Attention' : 'All healthy'}</span></b><span>Health</span></div>
      </div></div></div>
    <div class="card glass c4"><h2>CPU</h2><div class="gauge" data-g="cpu">${gaugeSVG(0)}<div><div class="val" data-v>–</div><div class="lbl">Load <span data-load>${esc(o.load)}</span></div></div></div></div>
    <div class="card glass c4"><h2>Memory</h2><div class="gauge" data-g="mem">${gaugeSVG(o.mem.usedPercent)}<div><div class="val" data-v>${Math.round(o.mem.usedPercent)} %</div><div class="lbl" data-ml>${fmtBytes(o.mem.used)} of ${fmtBytes(o.mem.total)}</div></div></div></div>
    <div class="card glass c4"><h2>Temperature & fan</h2><div class="rows">
      ${o.cpu_temp ? `<div class="row"><div class="ico">${ic('cpu')}</div><div class="main-col"><div class="t">${o.cpu_temp} °C</div><div class="s">CPU</div></div></div>` : ''}
      <div class="row"><div class="ico">${ic('temp')}</div><div class="main-col"><div class="t">${o.disk_temp ? o.disk_temp + ' °C' : '–'}</div><div class="s">Hottest disk${o.disk_temp ? '' : ' (all asleep)'}</div></div></div>
      ${o.fan && o.fan.max ? `<div class="row"><div class="ico">${ic('fan')}</div><div class="main-col"><div class="t">${o.fan.rpm} rpm</div><div class="s">Fan (level ${o.fan.pwm}/255)</div></div></div>` : ''}
    </div></div>
    <div class="card glass c8"><h2>Network<span class="grow"></span><span class="faint" style="text-transform:none;letter-spacing:0">last 3 minutes</span></h2>
      <svg class="spark" viewBox="0 0 600 90" preserveAspectRatio="none" data-spark>
        <defs><linearGradient id="sg1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#5eead4" stop-opacity=".45"/><stop offset="1" stop-color="#5eead4" stop-opacity="0"/></linearGradient>
        <linearGradient id="sg2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#818cf8" stop-opacity=".4"/><stop offset="1" stop-color="#818cf8" stop-opacity="0"/></linearGradient></defs>
        <path data-a1 fill="url(#sg1)"/><path data-l1 fill="none" stroke="#5eead4" stroke-width="2"/>
        <path data-a2 fill="url(#sg2)"/><path data-l2 fill="none" stroke="#818cf8" stroke-width="2"/></svg>
      <div class="legend"><span><i class="swatch" style="background:#5eead4"></i>Received <b data-rx>–</b></span><span><i class="swatch" style="background:#818cf8"></i>Sent <b data-tx>–</b></span>
      <span>${o.net.map((n) => `<span class="dot ${n.state === 'up' ? '' : 'off'}" style="display:inline-block;margin:0 5px 0 10px"></span>${esc(n.name)}`).join('')}</span></div></div>
    <div class="card glass c4"><h2>RAID arrays</h2><div class="rows">${o.raid.length ? o.raid.filter((r) => r.mount !== 'swap').map((r) => {
      const bad = /_/.test(r.status);
      return `<div class="row"><div class="ico">${ic('layers')}</div><div class="main-col"><div class="t">${esc(r.name)} · ${esc(r.level)}</div><div class="s">${esc(r.mount || '–')} ${r.sync ? '· ' + esc(r.sync) : ''}</div></div><span class="badge ${bad ? 'warn' : ''}">${bad ? 'Degraded' : 'OK'}</span></div>`;
    }).join('') : '<div class="empty">No software RAID</div>'}</div></div>
    <div class="card glass c12"><h2>Volumes<span class="grow"></span><a class="btn sm" href="#storage">Manage storage</a></h2><div class="rows">${o.volumes.map(volRow).join('') || '<div class="empty">No volumes found</div>'}</div></div>
  </div>`;

  // live values
  const hist = [];
  let prev = null;
  const poll = async () => {
    try {
      const s = await api('GET', 'stats');
      const cg = $('[data-g=cpu]', v), mg = $('[data-g=mem]', v);
      if (!cg) return;
      setGauge(cg, s.cpu); $('[data-v]', cg).textContent = Math.round(s.cpu) + ' %'; $('[data-load]', v).textContent = s.load;
      setGauge(mg, s.mem.usedPercent); $('[data-v]', mg).textContent = Math.round(s.mem.usedPercent) + ' %';
      $('[data-ml]', v).textContent = `${fmtBytes(s.mem.used)} of ${fmtBytes(s.mem.total)}`;
      $('[data-up]', v).textContent = fmtUptime(s.uptime);
      const rx = s.net.reduce((a, n) => a + n.bytesRecv, 0), tx = s.net.reduce((a, n) => a + n.bytesSent, 0);
      if (prev) {
        const dt = Math.max(1, s.time - prev.time);
        hist.push([Math.max(0, (rx - prev.rx) / dt), Math.max(0, (tx - prev.tx) / dt)]);
        if (hist.length > 60) hist.shift();
        $('[data-rx]', v).textContent = fmtBytes(hist.at(-1)[0]) + '/s';
        $('[data-tx]', v).textContent = fmtBytes(hist.at(-1)[1]) + '/s';
        drawSpark(v, hist);
      }
      prev = { rx, tx, time: s.time };
    } catch { /* retry next tick */ }
  };
  poll();
  const t = setInterval(poll, 3000);
  return () => clearInterval(t);
}

function drawSpark(v, hist) {
  const max = Math.max(1024, ...hist.flat()) * 1.15, W = 600, H = 90, n = 60;
  for (const k of [0, 1]) {
    const pts = hist.map((h, i) => [((n - hist.length + i) / (n - 1)) * W, H - (h[k] / max) * (H - 4)]);
    const line = pts.map((p, i) => (i ? 'L' : 'M') + p[0].toFixed(1) + ' ' + p[1].toFixed(1)).join(' ');
    $(`[data-l${k + 1}]`, v).setAttribute('d', line);
    $(`[data-a${k + 1}]`, v).setAttribute('d', pts.length ? `${line} L${W} ${H} L${pts[0][0].toFixed(1)} ${H} Z` : '');
  }
}

function volRow(x) {
  const p = pct(x.used, x.size);
  return `<div class="row"><div class="ico">${ic('disk')}</div><div class="main-col"><div class="t">${esc(x.path)} <span class="faint" style="font-weight:400">· ${esc(x.dev)} · ${esc(x.fs)}</span></div>
    <div class="bar ${level(p)}" style="margin:7px 0 4px"><i style="width:${p}%"></i></div>
    <div class="s">${fmtBytes(x.used)} used · ${fmtBytes(x.avail)} free · ${fmtBytes(x.size)} total</div></div><b>${p} %</b></div>`;
}

// ── Storage ──
async function pageStorage(v) {
  const [st, sd] = await Promise.all([api('GET', 'storage'), api('GET', 'storage/spindown').catch(() => 0)]);
  setSub(`${st.disks.length} drives · ${st.volumes.length} volumes`);
  const opts = [0, 10, 20, 30, 60, 120, 180, 240, 330];
  v.innerHTML = `<div class="grid">
    <div class="card glass c12"><h2>Volumes</h2><div class="rows">${st.volumes.map(volRow).join('')}</div></div>
    <div class="card glass c7"><h2>Drives</h2><div class="rows">${st.disks.map((d) => {
      const warn = !d.health || d.reallocated > 0 || d.pending > 0;
      return `<div class="row"><div class="ico">${ic('disk')}</div><div class="main-col">
        <div class="t">${esc(d.name)} · ${esc(d.model || 'Drive')}</div>
        <div class="s">${fmtBytes(d.size)} · ${d.standby ? 'Asleep' : (d.temperature ? d.temperature + ' °C · ' : '') + (d.power_on_hours ? d.power_on_hours.toLocaleString('en-GB') + ' power-on hours' : '')}
        ${d.serial ? ' · SN ' + esc(d.serial) : ''}${d.reallocated ? ' · ' + d.reallocated + ' reallocated sectors' : ''}${d.pending ? ' · ' + d.pending + ' pending sectors' : ''}</div></div>
        <div class="acts"><span class="badge ${d.standby ? 'off' : warn ? 'bad' : ''}">${d.standby ? 'Asleep' : warn ? 'Check!' : 'Healthy'}</span>
        <button class="btn sm" data-smart="${esc(d.name)}">SMART</button>${d.standby ? '' : `<button class="btn sm" data-sleep="${esc(d.name)}">${ic('moon')}Sleep</button>`}</div></div>`;
    }).join('') || '<div class="empty">No drives detected</div>'}</div></div>
    <div class="card glass c5"><h2>Disk standby</h2>
      <p class="muted">Disks spin down after this much idle time. Saves power and makes the device quieter.</p>
      <div class="field"><label for="sd">Spin down after</label><select class="input" id="sd">${opts.map((m) => `<option value="${m}" ${m === sd ? 'selected' : ''}>${m ? (m < 60 ? m + ' minutes' : m / 60 + ' hour' + (m > 60 ? 's' : '')) : 'Never'}</option>`).join('')}</select></div>
      <button class="btn primary" data-sdsave>Apply</button>
      <h2 style="margin-top:26px">Software-RAID</h2><div class="rows">${st.raid.map((r) => {
        const bad = /_/.test(r.status);
        return `<div class="row"><div class="main-col"><div class="t">${esc(r.name)} · ${esc(r.level)} · ${esc(r.mount || '–')}</div><div class="s">${esc(r.members.join(', '))} ${esc(r.status)} ${r.sync ? '<br>' + esc(r.sync) : ''}</div></div><span class="badge ${bad ? 'warn' : ''}">${bad ? 'Degraded' : esc(r.state)}</span></div>`;
      }).join('') || '<div class="empty">No software RAID</div>'}</div>
    </div></div>`;
  $$('[data-smart]', v).forEach((b) => (b.onclick = async () => {
    const t = await api('GET', 'storage/smart/' + b.dataset.smart).catch(fail);
    if (t != null) modal({ title: 'SMART · ' + b.dataset.smart, wide: true, body: `<pre class="code">${esc(t)}</pre>` });
  }));
  $$('[data-sleep]', v).forEach((b) => (b.onclick = () => api('POST', `storage/${b.dataset.sleep}/sleep`).then(() => { toast(b.dataset.sleep + ' is asleep', 'ok'); route(); }).catch(fail)));
  $('[data-sdsave]', v).onclick = () => api('PUT', 'storage/spindown', { minutes: +$('#sd', v).value }).then(() => toast('Standby time set', 'ok')).catch(fail);
}

// ── Files ──
const fileState = { path: '', sel: new Set(), clip: null };
const TEXT_EXT = /\.(txt|md|conf|cfg|ini|log|json|ya?ml|xml|csv|sh|py|js|ts|css|html?|toml|env|service|list|properties|nfo|srt)$/i;
function fileIcon(it) {
  if (it.is_dir) return `<span style="color:#7dd3fc">${ic('folder')}</span>`;
  const n = it.name.toLowerCase();
  if (/\.(jpe?g|png|gif|webp|bmp|heic|svg)$/.test(n)) return `<span style="color:#f9a8d4">${ic('image')}</span>`;
  if (/\.(mp4|mkv|avi|mov|webm|m4v)$/.test(n)) return `<span style="color:#c4b5fd">${ic('film')}</span>`;
  if (/\.(mp3|flac|wav|ogg|m4a|aac)$/.test(n)) return `<span style="color:#fcd34d">${ic('music')}</span>`;
  if (/\.(zip|tar|gz|7z|rar|xz|bz2|iso)$/.test(n)) return `<span style="color:#fdba74">${ic('archive')}</span>`;
  return `<span class="muted">${ic('file')}</span>`;
}

async function pageFiles(v) {
  const p = decodeURIComponent(location.hash.split('/').slice(1).join('/') || '');
  fileState.path = p ? '/' + p.replace(/^\/+/, '') : '';
  fileState.sel.clear();
  const list = await api('GET', 'files?path=' + encodeURIComponent(fileState.path || '/'));
  const items = list.content || [];
  const cur = fileState.path;
  const parts = cur.split('/').filter(Boolean);
  setSub(cur || 'All locations');
  v.innerHTML = `<div class="card glass" data-drop>
    <div class="toolbar">
      <div class="crumbs"><button data-go="">Locations</button>${parts.map((x, i) => `<span class="faint">›</span><button data-go="/${esc(parts.slice(0, i + 1).join('/'))}">${esc(x)}</button>`).join('')}</div>
      <span class="grow"></span>
      ${cur ? `<button class="btn sm" data-mk>${ic('plus')}Folder</button>
      <label class="btn sm primary">${ic('upload')}Upload<input type="file" multiple hidden data-up></label>` : ''}
    </div>
    <div class="toolbar" data-selbar>
      <button class="btn sm" data-dl disabled>${ic('download')}Download</button>
      <button class="btn sm" data-ren disabled>${ic('edit')}Rename</button>
      <button class="btn sm" data-cp disabled>${ic('copy')}Copy</button>
      <button class="btn sm" data-mv disabled>${ic('cut')}Cut</button>
      ${fileState.clip && cur ? `<button class="btn sm" data-paste>${ic('paste')}Paste (${fileState.clip.paths.length})</button>` : ''}
      <button class="btn sm danger" data-del disabled>${ic('trash')}Delete</button>
      <span class="grow"></span><span class="faint" data-count>${list.total} items</span>
    </div>
    <div data-tasks></div>
    ${items.length ? `<table class="files"><thead><tr><th style="width:32px"><input type="checkbox" data-all aria-label="Select all"></th><th>Name</th><th class="num hide-sm">Size</th><th class="num hide-sm">Modified</th></tr></thead><tbody>
    ${items.map((it, i) => `<tr data-i="${i}"><td><input type="checkbox" data-s="${i}" aria-label="Select ${esc(it.name)}"></td>
      <td><div class="name" data-open="${i}">${fileIcon(it)}<span>${esc(it.name)}</span></div></td>
      <td class="num hide-sm">${it.is_dir ? '' : fmtBytes(it.size)}</td><td class="num hide-sm">${cur ? fmtDate(it.modified) : ''}</td></tr>`).join('')}
    </tbody></table>` : `<div class="empty">${cur ? 'This folder is empty. Drop files here to upload.' : 'No locations found.'}</div>`}
  </div>`;

  const go = (path) => { location.hash = 'files' + (path ? '/' + encodeURIComponent(path.replace(/^\//, '')) : ''); };
  $$('[data-go]', v).forEach((b) => (b.onclick = () => go(b.dataset.go)));
  const selected = () => [...fileState.sel].map((i) => items[i]);
  const upd = () => {
    const n = fileState.sel.size;
    for (const k of ['dl', 'cp', 'mv', 'del']) $(`[data-${k}]`, v).disabled = !n || !cur;
    $('[data-ren]', v).disabled = n !== 1 || !cur;
    $('[data-count]', v).textContent = n ? `${n} selected` : `${list.total} items`;
    $$('tbody tr', v).forEach((tr) => tr.classList.toggle('sel', fileState.sel.has(+tr.dataset.i)));
  };
  $$('[data-s]', v).forEach((c) => (c.onchange = () => { c.checked ? fileState.sel.add(+c.dataset.s) : fileState.sel.delete(+c.dataset.s); upd(); }));
  const all = $('[data-all]', v);
  if (all) all.onchange = () => { $$('[data-s]', v).forEach((c) => { c.checked = all.checked; c.onchange(); }); };
  $$('[data-open]', v).forEach((el) => (el.onclick = () => openItem(items[+el.dataset.open])));

  const reload = () => route();
  const mk = $('[data-mk]', v);
  if (mk) mk.onclick = async () => {
    const n = await promptDlg('New folder', 'Name');
    if (n) api('POST', 'files/mkdir', { path: cur + '/' + n }).then(reload).catch(fail);
  };
  $('[data-ren]', v).onclick = async () => {
    const it = selected()[0];
    const n = await promptDlg('Rename', 'New name', it.name);
    if (n && n !== it.name) api('POST', 'files/rename', { old_path: it.path, new_path: cur + '/' + n }).then(reload).catch(fail);
  };
  $('[data-del]', v).onclick = async () => {
    const s = selected();
    if (await confirmDlg(`Permanently delete ${s.length} item(s)? ${s.map((x) => x.name).slice(0, 5).join(', ')}${s.length > 5 ? ' …' : ''}`, 'Delete'))
      api('POST', 'files/delete', s.map((x) => x.path)).then(() => { toast('Deleted', 'ok'); reload(); }).catch(fail);
  };
  $('[data-dl]', v).onclick = () => {
    const a = document.createElement('a');
    a.href = '/api/files/download?files=' + encodeURIComponent(selected().map((x) => x.path).join(',')) + '&' + authQ();
    a.download = ''; document.body.append(a); a.click(); a.remove();
  };
  const clip = (type) => { fileState.clip = { type, paths: selected().map((x) => x.path) }; toast(`${fileState.clip.paths.length} item(s) ${type === 'copy' ? 'copied' : 'cut'} – open the target folder and press “Paste”`); reload(); };
  $('[data-cp]', v).onclick = () => clip('copy');
  $('[data-mv]', v).onclick = () => clip('move');
  const paste = $('[data-paste]', v);
  if (paste) paste.onclick = () => {
    const c = fileState.clip;
    api('POST', 'files/task', { type: c.type, item: c.paths.map((from) => ({ from })), to: cur, style: 'rename' })
      .then(() => { if (c.type === 'move') fileState.clip = null; watchTasks(v, reload); })
      .catch(fail);
  };

  const up = $('[data-up]', v);
  if (up) up.onchange = () => uploadFiles([...up.files], cur, reload);
  const zone = $('[data-drop]', v);
  if (cur) {
    zone.ondragover = (e) => { e.preventDefault(); zone.classList.add('drop'); };
    zone.ondragleave = () => zone.classList.remove('drop');
    zone.ondrop = (e) => { e.preventDefault(); zone.classList.remove('drop'); uploadFiles([...e.dataTransfer.files], cur, reload); };
  }

  function openItem(it) {
    if (it.is_dir) return go(it.path);
    const src = '/api/files/preview?path=' + encodeURIComponent(it.path) + '&' + authQ();
    const n = it.name.toLowerCase();
    if (/\.(jpe?g|png|gif|webp|bmp|svg)$/.test(n)) return modal({ title: it.name, wide: true, body: `<img src="${src}" alt="${esc(it.name)}" style="max-width:100%;max-height:70vh;display:block;margin:auto;border-radius:12px">` });
    if (/\.(mp4|webm|m4v|mov)$/.test(n)) return modal({ title: it.name, wide: true, body: `<video src="${src}" controls style="width:100%;max-height:70vh;border-radius:12px"></video>` });
    if (/\.(mp3|flac|wav|ogg|m4a|aac)$/.test(n)) return modal({ title: it.name, body: `<audio src="${src}" controls style="width:100%"></audio>` });
    if ((TEXT_EXT.test(n) || !/\./.test(n)) && it.size < 2 << 20) return editFile(it);
    const a = document.createElement('a');
    a.href = '/api/files/download?files=' + encodeURIComponent(it.path) + '&' + authQ(); a.download = it.name; document.body.append(a); a.click(); a.remove();
  }
  async function editFile(it) {
    const txt = await api('GET', 'files/content?path=' + encodeURIComponent(it.path)).catch(fail);
    if (txt == null) return;
    modal({ title: it.name, wide: true, body: `<textarea class="input code" spellcheck="false" aria-label="Content">${esc(txt)}</textarea>`,
      buttons: [{ label: 'Close' }, { label: 'Save', cls: 'primary', onClick: (m) => api('PUT', 'files/content', { path: it.path, content: $('textarea', m).value }).then(() => toast('Saved', 'ok')) }] });
  }
  upd();
}

async function watchTasks(v, done) {
  const box = $('[data-tasks]', v);
  const tick = async () => {
    if (!box.isConnected) return;
    const list = await api('GET', 'files/tasks').catch(() => []);
    const open = list.filter((t) => !t.finished);
    box.innerHTML = open.map((t) => { const p = pct(t.processed_size, t.total_size); return `<div class="row" style="margin-bottom:10px"><div class="main-col"><div class="t">${t.type === 'move' ? 'Moving' : 'Copying'} to ${esc(t.to)}</div><div class="bar progress"><i style="width:${p}%"></i></div><div class="s">${fmtBytes(t.processed_size)} of ${fmtBytes(t.total_size)}</div></div><b>${p} %</b></div>`; }).join('');
    if (open.length) setTimeout(tick, 1000); else { toast('Done', 'ok'); done(); }
  };
  setTimeout(tick, 400);
}

async function uploadFiles(files, dir, done) {
  if (!files.length) return;
  const CH = 8 << 20;
  const total = files.reduce((a, f) => a + f.size, 0);
  let sent = 0;
  const m = modal({ title: 'Upload', body: `<p data-n class="muted"></p><div class="bar progress"><i data-b style="width:0"></i></div><p data-p class="faint"></p>`, buttons: [] });
  try {
    for (const [fi, f] of files.entries()) {
      const chunks = Math.max(1, Math.ceil(f.size / CH));
      const ident = `${f.size}-${f.name}-${f.lastModified}`.replace(/[^\w.-]/g, '_');
      for (let c = 1; c <= chunks; c++) {
        $('[data-n]', m.el).textContent = `${fi + 1}/${files.length}: ${f.name}`;
        const fd = new FormData();
        fd.append('path', dir); fd.append('filename', f.name); fd.append('relativePath', f.name);
        fd.append('totalChunks', chunks); fd.append('chunkNumber', c); fd.append('identifier', ident);
        fd.append('file', f.slice((c - 1) * CH, c * CH), f.name);
        if (tok && tok.expires_at * 1000 < Date.now() + 60000) await refresh();
        const r = await fetch('/api/files/upload', { method: 'POST', headers: { Authorization: tok.access_token }, body: fd });
        const j = await r.json().catch(() => ({}));
        if (j.success !== 200) throw new Error(j.message || 'Upload failed');
        sent += Math.min(CH, f.size - (c - 1) * CH);
        $('[data-b]', m.el).style.width = pct(sent, total) + '%';
        $('[data-p]', m.el).textContent = `${fmtBytes(sent)} of ${fmtBytes(total)}`;
      }
    }
    toast(`${files.length} file(s) uploaded`, 'ok');
  } catch (e) { fail(e); }
  m.close();
  done();
}

// ── Shares ──
async function pageShares(v, tab) {
  tab = tab || location.hash.split('/')[1] || 'smb';
  const [smb, nfs, ul, net] = await Promise.all([api('GET', 'shares/smb'), api('GET', 'shares/nfs'), api('GET', 'users'), api('GET', 'network').catch(() => null)]);
  const roots = await api('GET', 'files/roots').catch(() => []);
  setSub('SMB (Windows, macOS) and NFS (Linux, Proxmox)');
  const users = ul.users.filter((u) => u.uid >= 1000);
  v.innerHTML = `<div class="tabs" role="tablist"><button data-t="smb" class="${tab === 'smb' ? 'on' : ''}">SMB / Windows</button><button data-t="nfs" class="${tab === 'nfs' ? 'on' : ''}">NFS / Linux</button></div><div data-c></div>`;
  $$('[data-t]', v).forEach((b) => (b.onclick = () => { location.hash = 'shares/' + b.dataset.t; }));
  const c = $('[data-c]', v);
  const pathField = (val = '') => `<div class="field"><label>Folder</label><input class="input" data-path list="rootlist" value="${esc(val || roots[0] || '')}"><datalist id="rootlist">${roots.map((r) => `<option value="${esc(r)}">`).join('')}</datalist></div>`;
  const rawEdit = (title, content, save) => modal({ title, wide: true, body: `<textarea class="input code" spellcheck="false" aria-label="${esc(title)}">${esc(content)}</textarea><p class="faint">If the new version is invalid, the previous one is restored automatically.</p>`,
    buttons: [{ label: 'Cancel' }, { label: 'Save', cls: 'primary', onClick: (m) => save($('textarea', m).value) }] });

  if (tab === 'smb') {
    if (!smb.installed) {
      c.innerHTML = `<div class="card glass"><h2>SMB is not installed</h2><p class="muted">Samba lets you map folders as network drives on Windows, macOS and phones.</p><button class="btn primary" data-inst>${ic('download')}Install Samba</button></div>`;
      $('[data-inst]', c).onclick = () => installPkgs('samba wsdd', () => route());
      return;
    }
    const secs = parseIni(smb.content).filter((s) => !['global', 'printers', 'print$'].includes(s.name.toLowerCase()));
    c.innerHTML = `<div class="card glass"><div class="toolbar"><span class="muted">Access with a user that has “SMB” enabled (see Users).</span><span class="grow"></span>
      <button class="btn sm" data-raw>Edit smb.conf</button><button class="btn sm primary" data-add>${ic('plus')}Share</button></div>
      <div class="rows">${secs.map((s, i) => `<div class="row"><div class="ico">${ic('folder')}</div><div class="main-col"><div class="t">\\\\${esc(hostInfo.hostname)}\\${esc(s.name)}</div>
        <div class="s">${esc(s.kv.path || '–')} · ${/^(no|false)$/i.test(s.kv['read only'] || '') || /^(yes|true)$/i.test(s.kv.writable || s.kv.writeable || '') ? 'Read & write' : 'Read only'}${/^(yes|true)$/i.test(s.kv['guest ok'] || '') ? ' · guests allowed' : ''}${s.kv['valid users'] ? ' · ' + esc(s.kv['valid users']) : ''}</div></div>
        <div class="acts"><button class="btn sm danger" data-rm="${i}">${ic('trash')}Remove</button></div></div>`).join('') || '<div class="empty">No shares yet.</div>'}</div></div>`;
    const save = (content) => api('PUT', 'shares/smb', { content }).then(() => { toast('Saved', 'ok'); route(); });
    $('[data-raw]', c).onclick = () => rawEdit('smb.conf', smb.content, save);
    $$('[data-rm]', c).forEach((b) => (b.onclick = async () => {
      const s = secs[+b.dataset.rm];
      if (await confirmDlg(`Remove share “${s.name}”? The files are kept.`, 'Remove')) save(removeLines(smb.content, s.start, s.end)).catch(fail);
    }));
    $('[data-add]', c).onclick = () => modal({ title: 'Add SMB share', body: `
      <div class="field"><label>Name</label><input class="input" data-name placeholder="e.g. Media"></div>${pathField()}
      <label class="check"><input type="checkbox" data-rw checked>Allow writing</label><label class="check"><input type="checkbox" data-guest>Guests without password</label>
      <div class="field" style="margin-top:8px"><label>Allowed users (none = everyone with SMB access)</label><div>${users.map((u) => `<label class="check"><input type="checkbox" data-u="${esc(u.name)}">${esc(u.name)}${u.smb ? '' : ' <span class="faint">(no SMB)</span>'}</label>`).join('') || '<span class="faint">No users – create one first.</span>'}</div></div>`,
      buttons: [{ label: 'Cancel' }, { label: 'Add', cls: 'primary', onClick: (m) => {
        const name = $('[data-name]', m).value.trim();
        if (!/^[\w .-]{1,40}$/.test(name)) throw new Error('Invalid name');
        const vu = $$('[data-u]:checked', m).map((x) => x.dataset.u);
        const block = `\n[${name}]\n   path = ${$('[data-path]', m).value.trim()}\n   browseable = yes\n   read only = ${$('[data-rw]', m).checked ? 'no' : 'yes'}\n   guest ok = ${$('[data-guest]', m).checked ? 'yes' : 'no'}\n   create mask = 0664\n   directory mask = 0775\n${vu.length ? '   valid users = ' + vu.join(' ') + '\n' : ''}`;
        return save(smb.content.replace(/\s*$/, '\n') + block);
      } }] });
  } else {
    if (!nfs.installed) {
      c.innerHTML = `<div class="card glass"><h2>NFS is not installed</h2><p class="muted">NFS is ideal for Linux machines and Proxmox.</p><button class="btn primary" data-inst>${ic('download')}Install NFS server</button></div>`;
      $('[data-inst]', c).onclick = () => installPkgs('nfs-kernel-server', () => route());
      return;
    }
    const lines = nfs.content.split('\n');
    const ex = lines.map((l, i) => ({ l: l.trim(), i })).filter((x) => x.l && !x.l.startsWith('#'));
    const subnet = guessSubnet(net);
    c.innerHTML = `<div class="card glass"><div class="toolbar"><span class="muted">Active exports according to <code>exportfs</code>: ${esc((nfs.active || '').split('\n').filter((x) => x.startsWith('/')).length)}</span><span class="grow"></span>
      <button class="btn sm" data-raw>Edit /etc/exports</button><button class="btn sm primary" data-add>${ic('plus')}Share</button></div>
      <div class="rows">${ex.map((x, k) => { const [p, ...cl] = x.l.split(/\s+/); return `<div class="row"><div class="ico">${ic('folder')}</div><div class="main-col"><div class="t">${esc(p)}</div><div class="s">${esc(cl.join(' '))}</div></div><div class="acts"><button class="btn sm danger" data-rm="${k}">${ic('trash')}Remove</button></div></div>`; }).join('') || '<div class="empty">No shares yet.</div>'}</div></div>`;
    const save = (content) => api('PUT', 'shares/nfs', { content }).then(() => { toast('Saved', 'ok'); route(); });
    $('[data-raw]', c).onclick = () => rawEdit('/etc/exports', nfs.content, save);
    $$('[data-rm]', c).forEach((b) => (b.onclick = async () => {
      const x = ex[+b.dataset.rm];
      if (await confirmDlg(`Remove export ${x.l.split(/\s+/)[0]}?`, 'Remove')) save(removeLines(nfs.content, x.i, x.i + 1)).catch(fail);
    }));
    $('[data-add]', c).onclick = () => modal({ title: 'Add NFS export', body: `${pathField()}
      <div class="field"><label>Allowed clients (IP, subnet or *)</label><input class="input" data-cl value="${esc(subnet)}"></div>
      <label class="check"><input type="checkbox" data-rw checked>Allow writing</label><label class="check"><input type="checkbox" data-nrs>Keep root privileges (no_root_squash, e.g. for Proxmox)</label>`,
      buttons: [{ label: 'Cancel' }, { label: 'Add', cls: 'primary', onClick: (m) => {
        const p = $('[data-path]', m).value.trim(), cl = $('[data-cl]', m).value.trim();
        if (!p.startsWith('/') || /\s/.test(p) || !/^[\w.*/:-]+$/.test(cl)) throw new Error('Invalid path or clients');
        const line = `${p} ${cl}(${$('[data-rw]', m).checked ? 'rw' : 'ro'},sync,no_subtree_check${$('[data-nrs]', m).checked ? ',no_root_squash' : ''})`;
        return save(nfs.content.replace(/\s*$/, '\n') + line + '\n');
      } }] });
  }
}
function parseIni(txt) {
  const out = [], lines = txt.split('\n');
  let cur = null;
  lines.forEach((l, i) => {
    const m = l.match(/^\s*\[([^\]]+)\]\s*$/);
    if (m) { if (cur) cur.end = i; cur = { name: m[1], kv: {}, start: i, end: lines.length }; out.push(cur); return; }
    const kv = l.match(/^\s*([^#;=][^=]*?)\s*=\s*(.*?)\s*$/);
    if (cur && kv) cur.kv[kv[1].toLowerCase()] = kv[2];
  });
  return out;
}
const removeLines = (txt, a, b) => { const l = txt.split('\n'); l.splice(a, b - a); return l.join('\n'); };
function guessSubnet(net) {
  for (const i of net?.addr || []) for (const a of i.addr_info || []) {
    if (a.family === 'inet' && a.scope === 'global' && !/^(docker|br-|veth|tailscale)/.test(i.ifname)) {
      const ip = a.local.split('.').map(Number).reduce((x, y) => (x << 8) + y, 0) >>> 0;
      const mask = a.prefixlen ? (~0 << (32 - a.prefixlen)) >>> 0 : 0;
      const n = (ip & mask) >>> 0;
      return [24, 16, 8, 0].map((s) => (n >>> s) & 255).join('.') + '/' + a.prefixlen;
    }
  }
  return '*';
}

// ── Users ──
async function pageUsers(v) {
  const d = await api('GET', 'users');
  const users = d.users.filter((u) => u.uid >= 1000);
  setSub('Accounts for SMB, NFS and SSH');
  v.innerHTML = `<div class="card glass"><div class="toolbar"><span class="muted">These are real Linux users. “SMB” grants access to Windows shares.</span><span class="grow"></span>
    <button class="btn sm" data-g>${ic('plus')}Group</button><button class="btn sm primary" data-add>${ic('plus')}User</button></div>
    <div class="rows">${users.map((u, i) => `<div class="row"><div class="ico">${ic('users')}</div><div class="main-col"><div class="t">${esc(u.name)}</div>
      <div class="s">${esc(u.home)} · ${(u.groups || []).map(esc).join(', ') || 'no extra groups'}</div></div>
      <div class="acts">${d.samba ? `<span class="badge ${u.smb ? '' : 'off'}">SMB ${u.smb ? 'on' : 'off'}</span>` : ''}<span class="badge ${/nologin|false/.test(u.shell) ? 'off' : ''}">SSH ${/nologin|false/.test(u.shell) ? 'off' : 'on'}</span>
      <button class="btn sm" data-pw="${i}">${ic('key')}Password</button><button class="btn sm" data-ed="${i}">${ic('edit')}Edit</button><button class="btn sm danger" data-rm="${i}">${ic('trash')}</button></div></div>`).join('') || '<div class="empty">No users yet.</div>'}</div></div>`;
  const groupsBox = (sel = []) => `<div>${d.groups.map((g) => `<label class="check"><input type="checkbox" data-gr="${esc(g)}" ${sel.includes(g) ? 'checked' : ''}>${esc(g)}</label>`).join('')}</div>`;
  $('[data-add]', v).onclick = () => modal({ title: 'Create user', body: `
    <div class="field"><label>Username (lowercase, no spaces)</label><input class="input" data-n autocomplete="off"></div>
    <div class="field"><label>Password</label><input class="input" type="password" data-p autocomplete="new-password"></div>
    ${d.samba ? '<label class="check"><input type="checkbox" data-smb checked>SMB access (Windows shares)</label>' : ''}<label class="check"><input type="checkbox" data-sh>Allow SSH login</label>
    <div class="field" style="margin-top:8px"><label>Groups</label>${groupsBox(['users'])}</div>`,
    buttons: [{ label: 'Cancel' }, { label: 'Create', cls: 'primary', onClick: (m) => api('POST', 'users', { name: $('[data-n]', m).value.trim(), password: $('[data-p]', m).value,
      smb: !!$('[data-smb]', m)?.checked, shell: $('[data-sh]', m).checked, groups: $$('[data-gr]:checked', m).map((x) => x.dataset.gr) }).then(() => { toast('User created', 'ok'); route(); }) }] });
  $('[data-g]', v).onclick = async () => { const n = await promptDlg('New group', 'Name'); if (n) api('POST', 'groups', { name: n.trim() }).then(() => { toast('Group created', 'ok'); route(); }).catch(fail); };
  $$('[data-pw]', v).forEach((b) => (b.onclick = async () => {
    const u = users[+b.dataset.pw];
    const p = await promptDlg('Password for ' + u.name, 'New password', '', 'password');
    if (p) api('PUT', 'users/' + u.name, { password: p }).then(() => toast('Password changed', 'ok')).catch(fail);
  }));
  $$('[data-ed]', v).forEach((b) => (b.onclick = () => {
    const u = users[+b.dataset.ed];
    modal({ title: 'Edit: ' + u.name, body: `${d.samba ? `<label class="check"><input type="checkbox" data-smb ${u.smb ? 'checked' : ''}>SMB access</label>` : ''}
      ${d.samba && !u.smb ? '<div class="field"><label>Password (required to enable SMB)</label><input class="input" type="password" data-p autocomplete="new-password"></div>' : ''}
      <div class="field"><label>Groups</label>${groupsBox(u.groups || [])}</div>`,
      buttons: [{ label: 'Cancel' }, { label: 'Save', cls: 'primary', onClick: (m) => api('PUT', 'users/' + u.name, { groups: $$('[data-gr]:checked', m).map((x) => x.dataset.gr),
        smb: d.samba ? $('[data-smb]', m).checked : undefined, password: $('[data-p]', m)?.value || '' }).then(() => { toast('Saved', 'ok'); route(); }) }] });
  }));
  $$('[data-rm]', v).forEach((b) => (b.onclick = async () => {
    const u = users[+b.dataset.rm];
    if (await confirmDlg(`Delete user ${u.name}? The folder ${u.home} is kept.`, 'Delete')) api('DELETE', 'users/' + u.name).then(() => { toast('Deleted', 'ok'); route(); }).catch(fail);
  }));
}

// ── Network ──
async function pageNetwork(v) {
  const n = await api('GET', 'network');
  const gw = (n.route || []).find((r) => r.dst === 'default');
  const ifs = (n.addr || []).filter((i) => i.ifname !== 'lo');
  setSub(`${ifs.filter((i) => i.operstate === 'UP').length} interfaces up`);
  v.innerHTML = `<div class="grid">
    <div class="card glass c4"><h2>Device</h2><dl class="kv"><dt>Name</dt><dd>${esc(n.hostname)}</dd><dt>Gateway</dt><dd>${esc(gw ? gw.gateway + ' (' + gw.dev + ')' : '–')}</dd><dt>DNS</dt><dd>${(n.dns || []).map(esc).join('<br>') || '–'}</dd></dl>
      <button class="btn sm" style="margin-top:16px" data-hn>${ic('edit')}Change hostname</button></div>
    <div class="card glass c8"><h2>Interfaces</h2><div class="rows">${ifs.map((i) => {
      const sp = n.speed?.[i.ifname];
      return `<div class="row"><div class="ico">${ic('net')}</div><div class="main-col"><div class="t">${esc(i.ifname)} <span class="faint" style="font-weight:400">${esc(i.address || '')}</span></div>
      <div class="s">${(i.addr_info || []).map((a) => esc(a.local + '/' + a.prefixlen)).join(' · ') || 'no address'}${sp && sp > 0 ? ' · ' + (sp >= 1000 ? sp / 1000 + ' Gbit/s' : sp + ' Mbit/s') : ''}${i.mtu ? ' · MTU ' + i.mtu : ''}</div></div>
      <span class="badge ${i.operstate === 'UP' ? '' : 'off'}">${i.operstate === 'UP' ? 'Connected' : esc(i.operstate === 'DOWN' ? 'Disconnected' : i.operstate)}</span></div>`;
    }).join('')}</div>
    <p class="faint" style="margin-top:14px">Addresses come from DHCP or /etc/network/interfaces. Set a fixed address in your router (DHCP reservation) so the NAS can never lock itself out.</p></div></div>`;
  $('[data-hn]', v).onclick = async () => {
    const h = await promptDlg('Hostname', 'New name (letters, digits, hyphen)', n.hostname);
    if (h && h !== n.hostname) api('PUT', 'hostname', { hostname: h }).then(() => { toast('Hostname changed', 'ok'); $('[data-host]').textContent = h; route(); }).catch(fail);
  };
}

// ── Services ──
async function pageServices(v) {
  const list = await api('GET', 'services');
  setSub(`${list.filter((s) => s.ActiveState === 'active').length} of ${list.length} running`);
  v.innerHTML = `<div class="card glass"><div class="rows">${list.map((s) => {
    const on = s.ActiveState === 'active', auto = s.UnitFileState === 'enabled';
    const self = s.name === 'rackpanel.service';
    return `<div class="row"><span class="dot ${on ? '' : s.ActiveState === 'failed' ? 'bad' : 'off'}"></span><div class="main-col"><div class="t">${esc(s.Description || s.name)}</div>
      <div class="s">${esc(s.name)} · ${esc(s.ActiveState)} (${esc(s.SubState)}) · autostart ${auto ? 'on' : s.UnitFileState === 'static' ? 'static' : 'off'}</div></div>
      <div class="acts">${self ? '' : on ? `<button class="btn sm" data-a="stop" data-n="${esc(s.name)}">${ic('stop')}Stop</button>` : `<button class="btn sm" data-a="start" data-n="${esc(s.name)}">${ic('play')}Start</button>`}
      <button class="btn sm" data-a="restart" data-n="${esc(s.name)}">${ic('rotate')}Restart</button>
      ${self || s.UnitFileState === 'static' ? '' : `<button class="btn sm" data-a="${auto ? 'disable' : 'enable'}" data-n="${esc(s.name)}">Autostart ${auto ? 'off' : 'on'}</button>`}
      <button class="btn sm" data-log="${esc(s.name)}">${ic('logs')}Log</button></div></div>`;
  }).join('')}</div></div>`;
  $$('[data-a]', v).forEach((b) => (b.onclick = async () => {
    if (b.dataset.a === 'stop' && b.dataset.n === 'ssh.service' && !(await confirmDlg('Stop SSH? You will only reach the device through this panel or the console.', 'Stop'))) return;
    b.disabled = true;
    api('POST', `services/${b.dataset.n}/${b.dataset.a}`).then(() => { toast('Done', 'ok'); setTimeout(route, 600); }).catch((e) => { fail(e); b.disabled = false; });
  }));
  $$('[data-log]', v).forEach((b) => (b.onclick = () => showLog(b.dataset.log)));
}
async function showLog(unit) {
  const t = await api('GET', 'logs?unit=' + encodeURIComponent(unit)).catch(fail);
  if (t == null) return;
  const m = modal({ title: 'Log · ' + unit, wide: true, body: `<pre class="code">${esc(t || 'No entries')}</pre>` });
  const pre = $('pre', m.el); pre.scrollTop = pre.scrollHeight;
}

// ── Apps ──
async function pageApps(v) {
  const list = await api('GET', 'apps');
  setSub(`${list.filter((a) => a.installed).length} of ${list.length} installed`);
  v.innerHTML = `<div class="grid">${list.map((a) => `<div class="card glass c6"><div class="row" style="border:0;padding:0;background:none">
      <span class="ico">${ic(a.icon || 'grid')}</span><div class="main-col"><div class="t">${esc(a.name)}</div><div class="s">${esc(a.packages.join(' '))}</div></div>
      <span class="badge ${a.installed ? (a.active || !a.service ? '' : 'warn') : 'off'}">${a.installed ? (a.active || !a.service ? 'Running' : 'Stopped') : 'Not installed'}</span></div>
    <p class="muted">${esc(a.desc)}</p>${a.installed && a.note ? `<p class="faint" style="font-size:12.5px">${esc(a.note)}</p>` : ''}
    <div class="toolbar">${!a.installed ? `<button class="btn primary sm" data-app="install" data-id="${esc(a.id)}">${ic('download')}Install</button>` : `
      ${a.port && a.active ? `<a class="btn primary sm" target="_blank" rel="noopener" href="http://${esc(location.hostname)}:${a.port}${esc(a.path || '/')}">${ic('link')}Open</a>` : ''}
      ${a.service ? `<button class="btn sm" data-svc="${a.active ? 'stop' : 'start'}" data-n="${esc(a.service)}">${ic(a.active ? 'stop' : 'play')}${a.active ? 'Stop' : 'Start'}</button><button class="btn sm" data-log="${esc(a.service)}">${ic('logs')}Log</button>` : ''}
      <button class="btn sm danger" data-app="remove" data-id="${esc(a.id)}">${ic('trash')}Remove</button>`}</div></div>`).join('') || '<div class="card glass c12 empty">No app catalog found.</div>'}</div>`;
  $$('[data-app]', v).forEach((b) => (b.onclick = async () => {
    const a = list.find((x) => x.id === b.dataset.id), rm = b.dataset.app === 'remove';
    if (rm && !(await confirmDlg(`Remove ${a.name}? Its settings are kept.`, 'Remove'))) return;
    api('POST', `apps/${a.id}/${b.dataset.app}`).then((id) => runJob(id, (rm ? 'Remove ' : 'Install ') + a.name, () => route())).catch(fail);
  }));
  $$('[data-svc]', v).forEach((b) => (b.onclick = () => { b.disabled = true; api('POST', `services/${b.dataset.n}/${b.dataset.svc}`).then(() => setTimeout(route, 600)).catch((e) => { fail(e); b.disabled = false; }); }));
  $$('[data-log]', v).forEach((b) => (b.onclick = () => showLog(b.dataset.log)));
}

// ── Updates & packages ──
async function pageUpdates(v) {
  setSub('Keep Debian packages up to date');
  const quick = [['samba wsdd', 'SMB shares'], ['nfs-kernel-server', 'NFS exports'], ['smartmontools', 'Disk health (SMART)'], ['hdparm', 'Disk standby'],
    ['rsync', 'Backups'], ['avahi-daemon', 'Visible on the network (.local)'], ['unattended-upgrades', 'Automatic security updates'], ['htop', 'Process monitor']];
  v.innerHTML = `<div class="grid">
    <div class="card glass c7"><h2>System updates</h2><p class="muted" data-st>Not checked yet.</p><div class="toolbar"><button class="btn primary" data-chk>${ic('rotate')}Check for updates</button><button class="btn" data-ins disabled>${ic('download')}Install all</button></div><div data-list></div></div>
    <div class="card glass c5"><h2>Install packages</h2>
      <div class="field"><label for="pk">Package name(s), separated by spaces</label><input class="input" id="pk" placeholder="e.g. rsync"></div><button class="btn" data-pk>Install</button>
      <h2 style="margin-top:24px">Recommended</h2><div class="rows">${quick.map(([p, d]) => `<div class="row"><div class="main-col"><div class="t">${esc(p)}</div><div class="s">${esc(d)}</div></div><button class="btn sm" data-q="${esc(p)}">Install</button></div>`).join('')}</div></div></div>`;
  $('[data-chk]', v).onclick = async (e) => {
    e.target.disabled = true; $('[data-st]', v).textContent = 'Checking … (may take a minute on slow hardware)';
    try {
      const l = await api('POST', 'updates/check');
      $('[data-st]', v).textContent = l.length ? `${l.length} updates available` : 'Everything is up to date.';
      $('[data-ins]', v).disabled = !l.length;
      $('[data-list]', v).innerHTML = l.length ? `<pre class="code" style="max-height:340px">${esc(l.join('\n'))}</pre>` : '';
    } catch (er) { fail(er); $('[data-st]', v).textContent = ''; }
    e.target.disabled = false;
  };
  $('[data-ins]', v).onclick = () => api('POST', 'updates/install').then((id) => runJob(id, 'Install updates')).catch(fail);
  $('[data-pk]', v).onclick = () => { const p = $('#pk', v).value.trim(); if (p) installPkgs(p); };
  $$('[data-q]', v).forEach((b) => (b.onclick = () => installPkgs(b.dataset.q)));
}

// ── Logs ──
async function pageLogs(v) {
  const svc = await api('GET', 'services').catch(() => []);
  setSub('System journal');
  v.innerHTML = `<div class="card glass"><div class="toolbar"><select class="input" style="width:auto" data-u aria-label="Source"><option value="">Everything</option><option value="!k">Kernel</option>${svc.map((s) => `<option>${esc(s.name)}</option>`).join('')}</select>
    <button class="btn sm" data-r>${ic('rotate')}Reload</button></div><pre class="code" style="max-height:70vh" data-o>Loading …</pre></div>`;
  const load = async () => {
    const u = $('[data-u]', v).value;
    const t = await api('GET', 'logs' + (u === '!k' ? '?kernel=1' : u ? '?unit=' + encodeURIComponent(u) : '')).catch((e) => e.message);
    const pre = $('[data-o]', v); pre.textContent = t || 'No entries'; pre.scrollTop = pre.scrollHeight;
  };
  $('[data-u]', v).onchange = load; $('[data-r]', v).onclick = load;
  load();
}

// ── System ──
async function pageSystem(v) {
  const [t, me, keys] = await Promise.all([api('GET', 'time').catch(() => ({ info: {}, zones: [] })), api('GET', 'me'), api('GET', 'keys')]);
  setSub('Time, power and panel');
  const i = t.info;
  v.innerHTML = `<div class="grid">
    <div class="card glass c6"><h2>${ic('clock')}Date & time</h2>
      <dl class="kv"><dt>Now</dt><dd>${esc(i.TimeUSec || new Date().toLocaleString('en-GB'))}</dd><dt>Synchronized</dt><dd>${i.NTPSynchronized === 'yes' ? 'Yes' : 'No'}</dd></dl>
      <div class="field" style="margin-top:14px"><label for="tz">Time zone</label><select class="input" id="tz">${t.zones.map((z) => `<option ${z === i.Timezone ? 'selected' : ''}>${esc(z)}</option>`).join('')}</select></div>
      <label class="check"><input type="checkbox" id="ntp" ${i.NTP === 'yes' ? 'checked' : ''}>Set time automatically (NTP)</label>
      <div><button class="btn primary" data-ts>Apply</button></div></div>
    <div class="card glass c6"><h2>${ic('power')}Power</h2><p class="muted">A restart takes a few minutes depending on the device. Shares are unavailable meanwhile.</p>
      <div class="toolbar"><button class="btn" data-pw="reboot">${ic('rotate')}Restart</button><button class="btn danger" data-pw="poweroff">${ic('power')}Shut down</button></div></div>
    <div class="card glass c6"><h2>${ic('shield')}Panel access</h2><p class="muted">Signed in as <b>${esc(me.username)}</b>. This account is only for RackPanel, not for SMB or SSH.</p>
      <div class="field"><label for="op">Current password</label><input class="input" type="password" id="op" autocomplete="current-password"></div>
      <div class="field"><label for="np">New password</label><input class="input" type="password" id="np" autocomplete="new-password"></div>
      <button class="btn primary" data-cp>Change password</button></div>
    <div class="card glass c6"><h2>${ic('link')}About RackPanel</h2>
      <dl class="kv"><dt>Version</dt><dd>${esc(hostInfo.version || '')}</dd><dt>Device</dt><dd>${esc(hostInfo.model || '–')}</dd><dt>System</dt><dd>${esc(hostInfo.os || '')}</dd></dl>
      <p class="muted">Open source on GitHub: <a href="${REPO}" target="_blank" rel="noopener">${REPO.replace('https://', '')}</a>.<br>Run a Synology RS812 without DSM: <a href="https://github.com/Syntax-XXX/rs812-debian" target="_blank" rel="noopener">rs812-debian</a>.<br>Packages and updates: <a href="https://syntax-xxx.github.io/rsrb/" target="_blank" rel="noopener">RSRB repository</a>.</p></div>
    <div class="card glass c12"><h2>${ic('key')}API keys</h2><p class="muted">For your own dashboards and scripts: send the key as header <code>X-API-Key</code>, e.g. to <code>/api/overview</code>. Guide: <a href="${REPO}/blob/main/docs/API.md" target="_blank" rel="noopener">docs/API.md</a>.</p>
      <div class="rows">${keys.map((k) => `<div class="row"><span class="ico">${ic('key')}</span><div class="main-col"><div class="t">${esc(k.name)}</div><div class="s">${k.read_only ? 'Read only' : 'Full access'} · since ${new Date(k.created).toLocaleDateString('en-GB')}</div></div><button class="btn sm danger" data-kd="${esc(k.id)}">${ic('trash')}Delete</button></div>`).join('') || '<div class="empty">No keys yet.</div>'}</div>
      <div class="toolbar"><button class="btn" data-kn>${ic('plus')}New key</button></div></div>
</div>`;
  $('[data-ts]', v).onclick = () => api('PUT', 'time', { zone: $('#tz', v).value, ntp: $('#ntp', v).checked }).then(() => { toast('Saved', 'ok'); route(); }).catch(fail);
  $$('[data-pw]', v).forEach((b) => (b.onclick = async () => {
    const off = b.dataset.pw === 'poweroff';
    if (!(await confirmDlg(off ? 'Shut down the NAS now? It can only be powered on again at the device.' : 'Restart the NAS now?', off ? 'Shut down' : 'Restart'))) return;
    api('POST', 'power/' + b.dataset.pw).then(() => toast(off ? 'Shutting down …' : 'Restarting …', 'ok')).catch(fail);
  }));
  $('[data-kn]', v).onclick = () => modal({ title: 'New API key', body: '<div class="field"><label for="kn">Name (what is it for?)</label><input class="input" id="kn" placeholder="e.g. Homelab dashboard"></div><label class="check"><input type="checkbox" id="kr" checked>Read only (recommended for dashboards)</label>',
    buttons: [{ label: 'Cancel' }, { label: 'Create', cls: 'primary', onClick: async (m) => {
      const r = await api('POST', 'keys', { name: $('#kn', m).value, read_only: $('#kr', m).checked });
      modal({ title: 'Copy the key now', body: `<p class="muted">It is shown only this once.</p><pre class="code" style="user-select:all">${esc(r.key)}</pre>` });
      route();
    } }] });
  $$('[data-kd]', v).forEach((b) => (b.onclick = async () => { if (await confirmDlg('Delete key? Dashboards using it lose access.', 'Delete')) api('DELETE', 'keys/' + b.dataset.kd).then(() => route()).catch(fail); }));
  $('[data-cp]', v).onclick = () => api('PUT', 'me/password', { old_password: $('#op', v).value, password: $('#np', v).value }).then(() => { toast('Password changed', 'ok'); $('#op', v).value = $('#np', v).value = ''; }).catch(fail);
}

// ── Sign-in ──
function loginView() {
  $('#root').innerHTML = `<div class="login"><form class="glass" autocomplete="on">
    <div class="brand"><span class="brand-mark">${ic('server')}</span><div>RackPanel<small>NAS management</small></div></div>
    <div class="field"><label for="u">User</label><input class="input" id="u" autocomplete="username" required></div>
    <div class="field"><label for="p">Password</label><input class="input" id="p" type="password" autocomplete="current-password" required></div>
    <button class="btn primary" style="width:100%;justify-content:center;padding:11px">Sign in</button>
    <p class="faint" style="text-align:center;margin:16px 0 0;font-size:12px" data-e></p></form></div>`;
  $('#u').value = store.get('rp.user') || '';
  ($('#u').value ? $('#p') : $('#u')).focus();
  $('form').onsubmit = async (e) => {
    e.preventDefault();
    const b = $('form .btn'); b.disabled = true;
    try {
      const d = await api('POST', 'login', { username: $('#u').value.trim(), password: $('#p').value });
      tok = d.token; store.set('rp.tok', JSON.stringify(tok)); store.set('rp.user', d.user.username);
      shell();
    } catch (er) { $('[data-e]').textContent = er.message; $('[data-e]').style.color = 'var(--bad)'; b.disabled = false; }
  };
}
function logout() { tok = null; store.set('rp.tok', null); if (cleanup) { cleanup(); cleanup = null; } loginView(); }

tok ? shell() : loginView();
