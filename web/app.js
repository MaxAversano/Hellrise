'use strict';
// Hellrise · companion per Diablo IV

const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const S = { st: null, page: '', season: 0, cls: localStorage.getItem('hr.cls') || 'barbarian', builds: {}, loot: {}, events: null, ui: {} };

async function api(path, opts) {
  try {
    const r = await fetch(path, opts);
    const ct = r.headers.get('content-type') || '';
    const body = ct.includes('json') ? await r.json() : null;
    if (!r.ok) throw new Error(body?.error || ('HTTP ' + r.status));
    return body;
  } catch (e) {
    if (window.DEMO) { const d = window.DEMO(path, opts); if (d !== undefined) return d; }
    throw e;
  }
}
const post = (path, body) => api(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
function toast(msg) { const t = $('#toast'); t.textContent = msg; t.classList.add('on'); clearTimeout(t._h); t._h = setTimeout(() => t.classList.remove('on'), 2600); }
function openLink(u) { if (window.DEMO) window.open(u, '_blank'); else fetch('/api/open?u=' + encodeURIComponent(u)); }
function openGuide(u) { if (window.DEMO) window.open(u, '_blank'); else fetch('/api/guide?u=' + encodeURIComponent(u)); }
async function copy(text, what) { try { await navigator.clipboard.writeText(text); toast((what || 'Testo') + ' copiato'); } catch { toast('Copia non riuscita'); } }
const lang = () => S.st?.settings?.lang || 'it';
const nm = (it, en) => (lang() === 'en' ? (en || it) : (it || en)) || '';
const sub = (it, en) => (lang() === 'en' ? (it && it !== en ? it : '') : (en && en !== it ? en : ''));

// ---------- icone ----------
const IC = {
  boss: '<svg viewBox="0 0 24 24"><path d="M12 3c-4.4 0-8 3.2-8 7.5 0 2.6 1.3 4.6 3 5.8V19a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2v-2.7c1.7-1.2 3-3.2 3-5.8C20 6.2 16.4 3 12 3z"/><circle cx="9" cy="11" r="1.6"/><circle cx="15" cy="11" r="1.6"/><path d="M10 21v-3M14 21v-3"/></svg>',
  helltide: '<svg viewBox="0 0 24 24"><path d="M12 2c1 4 5 6 5 11a5 5 0 0 1-10 0c0-2 1-3.5 2-4.5 0 2 1 3 2 3.5C10.5 9 11 5 12 2z"/></svg>',
  legion: '<svg viewBox="0 0 24 24"><path d="M12 3l2.5 5 5.5.8-4 3.9.9 5.5L12 15.6 7.1 18.2 8 12.7 4 8.8l5.5-.8z"/></svg>',
  search: '<svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="M20 20l-4-4"/></svg>',
  star: '<svg viewBox="0 0 24 24"><path d="M12 3l2.6 5.4 5.9.8-4.3 4.1 1 5.9L12 16.4 6.8 19.2l1-5.9L3.5 9.2l5.9-.8z"/></svg>',
  ext: '<svg viewBox="0 0 24 24"><path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5"/></svg>',
  copy: '<svg viewBox="0 0 24 24"><rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3"/></svg>',
  sprout: '<svg viewBox="0 0 24 24"><path d="M12 21v-9M12 12C12 7 8 5 4 5c0 5 3 7 8 7zM12 14c0-4 3-6 8-6 0 4-3 6-8 6z"/></svg>',
  keyb: '<svg viewBox="0 0 24 24"><rect x="2.5" y="6" width="19" height="12" rx="2.5"/><path d="M7 14h10"/></svg>',
  warn: '<svg viewBox="0 0 24 24"><path d="M12 3l9.5 17h-19z"/><path d="M12 10v4M12 17.5v.01"/></svg>',
  refresh: '<svg viewBox="0 0 24 24"><path d="M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7"/></svg>',
  play: '<svg viewBox="0 0 24 24"><path d="M7 4l13 8-13 8z"/></svg>',
  plus: '<svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>',
  trash: '<svg viewBox="0 0 24 24"><path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3"/></svg>',
  doc: '<svg viewBox="0 0 24 24"><path d="M6 3h9l4 4v14H6z"/><path d="M14 3v5h5"/></svg>',
  chat: '<svg viewBox="0 0 24 24"><path d="M4 5h16v11H9l-5 4z"/></svg>',
  pin: '<svg viewBox="0 0 24 24"><path d="M12 21s7-6.2 7-12a7 7 0 0 0-14 0c0 5.8 7 12 7 12z"/><circle cx="12" cy="9" r="2.5"/></svg>',
};
const CLASSES = [
  ['barbarian', 'Barbarian', '<svg viewBox="0 0 24 24"><path d="M14 3l7 7-3 1-2-2-9 9-3-3 9-9-2-2z"/><path d="M5 19l-2 2"/></svg>'],
  ['druid', 'Druid', '<svg viewBox="0 0 24 24"><path d="M12 21c-5 0-8-3.5-8-8 0-4 3-8 8-10 5 2 8 6 8 10 0 4.5-3 8-8 8z"/><path d="M12 21V9M12 13l-3-3M12 16l3-3"/></svg>'],
  ['necromancer', 'Necromancer', '<svg viewBox="0 0 24 24"><path d="M12 3a7 7 0 0 0-7 7c0 2.4 1.2 4.3 3 5.5V19h8v-3.5c1.8-1.2 3-3.1 3-5.5a7 7 0 0 0-7-7z"/><circle cx="9.5" cy="10.5" r="1.4"/><circle cx="14.5" cy="10.5" r="1.4"/><path d="M10 19v2M14 19v2"/></svg>'],
  ['rogue', 'Rogue', '<svg viewBox="0 0 24 24"><path d="M5 19L17 7l2-4-4 2L3 17zM14 6l4 4M7 15l2 2"/></svg>'],
  ['sorcerer', 'Sorcerer', '<svg viewBox="0 0 24 24"><circle cx="12" cy="11" r="5"/><path d="M12 2v2M12 18v4M3 11h2M19 11h2M5.6 4.6L7 6M17 16l1.4 1.4M18.4 4.6L17 6"/></svg>'],
  ['spiritborn', 'Spiritborn', '<svg viewBox="0 0 24 24"><path d="M20 4C10 4 5 9 4 20c5-1 9-3 11-7-2 1-4 1-6 1 3-1 6-3 7-5-1 .5-3 .7-4 .5 3-1 6-3 8-5.5z"/></svg>'],
  ['paladin', 'Paladin', '<svg viewBox="0 0 24 24"><path d="M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z"/><path d="M12 7v10M8.5 11h7"/></svg>'],
  ['warlock', 'Warlock', '<svg viewBox="0 0 24 24"><path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3"/></svg>'],
];
const clsName = c => { const it = S.st?.classesIt?.[c]; const en = (CLASSES.find(x => x[0] === c) || [, c])[1]; return nm(it, en); };
const clsIcon = c => (CLASSES.find(x => x[0] === c) || [])[2] || '';
const tierCls = t => 'tier tier-' + (t === 'S+' ? 'Sp' : (/^[SABCDF]$/.test(t) ? t : 'q'));
const tierBadge = t => t ? (t === '?' ? '<span class="tier tier-q" title="Tier list di Rob2628 ancora senza voti (WIP)" style="font-size:9.5px">WIP</span>' : `<span class="${tierCls(t)}" title="Tier list di Rob2628">${esc(t)}</span>`) : '<span class="notier" title="Non presente nella tier list di Rob"></span>';

// ---------- tempo ----------
function fmtLeft(s) {
  s = Math.max(0, Math.round(s));
  const h = Math.floor(s / 3600), m = Math.floor(s % 3600 / 60), x = s % 60;
  if (h > 0) return `${h}h ${String(m).padStart(2, '0')}m`;
  return `${m}:${String(x).padStart(2, '0')}`;
}
const hhmm = t => new Date(t * 1000).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' });
const dayLbl = t => { const d = new Date(t * 1000), n = new Date(); const tm = new Date(n); tm.setDate(n.getDate() + 1); if (d.toDateString() === n.toDateString()) return 'oggi'; if (d.toDateString() === tm.toDateString()) return 'domani'; return d.toLocaleDateString('it-IT', { weekday: 'short', day: 'numeric' }); };
const dateLbl = t => t ? new Date(t * 1000).toLocaleDateString('it-IT', { day: 'numeric', month: 'short', year: 'numeric' }) : '';
const nowS = () => Date.now() / 1000 + (S.clockSkew || 0);
const zonesTxt = z => (z || []).map(x => x.name).join(' · ');
function evState(e) { const n = nowS(); return n >= e.start && n < e.end ? 'live' : (n < e.start ? 'soon' : 'past'); }

// ---------- avvio ----------
async function boot() {
  embers();
  try { S.st = await api('/api/state'); } catch { S.st = { settings: {}, seasons: [] }; }
  applySettings();
  S.season = S.st.season || 0;
  renderSeasonPick();
  await loadEvents();
  setInterval(loadEvents, 60000);
  setInterval(tick, 1000);
  setInterval(poll, 5000);
  addEventListener('hashchange', route);
  route();
}
async function poll() {
  try {
    const st = await api('/api/state?s=' + S.season);
    const seasonsChanged = JSON.stringify(st.seasons) !== JSON.stringify(S.st.seasons);
    const namesArrived = !S.st.classesIt && st.classesIt;
    const dataArrived = (S.st.remote?.at || 0) !== (st.remote?.at || 0);
    S.st = st;
    if (seasonsChanged || dataArrived) { S.builds = {}; S.loot = {}; renderSeasonPick(); }
    if ((seasonsChanged || namesArrived) && !document.querySelector('input:focus,textarea:focus')) route();
    updBar(st.update);
  } catch { }
}
function applySettings() {
  const s = S.st.settings || {};
  document.body.classList.toggle('frameless', !!S.st.frameless);
  document.body.classList.toggle('noblur', s.blur === false);
  document.body.classList.toggle('nomotion', s.motion === false);
  document.body.className = document.body.className.replace(/\bacc-\w+/g, '').trim();
  if (s.accent && s.accent !== 'ember') document.body.classList.add('acc-' + s.accent);
}
function embers() {
  const box = $('#embers'); if (!box) return;
  for (let i = 0; i < 26; i++) {
    const b = document.createElement('b');
    b.style.left = Math.random() * 100 + '%';
    b.style.setProperty('--dx', (Math.random() * 160 - 80) + 'px');
    b.style.animationDuration = (9 + Math.random() * 14) + 's';
    b.style.animationDelay = (-Math.random() * 20) + 's';
    const sz = 1.5 + Math.random() * 2.5; b.style.width = b.style.height = sz + 'px';
    box.appendChild(b);
  }
}
function renderSeasonPick() {
  const el = $('#seasonpick');
  const list = S.st.seasons || [];
  if (!list.length) { el.innerHTML = `<button class="on">Nessuna stagione</button>`; return; }
  if (!list.find(x => x.n === S.season)) S.season = list[0].n;
  el.innerHTML = list.map(s => `<button data-s="${s.n}" class="${s.n === S.season ? 'on' : ''}">Stagione ${s.n}${s.ptr ? '<span class="tag ptr">PTR</span>' : (s.live ? '<span class="tag">LIVE</span>' : '')}</button>`).join('');
  $$('button[data-s]', el).forEach(b => b.onclick = () => { S.season = +b.dataset.s; renderSeasonPick(); route(); });
}
function route() {
  const p = (location.hash.replace(/^#\/?/, '').split('/')[0]) || 'home';
  S.page = p;
  $$('#side a').forEach(a => a.classList.toggle('on', a.dataset.p === p));
  const fn = PAGES[p] || PAGES.home;
  const main = $('#main');
  main.scrollTop = 0;
  fn(main);
}

// ---------- eventi ----------
async function loadEvents() {
  try { S.events = await api('/api/events'); if (S.events?.now) S.clockSkew = S.events.now - Date.now() / 1000; } catch { }
  renderNextChips();
}
function nextOf(kind) { return (S.events?.events || []).find(e => e.kind === kind && evState(e) !== 'past'); }
function renderNextChips() {
  const el = $('#nextev'); if (!S.events) { el.innerHTML = ''; return; }
  el.innerHTML = ['boss', 'helltide', 'legion'].map(k => { const e = nextOf(k); if (!e) return ''; const live = evState(e) === 'live'; return `<div class="chipev ${live ? 'live' : ''}" onclick="location.hash='#/eventi'" title="${esc(e.title)}${e.zones ? ' · ' + esc(zonesTxt(e.zones)) : ''}"><span class="ic k-${k}">${IC[k]}</span>${k === 'boss' ? esc(e.guess ? 'Boss' : e.title) : (k === 'helltide' ? 'Marea' : 'Legione')} <b data-cd="${e.start}" data-end="${e.end}">${live ? 'in corso' : fmtLeft(e.start - nowS())}</b></div>`; }).join('');
}
function tick() {
  $$('[data-cd]').forEach(el => {
    const st = +el.dataset.cd, end = +el.dataset.end || st, n = nowS();
    el.textContent = n >= st && n < end ? (el.dataset.livefmt === 'left' ? 'termina tra ' + fmtLeft(end - n) : 'in corso') : fmtLeft(st - n);
  });
  $$('[data-bar]').forEach(el => { const st = +el.dataset.bar, end = +el.dataset.end, n = nowS(); const p = n >= st ? Math.min(100, (n - st) / (end - st) * 100) : 0; el.style.width = p + '%'; });
  if ((S.events?.events || []).some(e => e.end < nowS() - 2 && !e._gone)) { S.events.events.forEach(e => { if (e.end < nowS()) e._gone = true; }); S.events.events = S.events.events.filter(e => !e._gone); renderNextChips(); if (S.page === 'eventi' || S.page === 'home') route(); }
}

// ---------- dati ----------
async function getBuilds(season) {
  if (!S.builds[season]) S.builds[season] = api('/api/builds?s=' + season).catch(e => { delete S.builds[season]; throw e; });
  return S.builds[season];
}
async function getLoot(season) {
  if (!S.loot[season]) S.loot[season] = api('/api/loot?s=' + season).catch(e => { delete S.loot[season]; throw e; });
  return S.loot[season];
}
const plannerCache = {};
function getPlanner(id) { if (!plannerCache[id]) plannerCache[id] = api('/api/build?id=' + id).catch(e => { delete plannerCache[id]; throw e; }); return plannerCache[id]; }
function noData(msg) { return `<div class="glass empty">${IC.warn}<b>${esc(msg)}</b><span>Controlla il server in <a class="link" href="#/impostazioni">Impostazioni</a>.</span></div>`; }

// =====================================================================
const PAGES = {};

PAGES.home = async main => {
  const evs = S.events?.events || [];
  const card = (k, label) => {
    const e = evs.find(x => x.kind === k && evState(x) !== 'past');
    if (!e) return `<div class="glass evcard"><div class="ic k-${k}">${IC[k]}</div><h3>${label}</h3><div class="big">—</div></div>`;
    const live = evState(e) === 'live';
    const where = k === 'boss' ? (e.zones?.length ? `<b>${esc(e.title)}</b> · ${esc(zonesTxt(e.zones))}` : `<b>${esc(e.title)}</b>`) : (k === 'helltide' ? (live ? 'Attiva ora in una regione' : 'Prossima marea') : (live ? 'Raduno in corso' : 'Prossimo raduno'));
    return `<div class="glass evcard ${live ? 'live' : ''}" onclick="location.hash='#/eventi'"><i class="glow k-${k}"></i><div class="ic k-${k}">${IC[k]}</div><h3>${label} · ${live ? 'in corso' : dayLbl(e.start) + ' alle ' + hhmm(e.start)}</h3><div class="big" data-cd="${e.start}" data-end="${e.end}" data-livefmt="left">${live ? 'termina tra ' + fmtLeft(e.end - nowS()) : fmtLeft(e.start - nowS())}</div><div class="where">${where}${e.guess ? ' <span class="muted">(stimato)</span>' : ''}</div>${live ? `<div class="bar"><i data-bar="${e.start}" data-end="${e.end}"></i></div>` : ''}</div>`;
  };
  main.innerHTML = `<div class="page">
    <h1 class="pt">Bentornato nel Santuario</h1><p class="sub">Stagione ${S.season || '—'} · ${esc(seasonLabel())}</p>
    <div class="hero">${card('boss', 'Boss del mondo')}${card('helltide', 'Marea infernale')}${card('legion', 'Legione')}</div>
    <div class="home2">
      <div class="glass card"><h2>Le migliori build ora <span class="count">tier list di Rob2628</span><span class="sp"></span><a class="link" href="#/build" style="font-size:12.5px">Tutte le build →</a></h2><div class="mini" id="topb"><div class="skel" style="height:220px"></div></div></div>
      <div class="grid" style="align-content:start">
        <div class="glass card"><h2>Leveling consigliato <span class="count">${esc(clsName(S.cls))}</span></h2><div class="mini" id="topl"><div class="skel" style="height:60px"></div></div></div>
        <div class="glass card"><h2>Ultima patch</h2><div id="lastp"><div class="skel" style="height:60px"></div></div></div>
        <div class="glass card"><h2>Stato dei dati</h2><div id="dstat"></div></div>
      </div>
    </div></div>`;
  renderDataStatus($('#dstat'));
  try {
    const b = await getBuilds(S.season);
    const rob = b.rob?.tier || [];
    $('#topb').innerHTML = rob.length ? rob.slice(0, 8).map(r => `<div class="mrow" data-cls="${r.cls}">${tierBadge(r.tier)}<div class="t"><b>${esc(r.build)}</b><small>${esc(clsName(r.cls))}${r.info ? ' · ' + esc(r.info) : ''}</small></div><span class="ci" style="color:var(--mute)">${clsIcon(r.cls).replace('<svg', '<svg width="18" height="18"')}</span></div>`).join('') : '<div class="muted">Tier list non ancora disponibile per questa stagione.</div>';
    $$('#topb .mrow').forEach(r => r.onclick = () => { setCls(r.dataset.cls); location.hash = '#/build'; });
    const lv = levelingPick(b, S.cls);
    $('#topl').innerHTML = lv ? `<div class="mrow" onclick="location.hash='#/build'">${tierBadge(lv.tier)}<div class="t"><b>${esc(lv.title)}</b><small>${esc(lv.info || lv.src)}</small></div></div>` : '<div class="muted">Nessuna build leveling per questa classe.</div>';
  } catch (e) { $('#topb').innerHTML = noData(e.message); $('#topl').innerHTML = ''; }
  try {
    const p = await api('/api/patches');
    const a = p.articles?.[0], x = a?.patches?.[0];
    $('#lastp').innerHTML = x ? `<div class="mrow" onclick="location.hash='#/patch'"><span class="badge">${esc(x.version || 'Patch')}</span><div class="t"><b>${esc(a.title)}</b><small>${esc(x.title)}</small></div></div>` : '<div class="muted">Nessuna patch.</div>';
  } catch { $('#lastp').innerHTML = '<div class="muted">Note delle patch non ancora scaricate.</div>'; }
};
function seasonLabel() { const s = (S.st.seasons || []).find(x => x.n === S.season); return s ? (s.ptr ? 'dati del PTR, in arrivo' : 'stagione in corso') : 'in attesa dei dati'; }
function renderDataStatus(el) {
  const r = S.st.remote || {};
  const ok = r.configured && !r.error;
  const errs = Object.entries(r.errors || {});
  el.innerHTML = `<div class="statusline"><span class="dot ${ok ? (errs.length ? 'warn' : 'ok') : 'bad'}"></span>${ok ? 'Server aversano.design collegato' : (r.configured ? 'Server non raggiungibile' : 'Server non configurato')}${r.at ? ` · aggiornato ${new Date(r.at * 1000).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })}` : ''}</div>
    ${r.error ? `<div class="muted" style="margin-top:6px;font-size:12px">${esc(r.error)}</div>` : ''}
    ${errs.length ? `<div class="muted" style="margin-top:8px;font-size:12px">${errs.map(([k, v]) => `<div>⚠ ${esc(k)}: ${esc(v)}</div>`).join('')}</div>` : ''}
    ${S.st.gameData ? `<div class="statusline" style="margin-top:8px"><span class="dot warn"></span>${esc(S.st.gameData)}</div>` : ''}`;
}
function setCls(c) { S.cls = c; localStorage.setItem('hr.cls', c); }
function levelingPick(b, cls) {
  const rob = (b.rob?.leveling || []).filter(r => r.cls === cls);
  if (rob.length) { const r = rob[0]; const m = b.builds.find(x => x.cls === cls && /level/i.test(x.type) && x.src === 'Maxroll'); return { title: r.build, tier: r.tier, info: r.info, url: r.url, src: 'Rob2628', maxroll: m }; }
  const m = b.builds.find(x => x.cls === cls && /level/i.test(x.type));
  return m ? { title: m.title, tier: m.tier, info: m.src, url: m.url, src: m.src, maxroll: m.src === 'Maxroll' ? m : null } : null;
}

// ---------------------------------------------------------------- build
PAGES.build = async main => {
  const ui = S.ui.build ||= { type: 'Endgame', src: 'all', sel: null, variant: 0 };
  main.innerHTML = `<div class="page">
    <div class="hdr"><div><h1 class="pt">Build</h1><p class="sub" style="margin:0">Maxroll, D4Builds e Mobalytics, con la tier list di Rob2628 · stagione ${S.season || '—'}</p></div></div>
    <div class="classes">${CLASSES.map(([c]) => `<div class="cls ${c === S.cls ? 'on' : ''}" data-c="${c}"><span class="ci">${clsIcon(c)}</span>${esc(clsName(c))}</div>`).join('')}</div>
    <div id="lvb"></div>
    <div class="bsplit">
      <div class="glass" style="padding:0">
        <div class="row wrap" style="padding:12px 12px 0;gap:8px">
          <div class="seg" id="btype">${['Endgame', 'Leveling', 'Tutte'].map(t => `<button data-v="${t}" class="${ui.type === t ? 'on' : ''}">${t}</button>`).join('')}</div>
          <div class="seg" id="bsrc">${[['all', 'Tutte'], ['Maxroll', 'Maxroll'], ['D4Builds', 'D4Builds'], ['Mobalytics', 'Moba']].map(([v, l]) => `<button data-v="${v}" class="${ui.src === v ? 'on' : ''}">${l}</button>`).join('')}</div>
        </div>
        <div class="blist" id="blist"><div class="skel" style="height:300px"></div></div>
      </div>
      <div class="glass bdetail" id="bdet"><div class="empty">${IC.doc}<span>Scegli una build dall'elenco</span></div></div>
    </div></div>`;
  $$('.cls', main).forEach(el => el.onclick = () => { setCls(el.dataset.c); ui.sel = null; PAGES.build(main); });
  $$('#btype button').forEach(b => b.onclick = () => { ui.type = b.dataset.v; ui.sel = null; PAGES.build(main); });
  $$('#bsrc button').forEach(b => b.onclick = () => { ui.src = b.dataset.v; ui.sel = null; PAGES.build(main); });
  let data;
  try { data = await getBuilds(S.season); } catch (e) { $('#blist').innerHTML = noData(e.message); return; }
  const lv = levelingPick(data, S.cls);
  $('#lvb').innerHTML = lv ? `<div class="glass lvbanner"><span class="ic">${IC.sprout}</span><div style="flex:1"><div class="lbl">Leveling consigliato per ${esc(clsName(S.cls))}</div><div style="font:600 17px/1.3 var(--serif);margin-top:3px">${esc(lv.title)} ${lv.tier ? tierBadge(lv.tier) : ''}</div><div class="muted" style="font-size:12.5px">${esc(lv.info || '')}${lv.src === 'Rob2628' ? ' · dalla tier list leveling di Rob2628' : ''}</div></div>${lv.maxroll ? `<button class="btn" id="lvopen">Apri su Hellrise</button>` : ''}${lv.url ? `<button class="btn ghost" id="lvlink">${IC.ext} Guida</button>` : ''}</div>` : '';
  if (lv?.url) $('#lvlink').onclick = () => openGuide(lv.url);
  if (lv?.maxroll) $('#lvopen').onclick = () => { ui.type = 'Tutte'; ui.sel = lv.maxroll.url; ui.variantName = 'level'; PAGES.build(main); };
  ui.all = data.builds;
  let list = data.builds.filter(b => b.cls === S.cls);
  if (ui.type !== 'Tutte') list = list.filter(b => ui.type === 'Leveling' ? /level/i.test(b.type) : !/level/i.test(b.type));
  if (ui.src !== 'all') list = list.filter(b => b.src === ui.src);
  const tv = { 'S+': 0, S: 1, A: 2, B: 3, C: 4, D: 5 };
  list.sort((a, b) => ((tv[a.tier] ?? 9) - (tv[b.tier] ?? 9)) || ((a.src === 'Maxroll' ? 0 : 1) - (b.src === 'Maxroll' ? 0 : 1)) || ((b.likes || 0) - (a.likes || 0)) || ((b.updated || 0) - (a.updated || 0)));
  if (!list.length) { $('#blist').innerHTML = `<div class="empty">${IC.search}<span>Nessuna build con questi filtri</span></div>`; return; }
  $('#blist').innerHTML = list.map((b, i) => `<div class="bitem ${ui.sel === b.url ? 'on' : ''}" data-i="${i}">${tierBadge(b.tier)}<div class="t"><b>${esc(b.title)}</b><div class="meta"><span class="badge src-${b.src}">${b.src}</span><span class="badge ${/level/i.test(b.type) ? 'lv' : 'eg'}">${esc(b.type)}</span>${b.author ? `<span>${esc(b.author)}</span>` : ''}${b.likes ? `<span>♥ ${b.likes}</span>` : ''}</div></div></div>`).join('');
  $$('#blist .bitem').forEach(el => el.onclick = () => { $$('#blist .bitem').forEach(x => x.classList.remove('on')); el.classList.add('on'); ui.sel = list[+el.dataset.i].url; ui.variant = 0; showBuild(list[+el.dataset.i]); });
  const cur = list.find(b => b.url === ui.sel) || list[0];
  ui.sel = cur.url;
  $$('#blist .bitem')[list.indexOf(cur)]?.classList.add('on');
  showBuild(cur);
};

const SLOT_KEYS = ['LMB', 'RMB', '1', '2', '3', '4'];
const STOPW = new Set(['barb', 'barbarian', 'druid', 'necro', 'necromancer', 'rogue', 'sorc', 'sorcerer', 'spiritborn', 'paladin', 'pala', 'warlock', 'build', 'guide', 'endgame', 'leveling', 'the', 'of', 'and', 'season', 'starter', 'pushing', 'speedfarm']);
const kw = t => String(t || '').toLowerCase().match(/[a-z0-9]+/g)?.filter(w => w.length > 2 && !STOPW.has(w)) || [];
function similarMaxroll(b) {
  const words = new Set(kw(b.title));
  if (!words.size) return null;
  let best = null, bestN = 0;
  for (const x of S.ui.build.all || []) {
    if (x.src !== 'Maxroll' || !x.profile || x.cls !== b.cls) continue;
    if (/level/i.test(x.type) !== /level/i.test(b.type) && /level/i.test(b.type)) continue;
    const n = kw(x.title).filter(w => words.has(w)).length;
    if (n > bestN) { best = x; bestN = n; }
  }
  return best;
}
async function showBuild(b) {
  const el = $('#bdet'), ui = S.ui.build;
  const head = `<div class="row" style="align-items:flex-start">${tierBadge(b.tier)}<div style="flex:1"><h2 class="title">${esc(b.title)}</h2><div class="row wrap" style="gap:6px;margin-top:6px"><span class="badge src-${b.src}">${b.src}</span><span class="badge ${/level/i.test(b.type) ? 'lv' : 'eg'}">${esc(b.type)}</span>${b.author ? `<span class="muted" style="font-size:12.5px">di ${esc(b.author)}</span>` : ''}${b.updated ? `<span class="muted" style="font-size:12.5px">· aggiornata ${dateLbl(b.updated)}</span>` : ''}</div></div><button class="btn" id="bopen">${IC.ext} Apri guida</button></div>
    ${b.tier || b.info ? `<div class="robinfo"><b>Rob2628:</b> ${b.tier ? `tier ${esc(b.tier)}${b.tierNote ? ' (' + esc(b.tierNote) + ')' : ''}` : ''}${b.info ? ' · ' + esc(b.info) : ''}</div>` : ''}`;
  const bindOpen = () => { const o = $('#bopen'); if (o) o.onclick = () => openGuide(b.url); };
  let p = null, note = '';
  el.innerHTML = head + `<div class="skel" style="height:260px;margin-top:16px"></div>`;
  bindOpen();
  try {
    if (b.profile) p = await getPlanner(b.profile);
    else if (b.src === 'Mobalytics') p = await (plannerCache['m:' + b.url] ||= api('/api/mobabuild?u=' + encodeURIComponent(b.url)).catch(e => { delete plannerCache['m:' + b.url]; throw e; }));
    else {
      const sim = similarMaxroll(b);
      if (sim) { p = await getPlanner(sim.profile); note = `D4Builds tiene i dettagli delle build in un archivio non leggibile dall'app: qui sotto vedi la guida Maxroll più simile, <b>${esc(sim.title)}</b>. Per la versione esatta usa “Apri guida”.`; }
    }
  } catch (e) { note = 'Dettagli non disponibili: ' + esc(e.message); }
  if (!p || !p.variants?.length) {
    el.innerHTML = head + `${b.desc ? `<p class="muted">${esc(b.desc)}</p>` : ''}<div class="empty" style="padding:30px">${IC.doc}<span>${note || (b.src === 'D4Builds' ? 'D4Builds tiene i dettagli delle build in un archivio non leggibile dall\'app e non ho trovato una guida Maxroll simile.' : 'Dettagli non disponibili.')}<br>“Apri guida” la apre in una finestra di Hellrise.</span></div>`;
    bindOpen();
    return;
  }
  if (ui.variantName) { const i = p.variants.findIndex(v => v.name.toLowerCase().includes(ui.variantName)); if (i >= 0) ui.variant = i; ui.variantName = null; }
  if (!p.variants[ui.variant]) ui.variant = 0;
  const v = p.variants[ui.variant] || { skills: [], gear: [] };
  const filters = p.filters || [];
  el.innerHTML = head + (note ? `<div class="robinfo" style="background:rgba(120,160,255,.07);border-color:rgba(120,160,255,.2)">${note}</div>` : '') + `
    ${p.variants.length > 1 ? `<div class="vtabs seg">${p.variants.map((x, i) => `<button data-v="${i}" class="${i === ui.variant ? 'on' : ''}">${esc(x.name)}</button>`).join('')}</div>` : '<div style="height:16px"></div>'}
    ${v.skills?.length ? `<div class="lbl" style="margin-bottom:8px">Barra delle abilità</div>
    <div class="skills">${v.skills.map((s, i) => `<div class="skill"><span class="key">${SLOT_KEYS[i] || ''}</span><b>${esc(nm(s.name, s.en))}</b></div>`).join('')}</div>` : ''}
    <div class="lbl" style="margin-bottom:8px">Equipaggiamento</div>
    <div class="gear">${(v.gear || []).map(g => `<div class="gi ${g.kind}"><span class="ty">${esc(nm(g.type, g.typeEn) || 'Oggetto')}</span><b>${esc(nm(g.name, g.en))}</b>${g.aspect ? `<small>${esc(nm(g.aspect, g.aspectEn))}</small>` : ''}${sub(g.name, g.en) ? `<small class="muted">${esc(sub(g.name, g.en))}</small>` : ''}${g.sockets?.length ? `<div class="sock">${g.sockets.map(s => { const [i, e] = s.split('|'); return `<span>${esc(nm(i, e))}</span>`; }).join('')}</div>` : ''}</div>`).join('')}</div>
    ${filters.length ? `<div class="lbl" style="margin:20px 0 8px">Filtri loot · da importare in gioco</div><div class="grid" style="gap:8px">${filters.map((f, i) => `<div class="filter"><b>${esc(f.name)}</b><button class="btn sm" data-f="${i}">${IC.copy} Copia codice</button></div>`).join('')}</div>` : ''}
    ${v.paragon?.length ? `<div class="row" style="margin:22px 0 4px"><div class="lbl">Paragon</div><span class="sp"></span><div class="seg" id="pstep">${v.paragon.map((st, i) => `<button data-s="${i}" class="${i === (ui.pstep || 0) ? 'on' : ''}">${esc(st.name || 'Fase ' + (i + 1))}</button>`).join('')}</div></div>${PLEGEND}<div class="pboards" id="pboards"></div>` : ''}
    <div class="row" style="margin-top:20px;gap:8px;flex-wrap:wrap">${v.skills?.length ? `<button class="btn pri" id="mkmacro">${IC.keyb} Suggerisci profilo macro</button>` : ''}${v.paragon?.length ? `<button class="btn" id="toov">Paragon nell'overlay in gioco</button>` : ''}${p.id ? `<button class="btn ghost" id="planner">${IC.ext} Planner Maxroll</button>` : ''}</div>`;
  if (v.paragon?.length) {
    if (!v.paragon[ui.pstep || 0]) ui.pstep = 0;
    const drawSteps = () => { const st = v.paragon[ui.pstep || 0]; $('#pboards').innerHTML = st.boards.map((bd, i) => boardCard(bd, i)).join(''); $$('#pboards canvas').forEach((c, i) => drawBoard(c, st.boards[i])); };
    drawSteps();
    $$('#pstep button', el).forEach(x => x.onclick = () => { ui.pstep = +x.dataset.s; $$('#pstep button').forEach(y => y.classList.toggle('on', y === x)); drawSteps(); });
    $('#toov').onclick = async () => { const o = { ...(S.st.settings.overlay || {}), planner: p.id, title: b.title, variant: ui.variant, step: ui.pstep || 0, board: 0, enabled: true, paragon: { ...(S.st.settings.overlay?.paragon || {}), on: true } }; try { S.st.settings = await post('/api/settings', { overlay: o }); toast('Paragon impostato nell\'overlay'); } catch (e) { toast(e.message); } };
  }
  $$('.vtabs button', el).forEach(x => x.onclick = () => { ui.variant = +x.dataset.v; showBuild(b); });
  $$('[data-f]', el).forEach(x => x.onclick = () => copy(filters[+x.dataset.f].code, 'Codice del filtro'));
  bindOpen();
  $('#planner')?.addEventListener('click', () => openGuide(p.url));
  $('#mkmacro')?.addEventListener('click', () => { S.ui.macroDraft = draftFromBuild(b, p, v); location.hash = '#/macro'; });
}

// ---------------------------------------------------------------- paragon
const PCOL = { 1: ['#2a2422', '#d8cfc8'], 2: ['#1c2a44', '#4f8bff'], 3: ['#3a321a', '#ffd24a'], 4: ['#40200f', '#ff7a2a'], 5: ['#333', '#fff'], 6: ['#2c1a40', '#c07bff'], 7: ['#1a3a2a', '#7be0a5'] };
const PLEGEND = `<div class="plegend">${[['Normale', 1], ['Magica', 2], ['Rara', 3], ['Leggendaria', 4], ['Glifo', 6], ['Porta', 5]].map(([l, k]) => `<span><i style="background:${PCOL[k][1]}"></i>${l}</span>`).join('')}<span><i style="background:${PCOL[1][0]};border:1px solid #555"></i>non presa</span></div>`;
function boardCard(bd, i) {
  const leg = (bd.legend || []).map(x => { const [a, e] = x.split('|'); return nm(a, e); })[0];
  return `<div class="pboard" title="Ruota la tavola come indicato prima di posizionarla"><div class="h"><span class="n">${i + 1}</span><b>${esc(nm(bd.name, bd.en))}</b></div><canvas width="${bd.width * 8}" height="${bd.width * 8}"></canvas><small>${bd.rotation ? `Rotazione Maxroll: ${bd.rotation} ${bd.rotation === 1 ? 'scatto' : 'scatti'}` : 'Nessuna rotazione'} · ${bd.points} punti</small>${bd.glyph ? `<small>Glifo: <b style="color:var(--ink)">${esc(nm(bd.glyph, bd.glyphEn))}</b>${bd.glyphLvl ? ' liv. ' + bd.glyphLvl : ''}</small>` : ''}${leg ? `<small style="color:#ff9a52">${esc(leg)}</small>` : ''}</div>`;
}
function drawBoard(c, bd) {
  const g = c.getContext('2d'), w = bd.width, s = c.width / w;
  g.clearRect(0, 0, c.width, c.height);
  for (let i = 0; i < bd.cells.length; i++) {
    const t = bd.cells[i]; if (t === '0') continue;
    const sel = bd.sel[i] === '1', cc = PCOL[t] || PCOL[1];
    g.fillStyle = sel ? cc[1] : cc[0];
    const x = (i % w) * s, y = Math.floor(i / w) * s;
    g.fillRect(x + .5, y + .5, s - 1, s - 1);
  }
}

// ---------------------------------------------------------------- chi droppa cosa
PAGES.drop = async main => {
  const ui = S.ui.drop ||= { q: '', cls: '', boss: '', fav: false, cat: '' };
  main.innerHTML = `<div class="page">
    <div class="hdr"><div><h1 class="pt">Chi droppa cosa</h1><p class="sub" style="margin:0" id="dsrc">Boss da farmare per ogni unico · stagione ${S.season || '—'}</p></div></div>
    <div class="glass bigsearch">${IC.search}<input id="dq" placeholder="Cerca un unico, un mitico o un materiale (italiano o inglese)…" value="${esc(ui.q)}"><kbd>Esc</kbd></div>
    <div class="row wrap" style="gap:8px;margin-bottom:14px">
      <div class="seg" id="dcls"><button data-v="" class="${!ui.cls ? 'on' : ''}">Tutte le classi</button>${CLASSES.map(([c]) => `<button data-v="${c}" class="${ui.cls === c ? 'on' : ''}" title="${esc(clsName(c))}">${clsIcon(c).replace('<svg', '<svg width="16" height="16"')}</button>`).join('')}</div>
      <div class="seg" id="dcat" style="flex-wrap:wrap">${[['', 'Tutti'], ...ITEMCATS].map(([v, l]) => `<button data-v="${v}" class="${(ui.cat || '') === v ? 'on' : ''}">${l}</button>`).join('')}</div>
      <label class="tog" style="padding:0 6px"><input type="checkbox" id="dfav" ${ui.fav ? 'checked' : ''}><i></i><b style="font-size:13px">Solo seguiti</b></label>
      <span class="sp"></span><span class="muted" id="dcount" style="font-size:12.5px"></span>
    </div>
    <div class="dsplit"><div id="dres"><div class="skel" style="height:300px"></div></div><div class="glass" style="padding:0"><div class="bosses" id="dboss"></div></div></div></div>`;
  let L;
  try { L = await getLoot(S.season); } catch (e) { $('#dres').innerHTML = noData(e.message); return; }
  if (L.sources?.length) $('#dsrc').textContent = `Boss da farmare per ogni unico · stagione ${S.season} · fonti: ${L.sources.join(', ')}${L.ptr ? ' · dati del PTR' : ''}`;
  const favs = new Set(S.st.settings?.favorites || []);
  const bossById = Object.fromEntries((L.bosses || []).map(b => [b.id, b]));
  const draw = () => {
    const q = norm(ui.q);
    let items = L.items.filter(it => !it.mat);
    if (ui.cls) items = items.filter(it => !it.classes?.length || it.classes.includes(ui.cls) || it.sources.some(s => s.cls === ui.cls));
    if (ui.boss) items = items.filter(it => it.sources.some(s => s.boss === ui.boss));
    if (ui.fav) items = items.filter(it => favs.has(it.key || it.en));
    if (q) items = items.filter(it => norm(it.name).includes(q) || norm(it.en).includes(q) || norm(it.type).includes(q));
    if (ui.cat === 'mitici') items = items.filter(it => it.mythic); else if (ui.cat) items = items.filter(it => (it.cat || 'altro') === ui.cat && !it.mythic);
    $('#dcount').textContent = `${items.length} oggetti`;
    const shown = items.slice(0, 150);
    const b = ui.boss ? bossById[ui.boss] : null;
    $('#dres').innerHTML = (b ? `<div class="glass card" style="margin-bottom:14px"><h2>${IC.boss.replace('<svg', '<svg width="18" height="18"')} ${esc(b.name)}<span class="sp"></span><button class="btn sm ghost" id="dclr">Tutti i boss ✕</button></h2>${b.location ? `<div class="muted" style="font-size:12.5px">${IC.pin.replace('<svg', '<svg width="13" height="13"')} ${esc(b.location)}</div>` : ''}${b.info?.length ? `<div class="info">${b.info.map(x => `<div>${esc(x)}</div>`).join('')}</div>` : ''}${b.summon?.length ? `<div class="row wrap" style="gap:6px;margin-top:8px"><span class="lbl">Evocazione e rune</span>${b.summon.map(x => `<span class="badge">${esc(x)}</span>`).join('')}</div>` : ''}</div>` : '') +
      (shown.length ? groupByCat(shown).map(([cat, list]) => `<div class="lbl" style="margin:16px 2px 8px">${esc(catLabel(cat))} · ${list.length}</div><div class="ditems">${list.map((it, i) => itemCard(it, i, favs, bossById)).join('')}</div>`).join('') : `<div class="glass empty">${IC.search}<span>Nessun oggetto trovato${q ? ` per “${esc(ui.q)}”` : ''}</span></div>`) +
      (items.length > shown.length ? `<div class="muted" style="text-align:center;margin-top:12px">…e altri ${items.length - shown.length}: affina la ricerca</div>` : '');
    $('#dclr')?.addEventListener('click', () => { ui.boss = ''; draw(); drawBosses(); });
    $$('.star', $('#dres')).forEach(s => s.onclick = async ev => { ev.stopPropagation(); const k = s.dataset.k; favs.has(k) ? favs.delete(k) : favs.add(k); s.classList.toggle('on', favs.has(k)); try { const st = await post('/api/settings', { favorites: [...favs] }); S.st.settings = st; } catch { } });
    $$('.src[data-b]', $('#dres')).forEach(s => s.onclick = () => { ui.boss = s.dataset.b; draw(); drawBosses(); $('#main').scrollTop = 0; });
  };
  const drawBosses = () => {
    $('#dboss').innerHTML = `<div class="lbl" style="padding:6px 12px 8px">Boss e fonti</div>` + (L.bosses || []).map(b => `<div class="bossrow ${ui.boss === b.id ? 'on' : ''}" data-b="${b.id}"><span class="ic" style="width:30px;height:30px;border-radius:9px;display:grid;place-items:center;background:${b.kind === 'boss' ? 'rgba(255,61,31,.16)' : (b.kind === 'mythic' ? 'rgba(217,160,255,.16)' : 'rgba(255,255,255,.07)')};color:${b.kind === 'boss' ? '#ff8a6a' : (b.kind === 'mythic' ? '#d9a0ff' : 'var(--ink2)')}">${(b.kind === 'boss' ? IC.boss : IC.star).replace('<svg', '<svg width="16" height="16"')}</span><div class="t"><b>${esc(b.name)}</b><small>${b.items.length} oggetti${b.location ? ' · ' + esc(b.location) : ''}</small></div></div>`).join('');
    $$('.bossrow', $('#dboss')).forEach(r => r.onclick = () => { ui.boss = ui.boss === r.dataset.b ? '' : r.dataset.b; draw(); drawBosses(); });
  };
  draw(); drawBosses();
  const inp = $('#dq');
  inp.oninput = () => { ui.q = inp.value; draw(); };
  inp.onkeydown = e => { if (e.key === 'Escape') { inp.value = ''; ui.q = ''; draw(); } };
  $$('#dcls button').forEach(b => b.onclick = () => { ui.cls = b.dataset.v; $$('#dcls button').forEach(x => x.classList.toggle('on', x === b)); draw(); });
  $('#dfav').onchange = e => { ui.fav = e.target.checked; draw(); };
  $$('#dcat button').forEach(b => b.onclick = () => { ui.cat = b.dataset.v; $$('#dcat button').forEach(x => x.classList.toggle('on', x === b)); draw(); });
  setTimeout(() => inp.focus(), 50);
};
const ITEMCATS = [['mitici', 'Mitici'], ['armi', 'Armi'], ['mano secondaria', 'Mano secondaria'], ['elmi', 'Elmi'], ['torso', 'Torso'], ['guanti', 'Guanti'], ['pantaloni', 'Pantaloni'], ['stivali', 'Stivali'], ['amuleti', 'Amuleti'], ['anelli', 'Anelli'], ['talismani', 'Talismani'], ['altro', 'Altro']];
const catLabel = c => (ITEMCATS.find(x => x[0] === c) || [, 'Altro'])[1];
function groupByCat(items) {
  const g = {};
  for (const it of items) { const c = it.mythic ? 'mitici' : (it.cat || 'altro'); (g[c] ||= []).push(it); }
  return ITEMCATS.map(([c]) => [c, g[c]]).filter(x => x[1]?.length);
}
function norm(s) { return String(s || '').toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/[’'`]/g, "'"); }
function itemCard(it, i, favs, bossById) {
  const k = it.key || it.en;
  const srcs = {};
  it.sources.forEach(s => { (srcs[s.boss] ||= []).push(s); });
  return `<div class="glass ditem ${it.mythic ? 'mythic' : ''}"><div class="n"><b>${esc(nm(it.name, it.en))}</b><button class="star ${favs.has(k) ? 'on' : ''}" data-k="${esc(k)}" title="Segui">${IC.star}</button></div>
    <div class="en">${esc([it.type, sub(it.name, it.en)].filter(Boolean).join(' · '))}${it.classes?.length && it.classes.length < 8 ? ' · ' + it.classes.map(clsName).join(', ') : ''}</div>
    <div class="srcs">${Object.entries(srcs).slice(0, 4).map(([bid, list]) => { const b = bossById[bid] || { name: bid, kind: 'special' }; const cl = [...new Set(list.map(x => x.cls).filter(Boolean))]; return `<div class="src" data-b="${esc(bid)}" style="cursor:pointer"><span class="bi" style="${b.kind !== 'boss' ? 'background:rgba(255,255,255,.07);color:var(--ink2)' : ''}">${(b.kind === 'boss' ? IC.boss : IC.star).replace('<svg', '<svg')}</span><div><div>${esc(b.name)}${cl.length ? ` <small>· ${cl.map(clsName).join(', ')}</small>` : ''}</div>${b.location ? `<small>${esc(b.location)}</small>` : ''}</div></div>`; }).join('')}</div></div>`;
}

// ---------------------------------------------------------------- eventi e mappa
const MTYPES = [
  ['World Boss', 'Boss del mondo', '#ff6a2b', IC.boss],
  ['Boss Lair', 'Covi dei boss', '#ff4a6a', '<svg viewBox="0 0 24 24"><path d="M4 4c1 5 3 7 6 8M20 4c-1 5-3 7-6 8M8 14h8l-1 6H9z"/></svg>'],
  ['Waygate', 'Portali', '#6fd3ff', '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="7"/><circle cx="12" cy="12" r="3"/></svg>'],
  ['Stronghold', 'Roccaforti', '#ffcf6a', '<svg viewBox="0 0 24 24"><path d="M4 20V8l3 2V6l3 2V4h4v4l3-2v4l3-2v12z"/></svg>'],
  ['Dungeon', 'Dungeon', '#e0b8ff', '<svg viewBox="0 0 24 24"><path d="M5 21V10a7 7 0 0 1 14 0v11M9 21v-6h6v6"/></svg>'],
  ['Altar of Lilith', 'Altari di Lilith', '#ff7b9a', '<svg viewBox="0 0 24 24"><path d="M12 3l5 7-5 11-5-11z"/></svg>'],
  ['Tenet of Akarat', 'Tenet di Akarat', '#9ef0c8', '<svg viewBox="0 0 24 24"><path d="M4 5h7a2 2 0 0 1 2 2v13a2 2 0 0 0-2-2H4zM20 5h-7"/><path d="M20 5v13h-7"/></svg>'],
];
PAGES.eventi = async main => {
  const ui = S.ui.ev ||= { off: { 'Altar of Lilith': true, 'Tenet of Akarat': true, Dungeon: true } };
  const evs = S.events?.events || [];
  main.innerHTML = `<div class="page">
    <div class="hdr"><div><h1 class="pt">Eventi e mappa</h1><p class="sub" style="margin:0">Boss del mondo${S.events?.bossSource ? ' (orari e zone da ' + esc(S.events.bossSource) + ')' : ''}, Maree infernali e Legioni · orari locali</p></div></div>
    <div class="esplit">
      <div class="glass elist">${evs.length ? evs.slice(0, 40).map(e => { const st = evState(e); return `<div class="erow ${st} ${e.guess ? 'guess' : ''}"><span class="ic k-${e.kind}">${IC[e.kind]}</span><div class="t"><b>${esc(e.kind === 'boss' ? e.title : (e.kind === 'helltide' ? 'Marea infernale' : 'Legione'))}</b><small>${e.zones?.length ? esc(zonesTxt(e.zones)) + ' · ' : ''}${dayLbl(e.start)} ${hhmm(e.start)}${e.guess ? ' · stimato' : ''}</small></div><div class="cd"><span data-cd="${e.start}" data-end="${e.end}">${st === 'live' ? 'in corso' : fmtLeft(e.start - nowS())}</span><small>${st === 'live' ? 'fino alle ' + hhmm(e.end) : ''}</small></div></div>`; }).join('') : `<div class="empty">${IC.warn}<span>Orari non disponibili</span></div>`}</div>
      <div class="glass mapwrap"><div class="map" id="map"><div class="tiles"></div><div class="pins"></div></div>
        <div class="mapui"><div class="glass" id="mfil"></div></div>
        <div class="zoom"><button data-z="1">+</button><button data-z="-1">−</button></div>
      </div>
    </div></div>`;
  let md, found = {};
  try { md = await api('/api/map'); found = await api('/api/mapfound'); } catch (e) { $('#map').innerHTML = `<div class="empty">${IC.warn}<span>Mappa non disponibile: ${esc(e.message)}</span></div>`; return; }
  const markers = md?.result?.pageContext?.markers || [];
  const nextBoss = evs.find(e => e.kind === 'boss' && evState(e) !== 'past' && e.zones?.length);
  const bossZones = new Set((nextBoss?.zones || []).map(z => z.name));
  mapView($('#map'), markers, found, ui, bossZones);
};

function mapView(el, markers, found, ui, bossZones) {
  const tiles = $('.tiles', el), pins = $('.pins', el);
  const proj = (lat, lng, z) => { const s = 256 * Math.pow(2, z); const x = (lng + 180) / 360 * s; const r = lat * Math.PI / 180; const y = (1 - Math.log(Math.tan(r) + 1 / Math.cos(r)) / Math.PI) / 2 * s; return [x, y]; };
  const v = S.ui.mapv ||= (() => { const [x1, y1] = proj(84, -95, 0), [x2, y2] = proj(-76, 135, 0); return { z: 2.2, cx: (x1 + x2) / 2, cy: (y1 + y2) / 2, fit: [x1, y1, x2, y2] }; })();
  const W = () => el.clientWidth, H = () => el.clientHeight;
  if (!v.fitted && W()) { const [x1, y1, x2, y2] = v.fit; v.z = Math.max(1, Math.min(6, Math.log2(Math.min(W() / (x2 - x1), H() / (y2 - y1))))); v.fitted = true; }
  const tileUrl = (z, x, y) => window.DEMO ? `https://sunderarmor.com/DIABLO4/VoHMap/${z}/${x}/${y}.jpg` : `/api/tile/${z}/${x}/${y}`;
  const tip = document.createElement('div'); tip.className = 'tip'; tip.style.display = 'none'; el.appendChild(tip);
  const counts = {};
  markers.forEach(m => { const c = counts[m.type] ||= { n: 0, f: 0 }; c.n++; if (found[m.id]) c.f++; });
  const fil = $('#mfil');
  const drawFil = () => {
    fil.innerHTML = `<div class="lbl" style="padding:2px 8px 6px">Filtri</div>` + MTYPES.map(([t, l, c, ic]) => `<div class="mf ${ui.off[t] ? 'off' : ''}" data-t="${t}"><span class="sw" style="color:${c}">${ic}</span>${l}<small>${counts[t] ? (t === 'World Boss' || t === 'Boss Lair' ? counts[t].n : counts[t].f + '/' + counts[t].n) : ''}</small></div>`).join('') + `<div class="muted" style="font-size:11px;padding:6px 8px 0">Clic su un'icona per segnarla come trovata</div>`;
    $$('.mf', fil).forEach(r => r.onclick = () => { ui.off[r.dataset.t] = !ui.off[r.dataset.t]; drawFil(); render(); });
  };
  const meta = Object.fromEntries(MTYPES.map(x => [x[0], x]));
  function render() {
    const zi = Math.max(0, Math.min(6, Math.round(v.z))), sc = Math.pow(2, v.z - zi), n = 1 << zi;
    const w = W(), h = H();
    const ox = w / 2 - v.cx * Math.pow(2, v.z), oy = h / 2 - v.cy * Math.pow(2, v.z);
    tiles.style.transform = `translate(${ox}px,${oy}px) scale(${sc})`;
    const tsz = 256 * sc;
    const x0 = Math.max(0, Math.floor(-ox / tsz)), x1 = Math.min(n - 1, Math.floor((w - ox) / tsz));
    const y0 = Math.max(0, Math.floor(-oy / tsz)), y1 = Math.min(n - 1, Math.floor((h - oy) / tsz));
    const want = new Set();
    for (let x = x0; x <= x1; x++) for (let y = y0; y <= y1; y++) want.add(`${zi}/${x}/${y}`);
    $$('img', tiles).forEach(im => { if (!want.has(im.dataset.k)) im.remove(); else want.delete(im.dataset.k); });
    want.forEach(k => { const [z, x, y] = k.split('/').map(Number); const im = new Image(); im.dataset.k = k; im.src = tileUrl(z, x, y); im.style.left = x * 256 + 'px'; im.style.top = y * 256 + 'px'; im.draggable = false; tiles.appendChild(im); });
    // punti
    const html = [];
    for (const m of markers) {
      if (m.type === 'ZoneName') { const [x, y] = proj(m.coords[0], m.coords[1], v.z); html.push(`<div class="zlabel" style="left:${ox + x}px;top:${oy + y}px">${esc(m.name)}</div>`); continue; }
      if (ui.off[m.type] || !meta[m.type]) continue;
      const [x, y] = proj(m.coords[0], m.coords[1], v.z);
      const px = ox + x, py = oy + y;
      if (px < -30 || py < -30 || px > w + 30 || py > h + 30) continue;
      const [, , col, ic] = meta[m.type];
      const isWB = m.type === 'World Boss';
      const next = isWB && bossZones.has(m.zone);
      html.push(`<div class="pin ${isWB ? 't-wb' : ''} ${next ? 'next' : ''} ${found[m.id] ? 'found' : ''}" data-id="${m.id}" data-n="${esc(m.name)}" data-z="${esc(m.zone)}" data-t="${esc(m.type)}" style="left:${px}px;top:${py}px;color:${next ? '' : col}">${ic}</div>`);
    }
    pins.innerHTML = html.join('');
  }
  drawFil(); render();
  let drag = null;
  el.onmousedown = e => { if (e.button !== 0 || e.target.closest('.pin')) return; drag = { x: e.clientX, y: e.clientY, cx: v.cx, cy: v.cy, moved: false }; el.classList.add('drag'); };
  addEventListener('mousemove', e => { if (!drag || !document.body.contains(el)) return; const s = Math.pow(2, v.z); v.cx = drag.cx - (e.clientX - drag.x) / s; v.cy = drag.cy - (e.clientY - drag.y) / s; drag.moved = true; render(); });
  addEventListener('mouseup', () => { drag = null; el.classList.remove('drag'); });
  const zoomAt = (dz, cx, cy) => { const nz = Math.max(1, Math.min(6.4, v.z + dz)); const r = el.getBoundingClientRect(); const mx = (cx ?? r.left + r.width / 2) - r.left - r.width / 2, my = (cy ?? r.top + r.height / 2) - r.top - r.height / 2; const s0 = Math.pow(2, v.z), s1 = Math.pow(2, nz); v.cx += mx / s0 - mx / s1; v.cy += my / s0 - my / s1; v.z = nz; render(); };
  el.onwheel = e => { e.preventDefault(); zoomAt(e.deltaY < 0 ? .35 : -.35, e.clientX, e.clientY); };
  $$('.zoom button').forEach(b => b.onclick = () => zoomAt(+b.dataset.z * .6));
  pins.onmouseover = e => { const p = e.target.closest('.pin'); if (!p) { tip.style.display = 'none'; return; } const t = meta[p.dataset.t]; tip.innerHTML = `<b>${esc(p.dataset.t === 'World Boss' ? 'Boss del mondo · ' + p.dataset.n : p.dataset.n)}</b><br><span class="muted">${esc(t?.[1] || '')} · ${esc(p.dataset.z)}${p.classList.contains('found') ? ' · trovato' : ''}</span>`; tip.style.left = p.style.left; tip.style.top = p.style.top; tip.style.display = 'block'; };
  pins.onmouseleave = () => tip.style.display = 'none';
  pins.onclick = async e => { const p = e.target.closest('.pin'); if (!p || p.dataset.t === 'World Boss' || p.dataset.t === 'Boss Lair') return; const id = p.dataset.id; const on = !found[id]; if (on) found[id] = true; else delete found[id]; const c = counts[p.dataset.t]; c.f += on ? 1 : -1; p.classList.toggle('found', on); drawFil(); try { await post('/api/mapfound', { [id]: on }); } catch { } };
  new ResizeObserver(() => document.body.contains(el) && render()).observe(el);
}

// ---------------------------------------------------------------- patch
PAGES.patch = async main => {
  const ui = S.ui.patch ||= { sel: 'a0p0' };
  main.innerHTML = `<div class="page"><div class="hdr"><div><h1 class="pt">Patch e hotfix</h1><p class="sub" style="margin:0">Note ufficiali di Blizzard e messaggi dello staff sul forum</p></div></div><div class="psplit"><div class="glass plist" id="plist"><div class="skel" style="height:300px"></div></div><div class="glass pbody" id="pbody"></div></div></div>`;
  let d;
  try { d = await api('/api/patches'); } catch (e) { $('#plist').innerHTML = noData(e.message); $('#pbody').innerHTML = ''; return; }
  const items = [];
  (d.articles || []).forEach((a, ai) => (a.patches || []).forEach((p, pi) => items.push({ id: `a${ai}p${pi}`, title: p.title, group: a.title, html: p.html, url: a.url, date: p.date, kind: a.ptr ? 'PTR' : 'Patch' })));
  (d.forum || []).forEach((f, i) => items.push({ id: 'f' + i, title: f.title, group: 'Forum · staff Blizzard', html: f.html || `<p>${esc(f.excerpt)}</p>`, url: f.url, date: f.at, kind: 'Forum', author: f.author }));
  if (!items.length) { $('#plist').innerHTML = '<div class="muted" style="padding:12px">Nessuna patch.</div>'; return; }
  let g = '';
  $('#plist').innerHTML = items.map(it => { const h = it.group !== g ? `<div class="lbl" style="padding:12px 12px 6px">${esc(it.group)}</div>` : ''; g = it.group; return h + `<div class="prow ${ui.sel === it.id ? 'on' : ''}" data-id="${it.id}"><b>${esc(it.title)}</b><small>${it.date ? dateLbl(it.date) : ''}${it.author ? ' · ' + esc(it.author) : ''}</small></div>`; }).join('');
  const show = id => { const it = items.find(x => x.id === id) || items[0]; ui.sel = it.id; $$('.prow').forEach(r => r.classList.toggle('on', r.dataset.id === it.id)); $('#pbody').innerHTML = `<div class="row"><span class="badge">${esc(it.kind)}</span><span class="muted" style="font-size:12.5px">${esc(it.group)}${it.date ? ' · ' + dateLbl(it.date) : ''}</span><span class="sp"></span><button class="btn sm" id="pext">${IC.ext} Originale</button></div><h1 class="pt" style="font-size:24px;margin:12px 0 4px">${esc(it.title)}</h1><div class="${it.kind === 'Forum' ? 'blue' : ''}">${it.html || ''}</div>`; $('#pext').onclick = () => openLink(it.url); $('#pbody').scrollTop = 0; };
  $$('.prow').forEach(r => r.onclick = () => show(r.dataset.id));
  show(ui.sel);
};

// ---------------------------------------------------------------- macro (profili suggeriti per D4Pressure)
const CHANNEL = /whirlwind|turbine|incinerat|incenerire|dance of knives|danza dei pugnali|flay|scorticare|barrage|raffica|rapid fire|fuoco rapido|lightning spear|blood wave|siphon|drain|ice armor/i;
const ULTIMATE = /call of the ancients|richiamo degli antichi|wrath of the berserker|iron maelstrom|deep freeze|unstoppable monster|cataclysm|grizzly rage|lacerate|army of the dead|bone storm|blood mist|shadow clone|rain of arrows|death trap|inferno|arbiter|heaven|apocalypse|colossus|evade|leap|dash|teleport|charge/i;
function suggestRows(skills) {
  return skills.map((s, i) => {
    const n = (s.en || '') + ' ' + (s.name || '');
    const key = SLOT_KEYS[i] || '';
    if (CHANNEL.test(n)) return { skill: s.name, key, mode: 'Hold', delayMs: 0, on: true, why: 'Abilità canalizzata: si tiene premuta' };
    if (i === 0 && /basic|primary|base/i.test(s.key || '')) return { skill: s.name, key, mode: 'Tap', delayMs: 250, on: false, why: 'Abilità base: di solito non serve automatizzarla' };
    if (ULTIMATE.test(n)) return { skill: s.name, key, mode: 'Tap', delayMs: 1000, on: false, why: 'Movimento o ultimate: meglio usarla a mano' };
    return { skill: s.name, key, mode: 'Tap', delayMs: i < 2 ? 300 : 700, on: true, why: 'Grido, aura o abilità a ricarica breve: tocco periodico' };
  });
}
function draftFromBuild(b, p, v) {
  return { id: '', name: (b.title || p.name).replace(/build guide|guide|endgame|leveling/ig, '').trim(), class: (p.class || b.cls || '').replace(/^./, c => c.toUpperCase()), build: b.title, planner: p.id, notes: `Variante: ${v.name}`, rows: suggestRows(v.skills) };
}
PAGES.macro = async main => {
  let list = [];
  try { list = await api('/api/macro'); } catch { }
  const ui = S.ui.macro ||= { sel: null };
  if (S.ui.macroDraft) { ui.edit = S.ui.macroDraft; S.ui.macroDraft = null; ui.sel = null; }
  if (!ui.edit && ui.sel) ui.edit = list.find(x => x.id === ui.sel) || null;
  const d4p = S.st.d4p || {};
  main.innerHTML = `<div class="page">
    <div class="hdr"><div><h1 class="pt">Macro</h1><p class="sub" style="margin:0">Profili suggeriti da impostare a mano in D4Pressure, uno per build</p></div><span class="sp"></span>
      <span class="statusline"><span class="dot ${d4p.running ? 'ok' : (d4p.path ? 'warn' : 'bad')}"></span>${d4p.running ? 'D4Pressure aperto' : (d4p.path ? 'D4Pressure trovato' : 'D4Pressure non trovato')}</span>
      <button class="btn" id="mlaunch" ${d4p.path ? '' : 'disabled'}>${IC.play} Avvia D4Pressure</button></div>
    <div class="warnbox" style="margin-bottom:14px">${IC.warn}<div>Hellrise non preme mai tasti e non crea file per D4Pressure: ti suggerisce solo come impostarlo. Blizzard tollera "1 tasto = 1 azione"; l'automazione delle pressioni (come fa D4Pressure) è a tuo rischio.</div></div>
    <div class="bsplit">
      <div class="glass" style="padding:12px"><div class="row" style="padding:2px 4px 10px"><div class="lbl">I tuoi profili</div><span class="sp"></span><button class="btn sm" id="mnew">${IC.plus} Da una build</button></div>
        <div class="mini">${list.length ? list.map(p => `<div class="mrow ${ui.edit?.id === p.id ? 'on' : ''}" data-id="${p.id}" style="${ui.edit?.id === p.id ? 'background:rgba(255,255,255,.08)' : ''}"><span class="ci" style="color:var(--acc2)">${clsIcon(p.class.toLowerCase()).replace('<svg', '<svg width="20" height="20"')}</span><div class="t"><b>${esc(p.name)}</b><small>${esc(p.build || p.class)}</small></div></div>`).join('') : '<div class="muted" style="padding:6px 8px">Nessun profilo salvato. Apri una build Maxroll e premi "Suggerisci profilo macro".</div>'}</div></div>
      <div class="glass card" id="medit"></div>
    </div></div>`;
  $('#mlaunch').onclick = async () => { try { await post('/api/macro/launch', {}); toast('D4Pressure avviato'); } catch (e) { toast(e.message); } };
  $('#mnew').onclick = () => { location.hash = '#/build'; toast('Scegli una build Maxroll e premi "Suggerisci profilo macro"'); };
  $$('.mrow[data-id]', main).forEach(r => r.onclick = () => { ui.sel = r.dataset.id; ui.edit = null; PAGES.macro(main); });
  const e = ui.edit, ed = $('#medit');
  if (!e) { ed.innerHTML = `<div class="empty">${IC.keyb}<span>Scegli un profilo o creane uno da una build</span></div>`; return; }
  ed.innerHTML = `<div class="row wrap" style="gap:12px;align-items:flex-end">
      <label class="field" style="flex:1;min-width:180px"><span>Nome profilo (come il personaggio in D4Pressure)</span><input class="inp" id="mname" value="${esc(e.name)}"></label>
      <label class="field" style="width:170px"><span>Classe</span><select class="inp" id="mclass">${CLASSES.map(([c, en]) => `<option value="${en}" ${en.toLowerCase() === (e.class || '').toLowerCase() ? 'selected' : ''}>${esc(clsName(c))}</option>`).join('')}</select></label></div>
    ${e.build ? `<div class="muted" style="font-size:12.5px;margin-top:8px">Build: ${esc(e.build)}</div>` : ''}
    <table class="mtable" style="margin-top:14px"><thead><tr><th>Abilità</th><th>Tasto</th><th>Modalità</th><th>Ritardo</th><th>Attiva</th><th></th></tr></thead><tbody>${(e.rows || []).map((r, i) => `<tr data-i="${i}">
      <td><b>${esc(r.skill)}</b>${r.why ? `<div class="muted" style="font-size:11.5px">${esc(r.why)}</div>` : ''}</td>
      <td><input class="inp" data-k="key" value="${esc(r.key)}" style="width:70px;text-align:center"></td>
      <td><select class="inp" data-k="mode"><option ${r.mode === 'Tap' ? 'selected' : ''}>Tap</option><option ${r.mode === 'Hold' ? 'selected' : ''}>Hold</option></select></td>
      <td><input class="inp" data-k="delayMs" type="number" min="10" max="5000" step="10" value="${r.delayMs || 0}" style="width:90px" ${r.mode === 'Hold' ? 'disabled' : ''}> <span class="muted">ms</span></td>
      <td><label class="tog" style="padding:0"><input type="checkbox" data-k="on" ${r.on ? 'checked' : ''}><i></i></label></td>
      <td><button class="btn sm ghost" data-del="${i}" title="Togli">${IC.trash}</button></td></tr>`).join('')}</tbody></table>
    <button class="btn sm ghost" id="madd">${IC.plus} Aggiungi riga</button>
    <label class="field" style="margin-top:14px"><span>Note</span><textarea class="inp" id="mnotes">${esc(e.notes || '')}</textarea></label>
    <div class="glass" style="padding:14px;margin-top:14px;background:rgba(0,0,0,.18)"><div class="lbl" style="margin-bottom:8px">Come impostarlo in D4Pressure</div><div style="font-size:13px;color:var(--ink2)">Crea il personaggio <b>${esc(e.name)}</b> (${esc(e.class)}), poi una riga per ogni abilità attiva:<ul style="margin:6px 0 0;padding-left:18px">${(e.rows || []).filter(r => r.on).map(r => `<li><span class="keycap">${esc(r.key || '?')}</span> ${esc(r.skill)} · <b>${r.mode}</b>${r.mode === 'Tap' ? ' ogni ' + r.delayMs + ' ms' : ''}</li>`).join('') || '<li>nessuna riga attiva</li>'}</ul></div></div>
    <div class="row" style="margin-top:16px;gap:8px"><button class="btn pri" id="msave">Salva profilo</button><button class="btn" id="mcopy">${IC.copy} Copia come testo</button><span class="sp"></span>${e.id ? `<button class="btn ghost" id="mdel">${IC.trash} Elimina</button>` : ''}</div>`;
  const read = () => {
    e.name = $('#mname').value.trim(); e.class = $('#mclass').value; e.notes = $('#mnotes').value;
    $$('tr[data-i]', ed).forEach(tr => { const r = e.rows[+tr.dataset.i]; $$('[data-k]', tr).forEach(inp => { const k = inp.dataset.k; r[k] = inp.type === 'checkbox' ? inp.checked : (inp.type === 'number' ? +inp.value : inp.value); }); });
  };
  $$('select[data-k="mode"]', ed).forEach(s => s.onchange = () => { read(); PAGES.macro(main); });
  $$('[data-del]', ed).forEach(b => b.onclick = () => { read(); e.rows.splice(+b.dataset.del, 1); PAGES.macro(main); });
  $('#madd').onclick = () => { read(); e.rows.push({ skill: 'Nuova abilità', key: '', mode: 'Tap', delayMs: 500, on: true }); PAGES.macro(main); };
  $('#msave').onclick = async () => { read(); try { const saved = await post('/api/macro', e); ui.sel = saved.id; ui.edit = null; toast('Profilo salvato'); PAGES.macro(main); } catch (x) { toast(x.message); } };
  $('#mcopy').onclick = () => { read(); copy(`${e.name} (${e.class})\n` + e.rows.filter(r => r.on).map(r => `${r.key}\t${r.skill}\t${r.mode}${r.mode === 'Tap' ? ' ' + r.delayMs + 'ms' : ''}`).join('\n'), 'Profilo'); };
  $('#mdel')?.addEventListener('click', async () => { try { await post('/api/macro?delete=1', { id: e.id }); ui.sel = null; ui.edit = null; PAGES.macro(main); } catch { } });
};

// ---------------------------------------------------------------- overlay in gioco
const MOVEKEYS = [[0xA4, 'Alt sinistro'], [0xA5, 'Alt destro (AltGr)'], [0xA2, 'Ctrl sinistro'], [0xA3, 'Ctrl destro'], [0xA0, 'Shift sinistro'], [0x09, 'Tab'], [0x14, 'Bloc Maiusc'], [0xC0, 'ò'], [0xDC, '\\ (sotto Esc)'], [0x05, 'Tasto laterale mouse 4'], [0x06, 'Tasto laterale mouse 5'], ...Array.from({ length: 12 }, (_, i) => [0x70 + i, 'F' + (i + 1)])];
PAGES.overlay = async main => {
  const o = S.st.settings.overlay || {};
  const tg = (path, label, small) => { const v = path.split('.').reduce((a, k) => a?.[k], o); return `<label class="tog"><input type="checkbox" data-o="${path}" ${v ? 'checked' : ''}><i></i><span><b>${label}</b>${small ? `<small>${small}</small>` : ''}</span></label>`; };
  main.innerHTML = `<div class="page">
    <div class="hdr"><div><h1 class="pt">Overlay in gioco</h1><p class="sub" style="margin:0">Riquadri sopra Diablo IV (finestra senza bordi): timer degli eventi e guida paragon. Non legge e non tocca il gioco.</p></div><span class="sp"></span><span class="statusline"><span class="dot ${S.st.diablo ? 'ok' : ''}"></span>${S.st.diablo ? 'Diablo IV aperto' : 'Diablo IV chiuso'}</span></div>
    <div class="sgrid">
      <div class="glass card"><h2>Generale</h2>
        ${tg('enabled', 'Overlay attivo', 'Compare solo quando Diablo IV è in primo piano')}
        ${tg('always', 'Mostralo anche fuori dal gioco', 'Utile per provarlo e sistemare le posizioni')}
        <label class="field"><span>Dimensione · <b id="scv">${Math.round((o.scale || 1) * 100)}%</b></span><input type="range" min="0.6" max="1.6" step="0.05" value="${o.scale || 1}" data-o="scale"></label>
        <label class="field"><span>Opacità · <b id="opv">${o.opacity || 92}%</b></span><input type="range" min="20" max="100" step="2" value="${o.opacity || 92}" data-o="opacity"></label>
        <label class="field"><span>Tieni premuto per spostare e bloccare i riquadri</span><select class="inp" data-o="moveKey">${MOVEKEYS.map(([v, l]) => `<option value="${v}" ${(o.moveKey || 0xA4) === v ? 'selected' : ''}>${l}</option>`).join('')}</select></label>
        <div class="muted" style="font-size:12.5px;line-height:1.5">In gioco, tenendo premuto il tasto scelto, trascini i riquadri col mouse; il lucchetto in alto a destra di ogni riquadro lo blocca (o lo sblocca) dov'è. Lasciando il tasto i clic tornano al gioco.</div>
        ${tg('timers.lock', 'Timer bloccati')}${tg('paragon.lock', 'Riquadro paragon bloccato')}
        <div class="keys" style="margin-top:6px"><span class="keycap">Ctrl+Shift+O</span><span>mostra o nasconde tutto</span><span class="keycap">Ctrl+Shift+E</span><span>sposta i riquadri col mouse (poi di nuovo per finire)</span><span class="keycap">Ctrl+Shift+T</span><span>timer sì/no</span><span class="keycap">Ctrl+Shift+P</span><span>paragon sì/no</span><span class="keycap">Ctrl+Shift+← →</span><span>tavola paragon precedente / successiva</span><span class="keycap">Ctrl+Shift+↑ ↓</span><span>fase del paragon</span><span class="keycap">Ctrl+Shift+R / Q</span><span>solo riquadro: ruota la tavola in senso orario / antiorario</span><span class="keycap">Ctrl+Shift+M</span><span>automatico ↔ riquadro</span></div>
      </div>
      <div class="glass card"><h2>Timer degli eventi</h2>
        ${tg('timers.on', 'Mostra i timer')}
        ${tg('tBoss', 'Boss del mondo')}${tg('tHelltide', 'Marea infernale')}${tg('tLegion', 'Legione')}
      </div>
      <div class="glass card" style="grid-column:1/-1"><h2>Guida paragon</h2>
        ${tg('paragon.on', 'Mostra il paragon', 'Tavole nell\'ordine della build con le caselle da prendere')}
        <div class="row wrap" style="gap:14px;align-items:flex-start;margin:4px 0 10px">
          <div class="field"><span>Come mostrarlo</span><div class="seg" id="ovmode"><button data-m="auto" class="${(o.mode || 'auto') !== 'panel' ? 'on' : ''}">Automatico sulla tavola</button><button data-m="panel" class="${o.mode === 'panel' ? 'on' : ''}">Riquadro sempre visibile</button></div></div>
          <div class="muted" style="font-size:12.5px;flex:1;min-width:260px;line-height:1.55">${o.mode !== 'panel' ? 'Automatico: quando apri il paragon in gioco Hellrise riconosce da solo la tavola (grandezza delle caselle, posizione, rotazione e quale tavola della build è) e ci disegna sopra le cornici delle caselle da prendere. Nessun tasto da premere. Fuori dal paragon non mostra nulla.' : 'Riquadro: la tavola disegnata in un angolo dello schermo, da seguire a occhio. Cambi tavola con Ctrl+Shift+← →, la giri con Ctrl+Shift+R / Q.'}</div>
        </div>
        <div id="ovpar"></div>
      </div>
    </div></div>`;
  const save = async patch => { const cur = { ...(S.st.settings.overlay || {}) }; for (const [k, v] of Object.entries(patch)) { const ks = k.split('.'); if (ks.length === 2) cur[ks[0]] = { ...(cur[ks[0]] || {}), [ks[1]]: v }; else cur[k] = v; } try { S.st.settings = await post('/api/settings', { overlay: cur }); } catch (e) { toast(e.message); } };
  $$('#ovmode button', main).forEach(b => b.onclick = async () => { await save({ mode: b.dataset.m }); PAGES.overlay(main); });
  $$('[data-o]', main).forEach(inp => {
    if (inp.dataset.o === 'scale') { inp.oninput = () => $('#scv').textContent = Math.round(inp.value * 100) + '%'; inp.onchange = () => save({ scale: +inp.value }); return; }
    if (inp.dataset.o === 'opacity') { inp.oninput = () => { $('#opv').textContent = inp.value + '%'; clearTimeout(inp._t); inp._t = setTimeout(() => save({ opacity: +inp.value }), 150); }; return; }
    if (inp.tagName === 'SELECT') { inp.onchange = () => save({ [inp.dataset.o]: +inp.value }); return; }
    inp.onchange = () => save({ [inp.dataset.o]: inp.checked });
  });
  const box = $('#ovpar');
  if (!o.planner) { box.innerHTML = `<div class="muted" style="margin-top:6px">Nessuna build scelta. Apri una build Maxroll in <a class="link" href="#/build">Build</a> e premi “Paragon nell'overlay in gioco”.</div>`; return; }
  box.innerHTML = `<div class="skel" style="height:120px"></div>`;
  let p; try { p = await getPlanner(o.planner); } catch (e) { box.innerHTML = `<div class="muted">Planner non disponibile: ${esc(e.message)}</div>`; return; }
  const vi = Math.min(o.variant || 0, p.variants.length - 1), v = p.variants[vi] || {}, steps = v.paragon || [];
  const si = Math.min(o.step || 0, Math.max(steps.length - 1, 0)), st = steps[si];
  box.innerHTML = `<div class="row wrap" style="gap:10px;margin:6px 0 12px"><b style="font:600 16px var(--serif)">${esc(o.title || p.name)}</b><span class="sp"></span>
      <select class="inp" id="ovv">${p.variants.map((x, i) => `<option value="${i}" ${i === vi ? 'selected' : ''} ${x.paragon?.length ? '' : 'disabled'}>${esc(x.name)}</option>`).join('')}</select>
      ${steps.length > 1 ? `<select class="inp" id="ovs">${steps.map((x, i) => `<option value="${i}" ${i === si ? 'selected' : ''}>${esc(x.name || 'Fase ' + (i + 1))}</option>`).join('')}</select>` : ''}
      <label class="field" style="flex-direction:row;align-items:center;gap:8px"><span>Caselle del riquadro</span><input class="inp" type="number" min="6" max="28" value="${o.cell || 13}" id="ovc" style="width:70px"></label>
      <button class="btn ghost sm" id="ovx">${IC.trash} Togli</button></div>
    ${PLEGEND}<div class="pboards">${(st?.boards || []).map((bd, i) => boardCard(bd, i)).join('') || '<div class="muted">Questa variante non ha un paragon.</div>'}</div>`;
  $$('.pboards canvas', box).forEach((c, i) => drawBoard(c, st.boards[i]));
  $$('.pboard', box).forEach((el, i) => { el.classList.toggle('on', i === (o.board || 0)); el.onclick = async () => { await save({ board: i }); $$('.pboard', box).forEach((x, j) => x.classList.toggle('on', j === i)); toast('Tavola ' + (i + 1) + ' mostrata nell\'overlay'); }; });
  $('#ovv').onchange = async e => { await save({ variant: +e.target.value, step: 0, board: 0 }); PAGES.overlay(main); };
  $('#ovs')?.addEventListener('change', async e => { await save({ step: +e.target.value, board: 0 }); PAGES.overlay(main); });
  $('#ovc').onchange = e => save({ cell: +e.target.value });
  $('#ovx').onclick = async () => { await save({ planner: '', title: '' }); PAGES.overlay(main); };
};

// ---------------------------------------------------------------- impostazioni
PAGES.impostazioni = async main => {
  const s = S.st.settings || {};
  const tg = (k, label, small) => `<label class="tog"><input type="checkbox" data-k="${k}" ${s[k] ? 'checked' : ''}><i></i><span><b>${label}</b>${small ? `<small>${small}</small>` : ''}</span></label>`;
  main.innerHTML = `<div class="page"><h1 class="pt">Impostazioni</h1><p class="sub">Hellrise ${esc(S.st.version || '')} · i dati restano sul tuo PC</p>
    <div class="sgrid">
      <div class="glass card"><h2>Lingua dei nomi di gioco</h2>
        ${tg('langIt', 'Traduci in italiano', 'Oggetti, aspetti, abilità e classi con i nomi italiani del gioco. Disattiva per tenerli in inglese.')}
      </div>
      <div class="glass card"><h2>Notifiche</h2>
        ${tg('notifications', 'Notifiche di Windows')}
        ${tg('notifyBoss', 'Boss del mondo')}${tg('notifyHelltide', 'Maree infernali')}${tg('notifyLegion', 'Legioni')}
        <label class="field"><span>Avvisami prima di (minuti)</span><input class="inp" type="number" min="1" max="60" data-k="notifyBefore" value="${s.notifyBefore || 5}" style="width:110px"></label>
        ${tg('keepInTray', 'Resta vicino all\'orologio a finestra chiusa', 'Così le notifiche arrivano anche mentre giochi')}
        ${tg('autostart', 'Avvia con Windows')}
      </div>
      <div class="glass card"><h2>Server dati</h2>
        <label class="field"><span>Indirizzo</span><input class="inp" data-k="remoteUrl" value="${esc(s.remoteUrl || '')}"></label>
        <label class="field"><span>Codice del server (lo stesso di BetterStomp)</span><input class="inp" type="password" data-k="remoteToken" value="${esc(s.remoteToken || '')}"></label>
        <div id="sstat"></div>
        <div class="row" style="gap:8px"><button class="btn" id="srefresh">${IC.refresh} Aggiorna ora</button></div>
      </div>
      <div class="glass card"><h2>Aspetto</h2>
        ${tg('blur', 'Effetto vetro', 'Sfocatura dei pannelli (disattiva se il PC rallenta)')}
        ${tg('motion', 'Sfondo animato', 'Bagliori e braci in movimento')}
        <div class="field"><span>Colore</span><div class="swatches">${[['ember', 'linear-gradient(135deg,#ff3d1f,#ffb14a)'], ['blood', 'linear-gradient(135deg,#a30022,#ff9a8a)'], ['arcane', 'linear-gradient(135deg,#6a2bff,#6fd3ff)'], ['frost', 'linear-gradient(135deg,#1b5dff,#c6f6ff)']].map(([k, g]) => `<button data-acc="${k}" class="${(s.accent || 'ember') === k ? 'on' : ''}" style="background:${g}"></button>`).join('')}</div></div>
        ${tg('nativeFrame', 'Usa la barra di Windows', 'Richiede il riavvio dell\'app')}
      </div>
      <div class="glass card"><h2>D4Pressure</h2>
        <label class="field"><span>Percorso di D4Pressure.exe (vuoto = cerca da solo)</span><input class="inp" data-k="d4pressurePath" value="${esc(s.d4pressurePath || '')}" placeholder="${esc(S.st.d4p?.path || 'C:\\…\\D4Pressure.exe')}"></label>
      </div>
      <div class="glass card"><h2>Fonti</h2><div class="muted" style="font-size:12.5px;line-height:1.7">Build: Maxroll, D4Builds, Mobalytics · Tier list: Rob2628 · Loot: D4Builds, Maxroll, Mobalytics · Orari dei boss: helltides.com · Mappa: d4builds.gg · Patch: news.blizzard.com e forum ufficiale.<br>Hellrise non è affiliato con Blizzard Entertainment.</div></div>
    </div></div>`;
  $('[data-k="langIt"]').checked = (s.lang || 'it') === 'it';
  renderDataStatus($('#sstat'));
  const save = async patch => { try { S.st.settings = await post('/api/settings', patch); applySettings(); toast('Salvato'); } catch (e) { toast(e.message); } };
  $$('[data-k]', main).forEach(inp => inp.onchange = () => {
    const k = inp.dataset.k;
    if (k === 'langIt') return save({ lang: inp.checked ? 'it' : 'en' });
    save({ [k]: inp.type === 'checkbox' ? inp.checked : (inp.type === 'number' ? +inp.value : inp.value.trim()) });
  });
  $$('[data-acc]', main).forEach(b => b.onclick = () => { $$('[data-acc]').forEach(x => x.classList.toggle('on', x === b)); save({ accent: b.dataset.acc }); });
  $('#srefresh').onclick = async () => { const b = $('#srefresh'); b.disabled = true; try { await api('/api/refresh'); S.builds = {}; S.loot = {}; S.st = await api('/api/state'); renderSeasonPick(); toast('Dati aggiornati'); } catch (e) { toast(e.message); } b.disabled = false; renderDataStatus($('#sstat')); };
};

// ---------------------------------------------------------------- ricerca globale
(() => {
  const inp = $('#gsearch');
  const box = document.createElement('div'); box.className = 'glass gres'; document.body.appendChild(box);
  const place = () => { const r = inp.closest('.gsearch').getBoundingClientRect(); box.style.left = r.left + 'px'; box.style.top = r.bottom + 6 + 'px'; };
  let tm;
  inp.oninput = () => { clearTimeout(tm); tm = setTimeout(run, 120); };
  inp.onfocus = () => inp.value && run();
  inp.onkeydown = e => { if (e.key === 'Escape') { inp.value = ''; box.classList.remove('on'); inp.blur(); } if (e.key === 'Enter') { const f = $('.mrow', box); f?.click(); } };
  addEventListener('keydown', e => { if (e.ctrlKey && e.key.toLowerCase() === 'k') { e.preventDefault(); inp.focus(); inp.select(); } });
  addEventListener('mousedown', e => { if (!e.target.closest('.gres') && e.target !== inp) box.classList.remove('on'); });
  async function run() {
    const q = norm(inp.value.trim());
    if (q.length < 2) { box.classList.remove('on'); return; }
    place();
    let L = null, B = null;
    try { L = await getLoot(S.season); } catch { }
    try { B = await getBuilds(S.season); } catch { }
    const items = (L?.items || []).filter(it => !it.mat && (norm(it.name).includes(q) || norm(it.en).includes(q))).slice(0, 7);
    const bosses = (L?.bosses || []).filter(b => norm(b.name).includes(q)).slice(0, 3);
    const builds = (B?.builds || []).filter(b => norm(b.title).includes(q)).slice(0, 6);
    const bossById = Object.fromEntries((L?.bosses || []).map(b => [b.id, b]));
    box.innerHTML = (items.length ? `<div class="lbl" style="padding:6px 10px">Oggetti</div>` + items.map(it => `<div class="mrow" data-q="${esc(it.en)}"><div class="t"><b style="color:${it.mythic ? '#d9a0ff' : '#e8b56a'}">${esc(nm(it.name, it.en))}</b><small>${esc([...new Set(it.sources.map(s => bossById[s.boss]?.name).filter(Boolean))].slice(0, 3).join(' · '))}</small></div></div>`).join('') : '') +
      (bosses.length ? `<div class="lbl" style="padding:6px 10px">Boss</div>` + bosses.map(b => `<div class="mrow" data-boss="${b.id}"><div class="t"><b>${esc(b.name)}</b><small>${b.items.length} oggetti</small></div></div>`).join('') : '') +
      (builds.length ? `<div class="lbl" style="padding:6px 10px">Build</div>` + builds.map(b => `<div class="mrow" data-burl="${esc(b.url)}" data-bcls="${b.cls}">${tierBadge(b.tier)}<div class="t"><b>${esc(b.title)}</b><small>${esc(b.src)} · ${esc(clsName(b.cls))}</small></div></div>`).join('') : '') ||
      `<div class="muted" style="padding:12px">Nessun risultato</div>`;
    box.classList.add('on');
    $$('.mrow', box).forEach(r => r.onclick = () => {
      box.classList.remove('on');
      if (r.dataset.q) { S.ui.drop = { ...(S.ui.drop || {}), q: r.dataset.q, boss: '' }; location.hash = '#/drop'; if (S.page === 'drop') route(); }
      else if (r.dataset.boss) { S.ui.drop = { ...(S.ui.drop || {}), q: '', boss: r.dataset.boss }; location.hash = '#/drop'; if (S.page === 'drop') route(); }
      else { setCls(r.dataset.bcls); S.ui.build = { ...(S.ui.build || {}), type: 'Tutte', src: 'all', sel: r.dataset.burl, variant: 0 }; location.hash = '#/build'; if (S.page === 'build') route(); }
      inp.value = '';
    });
  }
})();

// ---------------------------------------------------------------- finestra senza bordi
document.addEventListener('mousedown', e => {
  if (e.button !== 0 || !document.body.classList.contains('frameless')) return;
  const z = e.target.closest('[data-rz]'); if (z) { e.preventDefault(); fetch('/api/win?a=size' + z.dataset.rz); return; }
  const b = e.target.closest('[data-win]'); if (b) { e.preventDefault(); fetch('/api/win?a=' + b.dataset.win); return; }
  if (e.target.closest('header.top') && !e.target.closest('a,button,input,select,label,.chipev,.seasonpick') && e.detail === 1) fetch('/api/win?a=drag');
});
document.addEventListener('dblclick', e => { if (document.body.classList.contains('frameless') && e.target.closest('header.top') && !e.target.closest('a,button,input,select,label')) fetch('/api/win?a=max'); });
function sendInner() { if (!window.DEMO) fetch(`/api/win?a=inner&h=${innerHeight}&d=${devicePixelRatio}`).catch(() => { }); }
addEventListener('resize', sendInner); setTimeout(sendInner, 300);
function updBar(ready) {
  let el = $('#updbar');
  if (!ready) { if (el) el.remove(); return; }
  if (el) return;
  el = document.createElement('div'); el.id = 'updbar'; el.className = 'glass';
  el.innerHTML = `<span>✨ <b>Nuova versione di Hellrise</b> pronta</span><button class="btn pri sm">Aggiorna ora</button>`;
  el.querySelector('button').onclick = () => { el.querySelector('button').textContent = 'Riavvio…'; fetch('/api/win?a=update'); };
  document.body.appendChild(el);
}

boot();
