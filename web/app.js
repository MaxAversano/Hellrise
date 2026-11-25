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
const CLASSES = [['barbarian', 'Barbarian'], ['druid', 'Druid'], ['necromancer', 'Necromancer'], ['rogue', 'Rogue'], ['sorcerer', 'Sorcerer'], ['spiritborn', 'Spiritborn'], ['paladin', 'Paladin'], ['warlock', 'Warlock']];
const clsName = c => { const it = S.st?.classesIt?.[c]; const en = (CLASSES.find(x => x[0] === c) || [, c])[1]; return nm(it, en); };
const clsIcon = (c, size) => CLASSES.some(x => x[0] === c) ? `<i class="clsi" style="--m:url(cls/${c}.png)${size ? `;width:${size}px;height:${size}px` : ''}"></i>` : '';
const tierCls = t => 'tier tier-' + (t === 'S+' ? 'Sp' : (/^[SABCDF]$/.test(t) ? t : 'q'));
const tierBadge = (t, c) => t && t !== '?' ? `<span class="${tierCls(t)}" title="Tier list di Rob2628">${esc(t)}</span>` : `<span class="notier" title="${t === '?' ? 'Tier list di Rob2628 ancora senza voti' : 'Non presente nella tier list di Rob2628'}">${c ? clsIcon(c, 16) : ''}</span>`;

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
  setInterval(poll, 3000);
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
    // dati nuovi: si ridisegna solo se la pagina era ancora vuota (così la raccolta non fa saltare la pagina mentre la leggi)
    const waiting = !!document.querySelector('#main .empty, #main .skel, #main .glass.empty');
    if ((seasonsChanged || namesArrived || (dataArrived && ['build', 'drop', 'home'].includes(S.page) && waiting)) && !document.querySelector('input:focus,textarea:focus,select:focus')) { const y = $('#main').scrollTop; route(); if (!seasonsChanged) requestAnimationFrame(() => { $('#main').scrollTop = y; }); }
    updBar(st.update);
    collectBar(st.collector);
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
  $$('#side a').forEach(a => a.classList.toggle('on', a.dataset.p === (p === 'b' ? 'build' : p)));
  const fn = PAGES[p] || PAGES.home;
  setTimeout(cmpBar, 0);
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
  el.innerHTML = ['boss', 'helltide', 'legion'].map(k => { const e = nextOf(k); if (!e) return ''; const live = evState(e) === 'live'; return `<div class="chipev ${live ? 'live' : ''}" onclick="location.hash='#/eventi'" title="${esc(e.title)}${e.zones ? ' · ' + esc(zonesTxt(e.zones)) : ''}"><span class="ic k-${k}">${IC[k]}</span><span class="cl">${k === 'boss' ? esc(e.guess ? 'Boss' : e.title) : (k === 'helltide' ? 'Marea' : 'Legione')}</span> <b data-cd="${e.start}" data-end="${e.end}">${live ? 'in corso' : fmtLeft(e.start - nowS())}</b></div>`; }).join('');
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
  if (!S.builds[season]) S.builds[season] = (async () => {
    const d = await api('/api/builds?s=' + season);
    // stagione nuova (PTR): le fonti che non hanno ancora build mostrano quelle della stagione precedente
    const prev = (S.st.seasons || []).map(x => x.n).filter(n => n < season).sort((a, b) => b - a)[0];
    const meta = (S.st.seasons || []).find(x => x.n === season);
    if (prev && meta?.ptr) {
      try {
        const p = await api('/api/builds?s=' + prev);
        const have = new Set(d.builds.map(b => b.src + '|' + b.cls));
        const add = p.builds.filter(b => !have.has(b.src + '|' + b.cls)).map(b => ({ ...b, fromSeason: prev, tier: '' , tierNote: '', info: '' }));
        d.builds = d.builds.concat(add);
      } catch { }
    }
    return d;
  })().catch(e => { delete S.builds[season]; throw e; });
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
        <div class="glass card"><h2>Dalla patch per te</h2><div id="hpatch"><div class="skel" style="height:60px"></div></div></div>
        <div class="glass card"><h2>Le tue build <span class="sp"></span><a class="link" href="#/build" style="font-size:12.5px">Tutte →</a></h2><div id="hsaved"></div></div>
        <div class="glass card"><h2>Leveling consigliato <span class="count">${esc(clsName(S.cls))}</span></h2><div class="mini" id="topl"><div class="skel" style="height:60px"></div></div></div>
        <div class="glass card"><h2>Ultima patch</h2><div id="lastp"><div class="skel" style="height:60px"></div></div></div>
        <div class="glass card"><h2>Stato dei dati</h2><div id="dstat"></div></div>
      </div>
    </div></div>`;
  renderDataStatus($('#dstat'));
  renderSaved($('#hsaved'), true);
  patchForYou($('#hpatch'));
  try {
    const b = await getBuilds(S.season);
    const rob = b.rob?.tier || [];
    $('#topb').innerHTML = rob.length ? rob.slice(0, 8).map(r => `<div class="mrow" data-cls="${r.cls}">${tierBadge(r.tier, r.cls)}<div class="t"><b>${esc(r.build)}</b><small>${esc(clsName(r.cls))}${r.info ? ' · ' + esc(r.info) : ''}</small></div><span class="ci" style="color:var(--mute)">${clsIcon(r.cls, 18)}</span></div>`).join('') : '<div class="muted">Tier list non ancora disponibile per questa stagione.</div>';
    $('#toplrow')?.addEventListener('click', () => { if (lv?.maxroll) { (S.ui.build ||= { type: 'Endgame', src: 'all', variant: 0 }).variantName = 'level'; openBuildPage(lv.maxroll); } else location.hash = '#/build'; });
    $$('#topb .mrow').forEach(r => r.onclick = () => { setCls(r.dataset.cls); location.hash = '#/build'; });
    const lv = levelingPick(b, S.cls);
    $('#topl').innerHTML = lv ? `<div class="mrow" id="toplrow">${tierBadge(lv.tier, S.cls)}<div class="t"><b>${esc(lv.title)}</b><small>${esc(lv.info || lv.src)}</small></div></div>` : '<div class="muted">Nessuna build leveling per questa classe.</div>';
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
  const ui = S.ui.build ||= { type: 'Endgame', src: 'all', variant: 0 };
  main.innerHTML = `<div class="page">
    <div class="hdr"><div><h1 class="pt">Build</h1><p class="sub" style="margin:0">Maxroll, D4Builds e Mobalytics, con la tier list di Rob2628 · stagione ${S.season || '—'}</p></div></div>
    <div id="saved"></div>
    <div class="classes">${CLASSES.map(([c]) => `<div class="cls ${c === S.cls ? 'on' : ''}" data-c="${c}"><span class="ci">${clsIcon(c)}</span>${esc(clsName(c))}</div>`).join('')}</div>
    <div id="lvb"></div>
    <div class="row wrap" style="gap:8px;margin-bottom:12px">
      <div class="seg" id="btype">${['Endgame', 'Leveling', 'Tutte'].map(t => `<button data-v="${t}" class="${ui.type === t ? 'on' : ''}">${t}</button>`).join('')}</div>
      <div class="seg" id="bsrc">${[['all', 'Tutte le fonti'], ['Maxroll', 'Maxroll'], ['D4Builds', 'D4Builds'], ['Mobalytics', 'Mobalytics']].map(([v, l]) => `<button data-v="${v}" class="${ui.src === v ? 'on' : ''}">${l}</button>`).join('')}</div>
      <span class="sp"></span><span class="muted" id="bcount" style="font-size:12.5px"></span>
    </div>
    <div class="bgrid" id="blist"><div class="skel" style="height:300px;grid-column:1/-1"></div></div></div>`;
  renderSaved($('#saved'));
  $$('.cls', main).forEach(el => el.onclick = () => { setCls(el.dataset.c); PAGES.build(main); });
  $$('#btype button').forEach(b => b.onclick = () => { ui.type = b.dataset.v; PAGES.build(main); });
  $$('#bsrc button').forEach(b => b.onclick = () => { ui.src = b.dataset.v; PAGES.build(main); });
  let data;
  try { data = await getBuilds(S.season); } catch (e) { $('#blist').innerHTML = noData(e.message); return; }
  ui.all = data.builds;
  const lv = levelingPick(data, S.cls);
  $('#lvb').innerHTML = lv ? `<div class="glass lvbanner"><span class="ic">${IC.sprout}</span><div style="flex:1"><div class="lbl">Leveling consigliato per ${esc(clsName(S.cls))}</div><div style="font:600 17px/1.3 var(--serif);margin-top:3px">${esc(lv.title)} ${lv.tier ? tierBadge(lv.tier) : ''}</div><div class="muted" style="font-size:12.5px">${esc(lv.info || '')}${lv.src === 'Rob2628' ? ' · dalla tier list leveling di Rob2628' : ''}</div></div>${lv.maxroll ? `<button class="btn" id="lvopen">Apri su Hellrise</button>` : ''}${lv.url ? `<button class="btn ghost" id="lvlink">${IC.ext} Guida</button>` : ''}</div>` : '';
  if (lv?.url) $('#lvlink').onclick = () => openGuide(lv.url);
  if (lv?.maxroll) $('#lvopen').onclick = () => { ui.variantName = 'level'; openBuildPage(lv.maxroll); };
  let list = data.builds.filter(b => b.cls === S.cls);
  if (ui.type !== 'Tutte') list = list.filter(b => ui.type === 'Leveling' ? /level/i.test(b.type) : !/level/i.test(b.type));
  if (ui.src !== 'all') list = list.filter(b => b.src === ui.src);
  const tv = { 'S+': 0, S: 1, A: 2, B: 3, C: 4, D: 5 };
  list.sort((a, b) => ((a.fromSeason ? 1 : 0) - (b.fromSeason ? 1 : 0)) || ((tv[a.tier] ?? 9) - (tv[b.tier] ?? 9)) || ((a.src === 'Maxroll' ? 0 : 1) - (b.src === 'Maxroll' ? 0 : 1)) || ((b.likes || 0) - (a.likes || 0)) || ((b.updated || 0) - (a.updated || 0)));
  $('#bcount').textContent = `${list.length} build`;
  if (!list.length) { $('#blist').innerHTML = `<div class="empty" style="grid-column:1/-1">${IC.search}<span>Nessuna build con questi filtri</span></div>`; return; }
  $('#blist').innerHTML = list.map((b, i) => buildCard(b, i)).join('');
  $$('#blist .bcard').forEach(el => el.onclick = ev => { const b = list[+el.dataset.i]; if (ev.target.closest('.bstar')) { toggleSave(b); el.querySelector('.bstar').classList.toggle('on', isSaved(b.url)); renderSaved($('#saved')); return; } openBuildPage(b); });
};
function buildCard(b, i) {
  const sv = savedOf(b.url);
  const changed = sv && sv.seen && b.updated && b.updated > sv.seen;
  return `<div class="bcard glass" data-i="${i}">${tierBadge(b.tier, b.cls)}<div class="t"><b>${esc(b.title)}</b><div class="meta"><span class="badge src-${b.src}">${b.src}</span>${b.fromSeason ? `<span class="badge">S${b.fromSeason}</span>` : ''}<span class="badge ${/level/i.test(b.type) ? 'lv' : 'eg'}">${esc(b.type)}</span>${b.author ? `<span>${esc(b.author)}</span>` : ''}${b.likes ? `<span>♥ ${b.likes}</span>` : ''}</div>${b.updated ? `<div class="upd ${changed ? 'new' : ''}">${changed ? '● aggiornata ' : 'aggiornata '}${dateLbl(b.updated)}</div>` : ''}</div><button class="bstar ${sv ? 'on' : ''}" title="${sv ? 'Togli dalle build salvate' : 'Salva questa build'}">${IC.star}</button></div>`;
}

// ---------- build salvate (con variante e scheda scelte), solo su questo PC ----------
const savedList = () => S.st.settings?.savedBuilds || [];
const savedOf = url => savedList().find(x => x.url === url);
const isSaved = url => !!savedOf(url);
async function saveList(list) { try { S.st.settings = await post('/api/settings', { savedBuilds: list }); } catch (e) { toast(e.message); } }
function toggleSave(b) {
  let list = savedList().slice();
  if (isSaved(b.url)) { list = list.filter(x => x.url !== b.url); toast('Build tolta dalle salvate'); }
  else { const ui = S.ui.build || {}; list.unshift({ url: b.url, src: b.src, title: b.title, cls: b.cls, type: b.type, season: b.fromSeason || S.season, profile: b.profile || '', variant: ui.openUrl === b.url ? (ui.variant || 0) : 0, tab: ui.openUrl === b.url ? (ui.btab || 'gear') : 'gear', at: Math.floor(Date.now() / 1000), seen: b.updated || 0 }); toast('Build salvata'); }
  S.st.settings.savedBuilds = list;
  saveList(list);
}
let savedT;
function touchSaved(url, patch) {
  const list = savedList().map(x => x.url === url ? { ...x, ...patch } : x);
  if (JSON.stringify(list) === JSON.stringify(savedList())) return;
  S.st.settings.savedBuilds = list;
  clearTimeout(savedT); savedT = setTimeout(() => saveList(list), 800);
}
async function savedWithUpdates() {
  const list = savedList();
  const seasons = [...new Set(list.map(x => x.season))];
  const by = {};
  for (const n of seasons) { try { (await getBuilds(n)).builds.forEach(b => { by[b.url] = b; }); } catch { } }
  // la prima data conosciuta fa da riferimento: da lì in poi ogni modifica viene segnalata
  list.forEach(x => { if (!x.seen && by[x.url]?.updated) touchSaved(x.url, { seen: by[x.url].updated }); });
  return list.map(x => ({ ...x, cur: by[x.url] || null, changed: !!(x.seen && by[x.url]?.updated && by[x.url].updated > x.seen) }));
}
async function renderSaved(el, compact) {
  if (!el) return;
  const list = savedList();
  if (!list.length) { el.innerHTML = compact ? '<div class="muted">Nessuna build salvata: usa la stella ☆ su una build.</div>' : ''; return; }
  const rows = await savedWithUpdates();
  const n = rows.filter(r => r.changed).length;
  el.innerHTML = `${compact ? '' : `<div class="lbl" style="margin:0 2px 8px">Le tue build salvate · ${rows.length}${n ? ` · <span style="color:var(--acc2)">${n} aggiornate</span>` : ''}</div>`}<div class="${compact ? 'mini' : 'savedrow'}">${rows.map((r, i) => `<div class="${compact ? 'mrow' : 'sbuild glass'} ${r.changed ? 'chg' : ''}" data-i="${i}">${compact ? '' : `<span class="ci">${clsIcon(r.cls, 18)}</span>`}<div class="t"><b>${esc(r.title)}</b><small>${esc(r.src)} · ${esc(clsName(r.cls))} · S${r.season}${r.cur?.updated ? ` · ${r.changed ? '<span class="nw">aggiornata ' + dateLbl(r.cur.updated) + '</span>' : 'agg. ' + dateLbl(r.cur.updated)}` : ''}${r.note ? `<br><i class="snote">${esc(r.note.length > 60 ? r.note.slice(0, 60) + '…' : r.note)}</i>` : ''}</small></div></div>`).join('')}</div>`;
  $$('[data-i]', el).forEach(d => d.onclick = () => { const r = rows[+d.dataset.i]; const b = r.cur || { src: r.src, title: r.title, cls: r.cls, type: r.type, url: r.url, profile: r.profile }; if (r.season !== S.season) { S.season = r.season; renderSeasonPick(); } setCls(r.cls); openBuildPage(b); });
}

function openBuildPage(b) {
  const ui = S.ui.build ||= { type: 'Endgame', src: 'all', variant: 0 };
  ui.cur = b;
  if (ui.openUrl !== b.url) { ui.variant = 0; ui.btab = 'gear'; ui.variantFor = null; }
  const sv = savedOf(b.url);
  if (sv && ui.openUrl !== b.url) { ui.variant = sv.variant || 0; ui.btab = sv.tab || 'gear'; ui.variantFor = b.url; }
  ui.openUrl = b.url;
  location.hash = '#/b/' + encodeURIComponent(b.url);
}
PAGES.b = async main => {
  const ui = S.ui.build ||= { type: 'Endgame', src: 'all', variant: 0 };
  const url = decodeURIComponent(location.hash.replace(/^#\/b\//, ''));
  let b = keyU(ui.cur?.url) === keyU(url) ? ui.cur : null;
  for (const n of [S.season, ...(S.st.seasons || []).map(x => x.n)]) { if (b) break; try { const d = await getBuilds(n); if (n === S.season) ui.all = d.builds; b = d.builds.find(x => keyU(x.url) === keyU(url)); } catch { } }
  if (!b) { const sv = savedList().find(x => keyU(x.url) === keyU(url)); if (sv) b = { src: sv.src, title: sv.title, cls: sv.cls, type: sv.type, url: sv.url, profile: sv.profile }; }
  if (!b) { location.hash = '#/build'; return; }
  if (ui.openUrl !== url) openBuildPage(b);
  ui.cur = b;
  main.innerHTML = `<div class="page"><div class="row" style="gap:8px;margin-bottom:12px"><button class="btn" id="bback">← Torna alle build</button><span class="muted" style="font-size:12.5px">${esc(clsName(b.cls))} · stagione ${b.fromSeason || S.season}</span></div><div class="glass bdetail" id="bdet"></div></div>`;
  $('#bback').onclick = () => { location.hash = '#/build'; };
  const sv = savedOf(url);
  if (sv && b.updated && b.updated > (sv.seen || 0)) { if (sv.seen) toast('Questa build è stata aggiornata il ' + dateLbl(b.updated)); touchSaved(url, { seen: b.updated }); }
  showBuild(b);
};

// ---------------------------------------------------------------- tier list
const TIERORD = ['S+', 'S', 'A', 'B', 'C', 'D', 'F', '?'];
const tierKind = r => /speed/i.test(r.tierNote || '') ? 'speed' : (/\d/.test(r.tierNote || '') ? 'push' : '');
PAGES.tier = async main => {
  const ui = S.ui.tier ||= { list: 'tier', cls: '', kind: '', q: '' };
  main.innerHTML = `<div class="page">
    <div class="hdr"><div><h1 class="pt">Tier list</h1><p class="sub" style="margin:0" id="tsub">Rob2628 · stagione ${S.season || '—'}</p></div><span class="sp"></span><button class="btn ghost" id="tsheet">${IC.ext} Foglio originale</button></div>
    <div class="row wrap" style="gap:8px;margin-bottom:10px">
      <div class="seg" id="tlist">${[['tier', 'Endgame'], ['leveling', 'Leveling']].map(([v, l]) => `<button data-v="${v}" class="${ui.list === v ? 'on' : ''}">${l}</button>`).join('')}</div>
      <div class="seg" id="tkind">${[['', 'Tutte'], ['push', 'Push (Fossa)'], ['speed', 'Speedfarm']].map(([v, l]) => `<button data-v="${v}" class="${ui.kind === v ? 'on' : ''}">${l}</button>`).join('')}</div>
      <label class="gsearch" style="flex:0 1 280px"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="M20 20l-4-4"/></svg><input id="tq" placeholder="Cerca una build…" value="${esc(ui.q)}"></label>
    </div>
    <div class="seg tcls" id="tcls" style="margin-bottom:14px"><button data-v="" class="${!ui.cls ? 'on' : ''}">Tutte le classi</button>${CLASSES.map(([c]) => `<button data-v="${c}" class="${ui.cls === c ? 'on' : ''}">${clsIcon(c, 16)} ${esc(clsName(c))}</button>`).join('')}</div>
    <div id="tbody"><div class="skel" style="height:400px"></div></div></div>`;
  let data;
  try { data = await getBuilds(S.season); } catch (e) { $('#tbody').innerHTML = noData(e.message); return; }
  const rob = data.rob || {};
  $('#tsub').textContent = `Rob2628 · ${rob.sheet || 'tier list'} · stagione ${S.season}${rob.wip ? ' · ancora in lavorazione' : ''}`;
  $('#tsheet').onclick = () => openLink(rob.source || 'https://docs.google.com/spreadsheets/d/1bN8C_Fe9nrWv__jUc_Vdbl220g4NM0mlux-ga_s_-5s/htmlview');
  const findBuild = (r, lvl) => (data.builds || []).find(b => r.url && b.url && keyU(b.url) === keyU(r.url)) || (data.builds || []).find(b => b.rank === r.rank && b.tier === r.tier && b.cls === r.cls && /level/i.test(b.type) === lvl);
  const draw = () => {
    const lvl = ui.list === 'leveling';
    let rows = (lvl ? rob.leveling : rob.tier) || [];
    if (ui.cls) rows = rows.filter(r => r.cls === ui.cls);
    if (ui.kind) rows = rows.filter(r => tierKind(r) === ui.kind);
    const q = norm(ui.q);
    if (q) rows = rows.filter(r => norm(r.build).includes(q) || norm(r.info).includes(q) || norm(clsName(r.cls)).includes(q));
    if (!rows.length) { $('#tbody').innerHTML = `<div class="glass empty">${IC.search}<span>${(lvl ? rob.leveling : rob.tier)?.length ? 'Nessuna build con questi filtri' : 'La tier list di questa stagione non è ancora uscita'}</span></div>`; return; }
    const groups = {};
    rows.forEach(r => { (groups[r.tier || '?'] ||= []).push(r); });
    const order = Object.keys(groups).sort((a, b) => (TIERORD.indexOf(a) + 99 * (TIERORD.indexOf(a) < 0)) - (TIERORD.indexOf(b) + 99 * (TIERORD.indexOf(b) < 0)));
    $('#tbody').innerHTML = order.map(t => `<div class="trow glass"><div class="tlab"><span class="${tierCls(t === '?' ? 'q' : t)}">${t === '?' ? '—' : esc(t)}</span><small>${groups[t].length}</small></div><div class="tcards">${groups[t].map(r => { const b = findBuild(r, lvl); return `<div class="tcard" data-k="${esc(r.cls + '|' + r.rank)}"><span class="ci">${clsIcon(r.cls, 20)}</span><div class="t"><b>${esc(r.build)}</b><small>${esc(clsName(r.cls))}${r.tierNote ? ` · <em>${esc(r.tierNote)}</em>` : ''}</small>${r.info ? `<p>${esc(r.info)}</p>` : ''}</div>${b ? `<span class="badge src-${b.src}">${b.src}</span>` : (r.url ? `<span class="badge">guida</span>` : '')}</div>`; }).join('')}</div></div>`).join('');
    $$('.tcard', $('#tbody')).forEach(el => el.onclick = () => { const [c, rk] = el.dataset.k.split('|'); const r = rows.find(x => x.cls === c && String(x.rank) === rk); if (!r) return; const b = findBuild(r, lvl); setCls(r.cls); if (b) openBuildPage(b); else if (r.url) openGuide(r.url); });
  };
  draw();
  $$('#tlist button').forEach(b => b.onclick = () => { ui.list = b.dataset.v; $$('#tlist button').forEach(x => x.classList.toggle('on', x === b)); draw(); });
  $$('#tkind button').forEach(b => b.onclick = () => { ui.kind = b.dataset.v; $$('#tkind button').forEach(x => x.classList.toggle('on', x === b)); draw(); });
  $$('#tcls button').forEach(b => b.onclick = () => { ui.cls = b.dataset.v; $$('#tcls button').forEach(x => x.classList.toggle('on', x === b)); draw(); });
  $('#tq').oninput = e => { ui.q = e.target.value; draw(); };
};
const keyU = u => String(u || '').replace(/[?#].*$/, '').replace(/\/$/, '');

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
// ---------------------------------------------------------------- scheda build (come su D4Builds)
const SA = 'https://sunderarmor.com/DIABLO4/';
const IMG = u => !u ? '' : (window.DEMO || !/^https:\/\/sunderarmor\.com\//.test(u) ? u : '/api/img?u=' + encodeURIComponent(u));
const snake = s => String(s || '').toLowerCase().replace(/[’]/g, "'").replace(/\s*\(.*?\)\s*/g, '').trim().replace(/[\s-]+/g, '_');
const SLOTICON = { 'helm': 'helm', 'chest armor': 'chest_armor', 'gloves': 'gloves', 'pants': 'pants', 'boots': 'boots', 'amulet': 'amulet', 'ring 1': 'ring_1', 'ring 2': 'ring_2', 'ring': 'ring_1', 'offhand': 'offhand', 'focus': 'offhand', 'shield': 'offhand', 'totem': 'offhand' };
const slotIcon = slot => SA + 'Uniques/' + (SLOTICON[String(slot).toLowerCase()] || 'weapon') + '.png';
const isWeapon = s => /weapon|two-handed|bludgeon|slash|dual|ranged|polearm|sword|mace|axe|staff|bow|dagger|scythe|wand|glaive|flail/i.test(s || '');
const KINDCLS = k => ({ mythic: 'mythic', unique: 'unique', legendary: 'legendary', runeword: 'runeword', talisman: 'talisman' }[k] || (k ? k : ''));

function gemIcon(n) {
  if (/^(ruby|sapphire|emerald|topaz|amethyst|diamond|skull)$/i.test(n)) return SA + 'Gems/grand-' + n.toLowerCase() + '.png';
  if (/^splinter of/i.test(n)) return SA + 'Gems/' + snake(n) + '.png';
  return '';
}
function viewFromPlanner(p, src) {
  const variants = (p.variants || []).map(v => {
    const gear = [], charms = [], extra = [], prime = [];
    let runes = [], seal = null;
    for (const g of v.gear || []) {
      if (g.typeEn === 'Active Runes') { runes = (g.sockets || []).map(s => { const [it, en] = s.split('|'); return { n: en || it, it }; }); continue; }
      if (g.kind === 'talisman') { const o = { n: g.en || g.name, it: g.name }; if (/seal/i.test(g.typeEn || '')) seal = o; else charms.push(o); continue; }
      if (!g.type && !g.typeEn) { if (g.name) extra.push({ n: g.en || g.name, it: g.name }); continue; }
      if (/seasonal power|prime evil/i.test(g.typeEn || g.type || '')) { prime.push({ n: g.en || g.name, it: g.name, i: SA + 'PrimeEvilPowers/' + snake(g.en || g.name).replace(/_/g, '-') + '.png' }); continue; }
      const en = g.en || g.name;
      const aspect = g.aspectEn || g.aspect;
      const asp = /aspect/i.test(en);
      const icon = asp ? SA + 'Codex/1/offensive.png' : (g.kind === 'unique' || g.kind === 'mythic' || g.kind === 'runeword' ? SA + 'Uniques/2/' + snake(en) + '.png' : slotIcon(g.typeEn || g.type));
      gear.push({ slot: g.typeEn || g.type, slotIt: g.type, n: en, it: g.name, k: asp ? 'legendary' : g.kind, i: icon, fb: slotIcon(g.typeEn || g.type), asp: aspect ? { n: g.aspectEn || aspect, it: g.aspect } : null, temp: g.tempered, gems: (g.sockets || []).map(s => { const [it, e] = s.split('|'); return { n: e || it, it, i: gemIcon(e || it) }; }) });
    }
    return { name: v.name, gear, charms, seal, runes, extra, prime, skills: (v.skills || []).map(s => ({ n: s.en || s.name, it: s.name, i: SA + 'Skills/VoH2/' + snake(s.en || s.name) + '.png' })), mxParagon: v.paragon || null };
  });
  return { src, variants, notes: null, tr: {}, filters: p.filters || [], plannerId: p.id, plannerUrl: p.url };
}
function viewFromFull(f) { (f.variants || []).forEach(v => { v.skills = (v.skills || []).filter(x => x && x.n && !/dropdown arrow/i.test(x.n) && !/\/UI\//.test(x.i || '')); }); return { src: 'd4b', from: f.src || 'd4b', variants: f.variants || [], notes: f.notes, notesTitle: f.notesTitle, tr: f.tr || {}, head: f.head || {}, at: f.src ? 0 : f.at, filters: f.filters || [], plannerId: f.planner || '', plannerUrl: f.planner ? 'https://maxroll.gg/d4/planner/' + f.planner : '' }; }
const fullCache = {};
const getFull = (key, url) => (fullCache[key] ||= api(url).catch(e => { delete fullCache[key]; throw e; }));
async function viewOf(b) {
  // formato completo (come D4Builds) per tutte le fonti; se non c'è ancora, la versione ridotta
  if (b.profile) { try { return viewFromFull(await getFull('mx:' + b.profile, '/api/fullbuild?mx=' + encodeURIComponent(b.profile))); } catch { return viewFromPlanner(await getPlanner(b.profile), 'maxroll'); } }
  try { return viewFromFull(await getFull('u:' + b.url, '/api/fullbuild?u=' + encodeURIComponent(b.url))); }
  catch (e) {
    if (b.src === 'D4Builds') { try { const v = viewFromPlanner(await (plannerCache['d:' + b.url] ||= api('/api/d4bbuild?u=' + encodeURIComponent(b.url)).catch(e2 => { delete plannerCache['d:' + b.url]; throw e2; })), 'd4b'); v.partial = 1; return v; } catch { throw e; } }
    const v = viewFromPlanner(await (plannerCache['m:' + b.url] ||= api('/api/mobabuild?u=' + encodeURIComponent(b.url)).catch(e2 => { delete plannerCache['m:' + b.url]; throw e2; })), 'moba'); v.partial = 1; return v;
  }
}
function TN(view, en, it) { if (lang() === 'en') return en || it || ''; return it || view.tr[en] || en || ''; }
function TSUB(view, en, it) { const main = TN(view, en, it); return main !== en && en ? en : ''; }
function icoImg(u, cls, alt, fb) { return u ? `<img class="${cls || ''}" src="${esc(IMG(u))}" alt="${esc(alt || '')}" loading="lazy" ${fb ? `data-fb="${esc(IMG(fb))}"` : ''} onerror="if(this.dataset.fb&&this.src!==location.origin+this.dataset.fb&&!this.dataset.f){this.dataset.f=1;this.src=this.dataset.fb}else{this.style.visibility='hidden'}">` : ''; }

const BTABS = [['gear', 'Equipaggiamento e abilità', SA + 'UI/gear_skills.png'], ['tree', 'Albero abilità', SA + 'UI/skill_tree.png'], ['paragon', 'Paragon', SA + 'UI/paragon.png'], ['merc', 'Mercenari', SA + 'UI/characters_icon.png'], ['war', 'Piani di guerra', SA + 'UI/warplans.png'], ['notes', 'Note', SA + 'UI/notes.png']];

async function showBuild(b) {
  TIPS = [];
  const el = $('#bdet'), ui = S.ui.build;
  const head = `<div class="bhead">${tierBadge(b.tier, b.cls)}<div style="flex:1;min-width:0"><h2 class="title">${esc(b.title)}</h2><div class="row wrap" style="gap:6px;margin-top:6px"><span class="badge src-${b.src}">${b.src}</span>${b.fromSeason ? `<span class="badge" title="La stagione ${S.season} non ha ancora build su ${b.src}">stagione ${b.fromSeason}</span>` : ''}<span class="badge ${/level/i.test(b.type) ? 'lv' : 'eg'}">${esc(b.type)}</span>${b.author ? `<span class="muted" style="font-size:12.5px">di ${esc(b.author)}</span>` : ''}${b.updated ? `<span class="muted" style="font-size:12.5px">· aggiornata ${dateLbl(b.updated)}</span>` : ''}</div></div><button class="btn ghost" id="bcmp" title="Aggiungi al confronto">${cmpHas(b.url) ? '⇄ Nel confronto' : '⇄ Confronta'}</button><button class="btn bsave ${isSaved(b.url) ? 'on' : ''}" id="bsave">${IC.star} ${isSaved(b.url) ? 'Salvata' : 'Salva build'}</button><button class="btn" id="bopen">${IC.ext} Apri guida</button></div>
    <div id="bnote"></div>
    ${b.tier || b.info ? `<div class="robinfo"><b>Rob2628:</b> ${b.tier ? `tier ${esc(b.tier)}${b.tierNote ? ' (' + esc(b.tierNote) + ')' : ''}` : ''}${b.info ? ' · ' + esc(b.info) : ''}</div>` : ''}`;
  const bindOpen = () => { const o = $('#bopen'); if (o) o.onclick = () => openGuide(b.url); const cb = $('#bcmp'); if (cb) cb.onclick = () => { cmpToggle(b); cb.textContent = cmpHas(b.url) ? '⇄ Nel confronto' : '⇄ Confronta'; }; bindNote(b); const sb = $('#bsave'); if (sb) sb.onclick = () => { toggleSave(b); sb.classList.toggle('on', isSaved(b.url)); sb.innerHTML = `${IC.star} ${isSaved(b.url) ? 'Salvata' : 'Salva build'}`; setTimeout(() => bindNote(b), 50); }; };
  el.innerHTML = head + `<div class="skel" style="height:420px;margin-top:16px"></div>`;
  bindOpen();
  const token = (ui._tok = (ui._tok || 0) + 1);
  let view = null, note = '';
  try {
    try { view = await viewOf(b); if (view.partial) note = 'Per ora c\'è solo l\'equipaggiamento: le altre schede (albero, paragon, mercenari, note) arrivano col prossimo aggiornamento dei dati.'; }
    catch (e) { const sim = similarMaxroll(b); if (sim) { view = await viewOf(sim); note = `${esc(e.message)}. Intanto qui sotto vedi la guida Maxroll più simile, <b>${esc(sim.title)}</b>.`; } else throw e; }
  } catch (e) { note = 'Dettagli non disponibili: ' + esc(e.message); }
  if (token !== ui._tok) return;
  ui.view = view;
  if (!view || !view.variants.length) {
    el.innerHTML = head + `<div class="empty" style="padding:30px">${IC.doc}<span>${note || 'Dettagli non disponibili.'}<br>“Apri guida” la apre in una finestra di Hellrise.</span></div>`;
    bindOpen();
    return;
  }
  if (ui.variantName) { const i = view.variants.findIndex(v => (v.name || '').toLowerCase().includes(ui.variantName)); if (i >= 0) ui.variant = i; ui.variantName = null; }
  else if (ui.variantFor !== b.url) { // build endgame: si parte dalla variante endgame, non dal leveling
    ui.variantFor = b.url;
    if (!/level/i.test(b.type) && /level/i.test(view.variants[ui.variant]?.name || '')) { const i = view.variants.findIndex(v => /endgame/i.test(v.name || '')); ui.variant = i >= 0 ? i : view.variants.findIndex(v => !/level/i.test(v.name || '')); if (ui.variant < 0) ui.variant = 0; }
  }
  if (!view.variants[ui.variant]) ui.variant = 0;
  const v = view.variants[ui.variant];
  const has = {
    gear: true,
    tree: !!(v.tree && v.tree.nodes?.length),
    paragon: !!(v.paragon?.boards?.length || v.mxParagon?.length),
    merc: !!(v.merc && (v.merc.slots?.length || v.merc.items?.length)),
    war: !!(v.war || []).some(w => w.tree),
    notes: !!(view.notes && view.notes.length),
  };
  if (!has[ui.btab]) ui.btab = 'gear';
  el.innerHTML = head + (note ? `<div class="robinfo" style="background:rgba(120,160,255,.07);border-color:rgba(120,160,255,.2)">${note}</div>` : '') + `
    <div class="btabs">${BTABS.filter(t => has[t[0]]).map(t => `<button data-t="${t[0]}" class="${ui.btab === t[0] ? 'on' : ''}">${icoImg(t[2], 'ti')}${t[1]}</button>`).join('')}</div>
    <div class="bvar">${view.variants.length > 1 ? `<select id="bvsel">${view.variants.map((x, i) => `<option value="${i}" ${i === ui.variant ? 'selected' : ''}>${esc(x.name || 'Variante ' + (i + 1))}</option>`).join('')}</select>` : `<span class="bvname">${esc(v.name || 'Build')}</span>`}${view.head?.season ? `<span class="muted">${esc(view.head.season.replace('Season', 'Stagione'))}</span>` : ''}<span class="sp"></span>${view.at ? `<span class="muted" style="font-size:11.5px">letta da D4Builds il ${dateLbl(view.at)}</span>` : ''}${view.from && view.from !== 'd4b' ? `<span class="muted" style="font-size:11.5px">da ${view.from === 'maxroll' ? 'Maxroll' : 'Mobalytics'}, nel formato di D4Builds</span>` : ''}</div>
    <div id="btab"></div>`;
  bindOpen();
  $$('.btabs button', el).forEach(x => x.onclick = () => { ui.btab = x.dataset.t; $$('.btabs button', el).forEach(y => y.classList.toggle('on', y === x)); drawTab(); if (isSaved(b.url)) touchSaved(b.url, { tab: ui.btab }); });
  const vs = $('#bvsel'); if (vs) vs.onchange = () => { ui.variant = +vs.value; if (isSaved(b.url)) touchSaved(b.url, { variant: ui.variant }); showBuild(b); };
  const drawTab = () => {
    const box = $('#btab');
    if (ui.btab === 'tree') { box.innerHTML = treeView(view, v.tree, 'skill'); bindViewer(box); return; }
    if (ui.btab === 'paragon') {
      parView(box, view, v, b);
      return;
    }
    if (ui.btab === 'merc') { box.innerHTML = mercView(view, v.merc); return; }
    if (ui.btab === 'war') { box.innerHTML = warView(view, v.war); $$('.wtabs button', box).forEach(x => x.onclick = () => { ui.war = +x.dataset.w; drawTab(); }); bindViewer(box); return; }
    if (ui.btab === 'notes') { box.innerHTML = `<div class="bnotes">${notesHTML(view, view.notes)}</div>`; return; }
    box.innerHTML = gearView(view, v, b);
    $$('[data-f]', box).forEach(x => x.onclick = () => copy(view.filters[+x.dataset.f].code, 'Codice del filtro'));
    $('#mkmacro')?.addEventListener('click', () => { S.ui.macroDraft = draftFromBuild(b, { variants: [{ skills: v.skills.map(s => ({ name: TN(view, s.n, s.it), en: s.n })) }] }, { skills: v.skills.map(s => ({ name: TN(view, s.n, s.it), en: s.n })) }); location.hash = '#/macro'; });
    $('#planner')?.addEventListener('click', () => openGuide(view.plannerUrl));
    bindGearExtras(box, view, v, b, drawTab);
  };
  drawTab();
}

function gearView(view, v, b) {
  const L = ['helm', 'chest armor', 'gloves', 'pants', 'boots'], R = ['amulet', 'ring 1', 'ring 2', 'ring', 'offhand', 'focus', 'shield', 'totem'];
  const left = [], right = [], used = new Set();
  const g = v.gear || [];
  for (const s of L) g.forEach((x, i) => { if (!used.has(i) && String(x.slot).toLowerCase() === s) { left.push(x); used.add(i); } });
  const weapons = g.map((x, i) => [x, i]).filter(([x, i]) => !used.has(i) && isWeapon(x.slot));
  if (weapons[0]) { left.push(weapons[0][0]); used.add(weapons[0][1]); }
  for (const s of R) g.forEach((x, i) => { if (!used.has(i) && String(x.slot).toLowerCase() === s && !/^empty$/i.test(x.slot)) { right.push(x); used.add(i); } });
  g.forEach((x, i) => { if (!used.has(i)) { right.push(x); used.add(i); } });
  const slotHTML = (x, side) => {
    const empty = /^empty$/i.test(x.slot) || !x.n;
    const kind = /aspect/i.test(x.n || '') ? 'legendary' : (view.src === 'd4b' && x.k === 'mythic' && !view.tr['~m:' + x.n] ? 'unique' : (view.src === 'd4b' && !x.k ? 'legendary' : KINDCLS(x.k)));
    const own = !empty && ownedOf(b.url).includes(x.slot);
    return `<div class="gslot ${side} ${empty ? 'empty' : ''} ${own ? 'have' : ''}" ${!empty ? `data-tip="${regTip({ kind: 'gear', x, view })}"` : ''}>${!empty ? `<button class="own ${own ? 'on' : ''}" data-own="${esc(x.slot)}" title="${own ? 'Ce l\'hai già' : 'Segna come già trovato'}">✓</button>` : ''}<div class="gico k-${kind}">${icoImg(x.i, 'gimg', x.n, x.fb || slotIcon(x.slot))}${x.gems?.length ? `<div class="ggems">${x.gems.map(gm => `<span title="${esc(TN(view, gm.n, gm.it))}">${gm.i ? icoImg(gm.i, '', gm.n) : `<i>${esc((TN(view, gm.n, gm.it) || '?').slice(0, 2))}</i>`}</span>`).join('')}</div>` : ''}</div><div class="gtx"><b class="k-${kind}">${esc(empty ? 'Vuoto' : TN(view, x.n, x.it))}</b><small>${esc(TN(view, x.slot, x.slotIt))}</small>${x.asp ? `<small class="asp">${esc(TN(view, x.asp.n, x.asp.it))}</small>` : ''}</div></div>`;
  };
  const POSA = { 'top-left': -120, 'top-right': -60, 'mid-right': 0, 'bottom-right': 60, 'bottom-left': 120, 'mid-left': 180 };
  const charms = v.charms || [];
  const useCharms = charms.length || v.seal;
  const ring = useCharms ? charms.slice(0, 6) : (v.skills || []).slice(0, 8);
  const ringHTML = ring.map((p, i) => { const deg = useCharms ? (POSA[p.pos] ?? (i * 60 - 120)) : (i / ring.length) * 360 - 90; const a = deg * Math.PI / 180, r = 38; const my = p.tip?.my || /mythic/i.test(p.i || ''); return `<div class="rnode ${useCharms ? 'ch' : ''} ${my ? 'my' : ''}" ${useCharms && p.tip ? `data-tip="${regTip({ kind: 'charm', x: p, view })}"` : ''} style="left:${50 + r * Math.cos(a)}%;top:${50 + r * Math.sin(a)}%" title="${esc(TN(view, p.n, p.it))}${p.tip?.vals?.length ? ' · ' + esc(p.tip.vals.map(x => x.t).join(', ')) : ''}">${p.i ? icoImg(p.i, '', p.n) : `<i>${esc(TN(view, p.n, p.it).slice(0, 14))}</i>`}</div>`; }).join('')
    + (useCharms && v.seal ? `<div class="rnode seal ${v.seal.tip?.my ? 'my' : ''}" style="left:50%;top:50%" title="${esc(TN(view, v.seal.tip?.n || v.seal.n, v.seal.it))}">${v.seal.i ? icoImg(v.seal.i, '', v.seal.n) : `<i>${esc(TN(view, v.seal.n, v.seal.it))}</i>`}</div>` : '');
  const ringLabel = useCharms ? (v.seal ? '' : '<span>Talismani</span>') : '<span>Abilità</span>';
  const chip = (x, extraCls) => `<div class="chipi ${extraCls || ''}" title="${esc(TSUB(view, x.n, x.it))}">${x.i ? icoImg(x.i, '', x.n) : ''}<span>${esc(TN(view, x.n, x.it))}</span></div>`;
  const keys = ['LMB', 'RMB', '1', '2', '3', '4'];
  const sec = (title, body) => body ? `<div class="bsec"><div class="lbl">${title}</div>${body}</div>` : '';
  return `<div class="gboard">
      <div class="gcol">${left.map(x => slotHTML(x, 'l')).join('')}</div>
      <div class="gcenter"><div class="gring ${useCharms ? 'hex' : ''}">${ringLabel ? `<div class="rmid">${ringLabel}</div>` : ''}${ringHTML}</div><div class="glbl">${useCharms ? 'Sigillo e talismani' : 'Abilità'}</div></div>
      <div class="gcol">${right.map(x => slotHTML(x, 'r')).join('')}</div>
    </div>
    ${sec('Rune attive', v.runes?.length ? `<div class="chips">${v.runes.map(x => chip(x, 'rune')).join('')}</div>` : '')}
    ${sec('Sigillo e talismani', (v.seal || v.charms?.length) ? charmsHTML(view, v) : '')}
    ${sec(v.skillsTitle && !/^skills$/i.test(v.skillsTitle) ? esc(v.skillsTitle) : 'Barra delle abilità', v.skills?.length ? `<div class="bskills">${v.skills.slice(0, 6).map((s, i) => `<div class="bskill"><span class="key">${keys[i] || ''}</span>${icoImg(s.i, 'sico', s.n)}<b>${esc(TN(view, s.n, s.it))}</b></div>`).join('')}</div>` : '')}
    ${(v.extra || []).filter(x => x.items?.length).map(x => sec(x.title === 'Technique' ? 'Tecnica' : esc(x.title), `<div class="chips">${x.items.map(y => chip(y)).join('')}</div>`)).join('')}
    ${v.prime?.length ? sec('Poteri dei Primi Mali', `<div class="chips">${v.prime.map(x => chip(x)).join('')}</div>`) : ''}
    ${v.shards?.length ? sec(v.shardsTitle === 'Soul Shards' ? 'Frammenti d\'anima' : esc(v.shardsTitle || 'Frammenti'), `<div class="shards">${v.shards.map(sh => `<div class="shard ${sh.a ? 'on' : ''}"><div class="scat">${icoImg(sh.cat?.i, '', sh.cat?.n)}<b>${esc(TN(view, sh.cat?.n, ''))}</b></div><div class="snodes">${sh.nodes.map(n => `<span class="${n.a ? 'on' : ''}" title="${esc(n.n)}">${icoImg(n.i, '', n.n)}</span>`).join('')}</div></div>`).join('')}</div>`) : ''}
    ${v.extra?.length && !v.extra[0].items ? sec('Altro', `<div class="chips">${v.extra.map(x => chip(x)).join('')}</div>`) : ''}
    ${v.stats?.length ? sec('Priorità delle statistiche', `<div class="bstats">${v.stats.map(st => `<div class="bstat"><div class="h">${icoImg(st.i, '', st.slot)}<b>${esc(TN(view, st.slot, ''))}</b></div><ol>${st.list.map(x => `<li class="${x.tf ? 'tf' : ''}">${x.ga ? '<i class="ga" title="Affisso superiore">★</i>' : ''}${x.tp ? '<i class="tp" title="Tempra">⚒</i>' : ''}${x.tf ? '<i class="tfi" title="Trasfigurazione">✦</i>' : ''}<span>${esc(x.t)}</span></li>`).join('')}</ol></div>`).join('')}</div>`) : ''}
    ${view.filters?.length ? sec('Filtri loot · da importare in gioco', `<div class="grid" style="gap:8px">${view.filters.map((f, i) => `<div class="filter"><b>${esc(f.name)}</b><button class="btn sm" data-f="${i}">${IC.copy} Copia codice</button></div>`).join('')}</div>`) : ''}
    ${rotationHTML(view, v)}
    ${farmHTML(view, v, b)}
    ${lootFilterHTML(view, v)}
    <div class="row" style="margin-top:20px;gap:8px;flex-wrap:wrap">${v.skills?.length ? `<button class="btn pri" id="mkmacro">${IC.keyb} Suggerisci profilo macro</button>` : ''}${view.plannerId ? `<button class="btn ghost" id="planner">${IC.ext} Planner Maxroll</button>` : ''}</div>`;
}

function charmsHTML(view, v) {
  const list = [...(v.seal ? [{ ...v.seal, isSeal: 1 }] : []), ...(v.charms || [])];
  const sets = {};
  const card = x => {
    const t = x.tip || {};
    if (t.set?.n && !sets[t.set.n]) sets[t.set.n] = t.set;
    const my = t.my || /mythic/i.test(x.i || '');
    return `<div class="ccard ${x.isSeal ? 'seal' : ''} ${my ? 'my' : ''}"><div class="h">${x.i ? icoImg(x.i, '', x.n) : ''}<div><b>${esc(TN(view, t.n || x.n, x.it))}</b><small>${x.isSeal ? 'Sigillo' : (t.set?.n ? esc(TN(view, t.set.n, '')) : (my ? 'Talismano mitico' : 'Talismano'))}</small></div></div>
      ${t.vals?.length ? `<ul class="cvals">${t.vals.map(y => `<li class="${y.k === 'm' ? 'm' : ''}">${y.b ? `<em>${esc(TN(view, y.b, ''))}:</em> ` : ''}${esc(TN(view, y.t, ''))}</li>`).join('')}</ul>` : ''}
      ${t.eff?.length ? `<div class="ceff">${t.eff.map(e => `<p>${esc(TN(view, e, ''))}</p>`).join('')}</div>` : ''}</div>`;
  };
  const cards = list.map(card).join('');
  const setHTML = Object.values(sets).map(st => { st = { ...st, pieces: st.pieces.filter(p => !st.pieces.some(q => q !== p && q[0] !== p[0] && p[0].includes(q[0]))) }; const have = st.pieces.filter(p => p[1]).length; return `<div class="cset"><div class="h"><b>${esc(TN(view, st.n, ''))}</b><span class="badge">${have}/${st.pieces.length} pezzi</span></div><div class="cpieces">${st.pieces.map(p => `<span class="${p[1] ? 'on' : ''}">${esc(TN(view, p[0], ''))}</span>`).join('')}</div>${st.bon.map(b => `<div class="cbon ${b[2] ? 'on' : ''}"><em>${esc(b[0].replace('Set:', 'set:'))}</em> ${esc(TN(view, b[1], ''))}</div>`).join('')}</div>`; }).join('');
  return `<div class="ccards">${cards}</div>${setHTML ? `<div class="csets">${setHTML}</div>` : ''}`;
}
function notesHTML(view, notes) {
  if (!notes?.length) return '<div class="muted">Nessuna nota.</div>';
  return notes.map(line => {
    if (!Array.isArray(line)) return '';
    if (line.length === 1 && line[0]?.b) return `<h3>${esc(line[0].b)}</h3>`;
    return `<p>${line.map(x => typeof x === 'string' ? esc(x) : (x.b ? `<b>${esc(x.b)}</b>` : `<span class="nchip t-${x.t || 'u'}">${x.i ? icoImg(x.i, '', x.n) : ''}${esc(TN(view, x.n, ''))}</span>`)).join('')}</p>`;
  }).join('');
}

// visore con trascinamento e zoom (albero, paragon, piani di guerra)
function bindViewer(box) {
  $$('.viewer', box).forEach(vw => {
    const inner = $('.vin', vw), W = +vw.dataset.w, H = +vw.dataset.h;
    const fit = () => { s = Math.min(vw.clientWidth / W, vw.clientHeight / H, 1) * .96; x = (vw.clientWidth - W * s) / 2; y = (vw.clientHeight - H * s) / 2; };
    let s, x, y;
    fit();
    const mn = +vw.dataset.min || 0;
    if (s < mn) { // troppo piccolo: si parte ingranditi sulla parte scelta della build
      s = mn;
      const fx = vw.dataset.fx !== undefined ? +vw.dataset.fx : W / 2, fy = vw.dataset.fy !== undefined ? +vw.dataset.fy : H / 2;
      x = vw.clientWidth / 2 - fx * s; y = vw.clientHeight / 2 - fy * s;
    }
    const ap = () => { inner.style.transform = `translate(${x}px,${y}px) scale(${s})`; };
    ap();
    vw.onwheel = e => { e.preventDefault(); const r = vw.getBoundingClientRect(), mx = e.clientX - r.left, my = e.clientY - r.top; const ns = Math.max(.08, Math.min(2.5, s * (e.deltaY < 0 ? 1.15 : 1 / 1.15))); x = mx - (mx - x) * ns / s; y = my - (my - y) * ns / s; s = ns; ap(); };
    vw.onpointerdown = e => { if (e.button !== 0) return; vw.setPointerCapture(e.pointerId); const sx = e.clientX - x, sy = e.clientY - y; vw.classList.add('drag'); vw.onpointermove = ev => { x = ev.clientX - sx; y = ev.clientY - sy; ap(); }; vw.onpointerup = () => { vw.onpointermove = null; vw.classList.remove('drag'); }; };
    $('.vfit', vw.parentElement)?.addEventListener('click', () => { fit(); ap(); });
  });
}
const NSIZE = { c: 64, l: 74, d: 52, s: 46, k: 18 };
function treeView(view, t, kind) {
  if (!t?.nodes?.length) return '<div class="muted">Albero non disponibile.</div>';
  const pad = 60;
  let minX = 1e9, minY = 1e9, maxX = -1e9, maxY = -1e9;
  t.nodes.forEach(n => { minX = Math.min(minX, n[0]); minY = Math.min(minY, n[1]); maxX = Math.max(maxX, n[0]); maxY = Math.max(maxY, n[1]); });
  const W = maxX - minX + pad * 2, H = maxY - minY + pad * 2, ox = pad - minX, oy = pad - minY;
  const lines = (t.lines || []).map(l => `<line x1="${l[0] + ox}" y1="${l[1] + oy}" x2="${l[2] + ox}" y2="${l[3] + oy}" class="${l[4] ? 'on' : ''}"/>`).join('');
  const nodes = t.nodes.map(n => {
    const [x, y, k, alt, icon, bg, rank, on] = n, sz = NSIZE[k] || 46;
    const got = k === 'c' ? on : (rank ? !/^0\//.test(rank) : on);
    return `<div class="tn k${k} ${got ? 'on' : ''}" style="left:${x + ox}px;top:${y + oy}px;width:${sz}px;height:${sz}px" title="${esc(TN(view, alt, ''))}${rank ? ' · ' + esc(rank) : ''}">${bg ? `<img class="tbg" src="${esc(IMG(bg))}" alt="">` : ''}${icon && k !== 'k' ? `<img class="tic" src="${esc(IMG(icon))}" alt="">` : ''}${rank && got ? `<i>${esc(rank)}</i>` : ''}</div>`;
  }).join('');
  const picked = t.nodes.filter(n => n[2] !== 'c' && n[2] !== 'k' && n[3] && n[6] && !/^0\//.test(n[6]));
  let fx = W / 2, fy = H / 2, zmin = 0.5;
  if (picked.length) { const xs = picked.map(n => n[0] + ox), ys = picked.map(n => n[1] + oy); fx = (Math.min(...xs) + Math.max(...xs)) / 2; fy = (Math.min(...ys) + Math.max(...ys)) / 2; zmin = Math.max(.3, Math.min(.9, 880 / (Math.max(...xs) - Math.min(...xs) + 200), 540 / (Math.max(...ys) - Math.min(...ys) + 200))); }
  const groups = {};
  picked.forEach(n => { (groups[n[2] === 'l' ? 'big' : 'small'] ||= []).push(n); });
  return `<div class="vwrap"><div class="viewer" data-w="${W}" data-h="${H}" data-min="${zmin}" data-fx="${fx}" data-fy="${fy}"><div class="vin" style="width:${W}px;height:${H}px"><svg class="vlines" width="${W}" height="${H}">${lines}</svg>${nodes}</div></div><button class="btn sm vfit">Adatta alla finestra</button>${t.remaining && t.remaining !== '0' ? `<span class="vrem">Punti rimasti: ${esc(t.remaining)}</span>` : ''}</div>
    <div class="bsec"><div class="lbl">${kind === 'war' ? 'Nodi scelti' : 'Abilità e passive scelte'} · ${picked.length}</div><div class="chips">${picked.map(n => `<div class="chipi ${n[2] === 'l' ? 'big' : ''}">${icoImg(n[4], '', n[3])}<span>${esc(TN(view, n[3], ''))}</span><em>${esc(n[6])}</em></div>`).join('')}</div></div>`;
}
// ---------------------------------------------------------------- paragon (uguale per D4Builds, Maxroll e Mobalytics)
let PINFO = null;
const pinfo = () => PINFO ||= fetch('/api/pinfo').then(r => r.json()).catch(() => ({ glyphs: {}, nodes: {}, board: {} }));
const RARN = { 1: 'Common', 2: 'Magic', 3: 'Rare', 4: 'Legendary', 5: 'Gate', 6: 'Glyph', 7: 'Start' };
const RARIT = { Common: 'Normale', Magic: 'Magica', Rare: 'Rara', Legendary: 'Leggendaria', Glyph: 'Incavo per glifo', 'Paragon Glyph': 'Incavo per glifo', Gate: 'Porta', Start: 'Inizio', 'Paragon Starting': 'Inizio' };
const PSTAT = { Str: 'Forza', Dex: 'Destrezza', Int: 'Intelligenza', Will: 'Volontà', Damage: 'Danni', DamageToElite: 'Danni agli élite', HPPercent: 'Vita massima', Armor: 'Armatura', ResistanceAll: 'Resistenza a tutti gli elementi', Fury: 'Furia massima', FuryOnKill: 'Furia per uccisione', HealingBonus: 'Guarigione ricevuta', DamageWhileFortified: 'Danni mentre sei fortificato', CriticalDamage: 'Danni dei colpi critici', BerserkDamage: 'Danni in berserk', HPRegen: 'Rigenerazione della vita', MoveSpeedEliteKill: 'Velocità dopo aver ucciso un élite', AttackSpeed: 'Velocità d\'attacco', DamageBleed: 'Danni da sanguinamento', DamageToVulnerable: 'Danni ai vulnerabili', CCDurationReduction: 'Durata dei controlli ridotta', Dodge: 'Probabilità di schivata', Mana: 'Mana massimo', Energy: 'Energia massima', Essence: 'Essenza massima', Spirit: 'Spirito massimo', DamageReduction: 'Riduzione dei danni', Thorns: 'Spine', Barrier: 'Barriera', Overpower: 'Danni travolgenti', DamagePoison: 'Danni da veleno', DamageFire: 'Danni da fuoco', DamageCold: 'Danni da freddo', DamageLightning: 'Danni da fulmine', DamageShadow: 'Danni d\'ombra', DamagePhysical: 'Danni fisici', CooldownReduction: 'Riduzione del tempo di recupero' };
const pstat = k => PSTAT[k] || (k || '').replace(/([a-z])([A-Z])/g, '$1 $2');
// tutte le fonti nello stesso formato: fasi → tavole con caselle [riga, colonna, presa, chiave, icona, rarità] non ruotate
function parNorm(v) {
  if (v.paragon?.boards?.length) return { points: v.paragon.points, glyphIcons: Object.fromEntries((v.paragon.glyphs || []).map(g => [g.n, g.i])), steps: [{ name: '', boards: v.paragon.boards.map(b => ({ n: b.n, g: b.g, rot: b.r || 0, x: b.x, y: b.y, w: 21, tiles: b.tiles || [], stats: b.s })) }] };
  if (v.mxParagon?.length) return { steps: v.mxParagon.map(st => ({ name: st.name, boards: st.boards.map(bd => ({ n: bd.en || bd.name, g: bd.glyphEn || bd.glyph, lvl: bd.glyphLvl, rot: (bd.rotation || 0) * 90, x: Math.abs(bd.x) < 60 ? bd.x * 1260 : bd.x, y: Math.abs(bd.y) < 60 ? bd.y * 1260 : bd.y, w: bd.width, legend: bd.legend, tiles: [...(bd.rawCells || '')].map((ch, i) => ch === '0' ? null : [Math.floor(i / bd.width), i % bd.width, bd.rawSel?.[i] === '1' ? 1 : 0, bd.keys?.[i] || '', '', RARN[ch]]).filter(Boolean) })) })) };
  return null;
}
function tileIcon(t, b, P) {
  const [, , , key, ic, rar] = t;
  if (ic) return ic;
  const m = /^Generic_(?:Normal|Magic|Rare)_(\w+)$/.exec(key || '');
  if (m) return '3/' + m[1] + '.png';
  if (/Legendary/.test(key || '') || rar === 'Legendary') { const c = (key || '').split('_')[0] || S.ui.build.cls || ''; return c ? '3/' + c[0].toUpperCase() + c.slice(1) + '.png' : ''; }
  return '';
}
function tileInfo(t, b, view, PI) {
  const [, , on, key, , rar] = t;
  const r = rar === 'Paragon Glyph' ? 'Glyph' : rar;
  let name = '', desc = '';
  if (r === 'Legendary') {
    const nk = PI.board[(b.n || '').toLowerCase()] || (PI.nodes[key] ? key : '');
    const nd = PI.nodes[nk];
    if (nd) { name = lang() === 'en' ? nd.en : nd.it; desc = lang() === 'en' ? nd.descEn : nd.descIt; }
    else name = TN(view, b.n, '');
  } else if (r === 'Glyph') name = b.g ? 'Glifo: ' + TN(view, b.g, '') : 'Incavo per glifo (vuoto)';
  else if (r === 'Gate') name = 'Porta verso un\'altra tavola';
  else if (/Start/.test(r || '')) name = 'Nodo di partenza';
  else if (PI.nodes[key]) name = lang() === 'en' ? PI.nodes[key].en : PI.nodes[key].it;
  else { const m = /^Generic_(?:Normal|Magic|Rare)_(\w+)$/.exec(key || ''); name = pstat(m ? m[1] : key) || RARIT[r] || ''; }
  return { name, desc, rar: RARIT[r] || r || '', on };
}
function glyphCard(view, b, PI, P) {
  if (!b.g) return `<div class="pcard"><div class="lbl">Glifo</div><div class="muted">Nessun glifo su questa tavola.</div></div>`;
  const gi = PI.glyphs[(b.g || '').toLowerCase()] || Object.values(PI.glyphs).find(x => x.it === b.g);
  const en = lang() === 'en';
  const RK = { '-1': 'Bonus extra se il requisito è soddisfatto', 0: 'Bonus a portata', 1: 'Dal livello 15', 2: 'Dal livello 46 (leggendario)' };
  return `<div class="pcard glyph"><div class="lbl">Glifo</div><div class="gh">${P?.glyphIcons?.[b.g] ? `<img class="gico" src="${esc(IMG(P.glyphIcons[b.g]))}" alt="">` : '<i class="gico"></i>'}<div><b>${esc(gi ? (en ? gi.en : gi.it) : TN(view, b.g, ''))}</b>${b.lvl ? `<small>livello ${b.lvl}</small>` : ''}</div></div>
    ${gi ? `<div class="greq">Raggio 3 (4 dal livello 15)${gi.reqIt ? ` · requisito: <b>${esc(en ? gi.reqEn : gi.reqIt)}</b> a portata` : ''}</div>
    ${gi.bonus.map(x => `<div class="gb r${x.rank}"><small>${RK[x.rank] ?? ''}</small>${esc(en ? x.en : x.it)}</div>`).join('')}
    <div class="muted" style="font-size:11px;margin-top:6px">I valori con la freccia vanno dal livello 1 al livello 100 del glifo.</div>` : '<div class="muted">Dettagli del glifo non trovati nei dati di gioco.</div>'}</div>`;
}
function boardGrid(view, P, b, PI, k) {
  const w = b.w || 21, q = ((Math.round((b.rot || 0) / 90) % 4) + 4) % 4;
  const R = (r, c) => q === 1 ? [c, w - 1 - r] : q === 2 ? [w - 1 - r, w - 1 - c] : q === 3 ? [w - 1 - c, r] : [r, c];
  const tiles = b.tiles.map((t, i) => {
    const [r, c] = R(t[0], t[1]), rar = t[5] === 'Paragon Glyph' ? 'Glyph' : (/Start/.test(t[5] || '') ? 'Start' : t[5]);
    let ic = tileIcon(t, b, P);
    let src = '';
    if (rar === 'Glyph') src = P.glyphIcons?.[b.g] || '';
    else if (ic) src = SA + 'Paragon/' + (t[2] && !/_active\.png$/.test(ic) ? ic.replace(/\.png$/, '_active.png') : ic);
    return `<div class="pt2 ${t[2] ? 'on' : ''} k-${(rar || 'Common').toLowerCase()}" data-i="${i}" style="left:${c / w * 100}%;top:${r / w * 100}%;width:${100 / w}%;height:${100 / w}%">${src ? `<img src="${esc(IMG(src))}" alt="" loading="lazy" onerror="this.remove()">` : ''}</div>`;
  }).join('');
  const on = b.tiles.filter(t => t[2]).length;
  const legT = b.tiles.find(t => (t[5] === 'Legendary'));
  const leg = legT ? tileInfo(legT, b, view, PI) : null;
  return `<div class="pmain"><div class="pgwrap"><div class="pgrid" id="pgrid">${tiles}</div><div class="plegend2">${[['Normale', 'common'], ['Magica', 'magic'], ['Rara', 'rare'], ['Leggendaria', 'legendary'], ['Glifo', 'glyph'], ['Porta', 'gate']].map(([l, c]) => `<span><i class="pt2 on k-${c}"></i>${l}</span>`).join('')}<span><i class="pt2 k-common"></i>non presa</span></div></div>
    <aside class="pside">
      <div class="pcard"><div class="lbl">Tavola ${k + 1}</div><b class="pbn">${esc(TN(view, b.n, ''))}</b><div class="muted" style="font-size:12px">${on} punti${b.rot ? ` · ruotata di ${b.rot}°` : ''}</div>${b.stats?.length ? `<div class="pstats">${b.stats.map(x => { const [a, n] = x.split(' '); return `<span><b>${esc(n || '')}</b> ${esc(pstat(a))}</span>`; }).join('')}</div>` : ''}</div>
      ${leg ? `<div class="pcard leg"><div class="lbl">Nodo leggendario${legT[2] ? '' : ' · non preso'}</div><b>${esc(leg.name)}</b>${leg.desc ? `<p>${esc(leg.desc)}</p>` : ''}</div>` : ''}
      ${glyphCard(view, b, PI, P)}
      <div class="pcard" id="phover"><div class="lbl">Casella</div><div class="muted" style="font-size:12px">Passa il mouse su una casella per vederne i dettagli.</div></div>
    </aside></div>`;
}
async function parView(box, view, v, b) {
  const ui = S.ui.build;
  const P = parNorm(v);
  if (!P) { box.innerHTML = '<div class="muted">Paragon non disponibile.</div>'; return; }
  box.innerHTML = '<div class="skel" style="height:420px"></div>';
  const PI = await pinfo();
  if (!P.steps[ui.pstep || 0]) ui.pstep = 0;
  const st = P.steps[ui.pstep || 0];
  if (ui.pboard == null || ui.pboard >= st.boards.length) ui.pboard = 0;
  const strip = `<div class="pstrip"><button data-k="-1" class="${ui.pboard === -1 ? 'on' : ''}"><span class="n">⊞</span><div><b>Panoramica</b><small>${st.boards.length} tavole</small></div></button>${st.boards.map((bd, i) => `<button data-k="${i}" class="${ui.pboard === i ? 'on' : ''}"><span class="n">${i + 1}</span><div><b>${esc(TN(view, bd.n, ''))}</b><small>${bd.g ? esc(TN(view, bd.g, '')) : 'senza glifo'}</small></div></button>`).join('')}</div>`;
  const steps = P.steps.length > 1 ? `<div class="seg" id="pstep">${P.steps.map((x, i) => `<button data-s="${i}" class="${i === (ui.pstep || 0) ? 'on' : ''}">${esc(x.name || 'Fase ' + (i + 1))}</button>`).join('')}</div>` : '';
  const total = P.points || st.boards.reduce((a, bd) => a + bd.tiles.filter(t => t[2]).length, 0);
  box.innerHTML = `<div class="row wrap" style="margin-bottom:10px;gap:8px">${steps}<span class="muted" style="font-size:12.5px">${total} punti paragon</span><span class="sp"></span><button class="btn" id="toov">Paragon nell'overlay in gioco</button></div>${strip}<div id="pbody"></div>`;
  const body = $('#pbody', box);
  if (ui.pboard === -1) { body.innerHTML = paragonView(view, { points: P.points, boards: st.boards.map(bd => ({ ...bd, r: bd.rot })) }, P); bindViewer(body);
    let down = null;
    body.addEventListener('pointerdown', e => { down = [e.clientX, e.clientY]; });
    $$('.pbd', body).forEach(x => x.addEventListener('click', e => { if (down && Math.hypot(e.clientX - down[0], e.clientY - down[1]) > 6) return; ui.pboard = +x.dataset.k; parView(box, view, v, b); }));
  }
  else {
    const bd = st.boards[ui.pboard];
    body.innerHTML = boardGrid(view, P, bd, PI, ui.pboard);
    const hov = $('#phover', body);
    $('#pgrid', body).addEventListener('mouseover', e => {
      const el = e.target.closest('.pt2'); if (!el) return;
      const ti = tileInfo(bd.tiles[+el.dataset.i], bd, view, PI);
      hov.innerHTML = `<div class="lbl">Casella · ${esc(ti.rar)}${ti.on ? ' · <span style="color:var(--acc)">presa</span>' : ''}</div><b>${esc(ti.name)}</b>${ti.desc ? `<p>${esc(ti.desc)}</p>` : ''}`;
    });
  }
  $$('.pstrip button', box).forEach(x => x.onclick = () => { ui.pboard = +x.dataset.k; parView(box, view, v, b); });
  $$('#pstep button', box).forEach(x => x.onclick = () => { ui.pstep = +x.dataset.s; parView(box, view, v, b); });
  $('#toov', box).onclick = () => v.paragon ? paragonToOverlay('d4b:' + keyU(b.url) + '/', b.title, ui.variant || 0) : paragonToOverlay(view.plannerId, b.title, ui.variant, ui.pstep || 0);
}
const PTILE = 60;
const PRAR = { Common: '#5b534e', Magic: '#4f8bff', Rare: '#ffd24a', Legendary: '#ff7a2a', Glyph: '#c07bff', Gate: '#ddd' };
function paragonView(view, p, P) {
  const bs = p.boards;
  const pts = [];
  bs.forEach(b => { pts.push([b.x, b.y], [b.x + 21 * PTILE, b.y + 21 * PTILE]); });
  const minX = Math.min(...pts.map(q => q[0])) - 40, minY = Math.min(...pts.map(q => q[1])) - 70, maxX = Math.max(...pts.map(q => q[0])) + 40, maxY = Math.max(...pts.map(q => q[1])) + 40;
  const W = maxX - minX, H = maxY - minY;
  const boards = bs.map((b, bi) => `<div class="pbd" data-k="${bi}" title="Apri la tavola ${bi + 1}" style="left:${b.x - minX}px;top:${b.y - minY}px;transform:rotate(${b.r}deg)">${b.tiles.map(t => { const [r, c, a, alt, ic, rar] = t; return `<div class="ptile ${a ? 'on' : ''}" style="left:${c * PTILE}px;top:${r * PTILE}px;--rc:${PRAR[rar] || '#555'};transform:rotate(${-b.r}deg)" title="${esc(alt)}${rar ? ' · ' + esc(rar) : ''}">${ic ? `<img src="${esc(IMG(SA + 'Paragon/' + (a && !/_active\.png$/.test(ic) ? ic.replace(/\.png$/, '_active.png') : ic)))}" alt="">` : ''}</div>`; }).join('')}</div><div class="pbl" style="left:${b.x - minX + 21 * PTILE / 2}px;top:${b.y - minY - 4}px"><b>${bi + 1}</b> ${esc(TN(view, b.n, ''))}${b.g ? ` <em>(${esc(TN(view, b.g, ''))})</em>` : ''}</div>`).join('');
  return `<div class="vwrap"><div class="viewer tall" data-w="${W}" data-h="${H}" data-min="0.16" data-fx="${bs[0].x - minX + 630}" data-fy="${bs[0].y - minY + 630}"><div class="vin" style="width:${W}px;height:${H}px">${boards}</div></div><button class="btn sm vfit">Adatta alla finestra</button>${p.points ? `<span class="vrem">Punti: ${esc(p.points)}</span>` : ''}</div>
    <div class="bsec"><div class="lbl">Tavole e glifi</div><div class="pblist">${bs.map((b, i) => `<div class="pbi"><span class="n">${i + 1}</span><div><b>${esc(TN(view, b.n, ''))}</b>${b.g ? `<small>Glifo: ${esc(TN(view, b.g, ''))}</small>` : ''}${b.s?.length ? `<small class="muted">${esc(b.s.join(' · '))}</small>` : ''}${b.r ? `<small class="muted">Rotazione ${b.r}°</small>` : ''}</div></div>`).join('')}</div></div>`;
}
function mxParagonHTML(v) {
  const ui = S.ui.build;
  if (!v.mxParagon?.length) return '<div class="muted">Paragon non disponibile.</div>';
  if (!v.mxParagon[ui.pstep || 0]) ui.pstep = 0;
  return `<div class="row" style="margin:6px 0 4px"><div class="seg" id="pstep">${v.mxParagon.map((st, i) => `<button data-s="${i}" class="${i === (ui.pstep || 0) ? 'on' : ''}">${esc(st.name || 'Fase ' + (i + 1))}</button>`).join('')}</div><span class="sp"></span><button class="btn" id="toov">Paragon nell'overlay in gioco</button></div>${PLEGEND}<div class="pboards" id="pboards"></div>`;
}
function bindMxParagon(box, view, v, b) {
  const ui = S.ui.build;
  if (!v.mxParagon?.length) return;
  const drawSteps = () => { const st = v.mxParagon[ui.pstep || 0]; $('#pboards', box).innerHTML = st.boards.map((bd, i) => boardCard(bd, i)).join(''); $$('#pboards canvas', box).forEach((c, i) => drawBoard(c, st.boards[i])); };
  drawSteps();
  $$('#pstep button', box).forEach(x => x.onclick = () => { ui.pstep = +x.dataset.s; $$('#pstep button', box).forEach(y => y.classList.toggle('on', y === x)); drawSteps(); });
  $('#toov', box).onclick = () => paragonToOverlay(view.plannerId, b.title, ui.variant, ui.pstep || 0);
}
const MERCL = { Mercenary: 'Mercenario', Reinforcement: 'Rinforzo', Skill: 'Abilità', Opportunity: 'Opportunità' };
function mercView(view, m) {
  if (!m) return '<div class="muted">Nessun mercenario indicato.</div>';
  return `<div class="mslots">${(m.slots || []).map(s => `<div class="mslot"><div class="lbl">${esc(MERCL[s.l] || s.l)}</div><div class="mico">${icoImg(s.i, '', s.n)}</div><b>${esc(TN(view, s.n, ''))}</b></div>`).join('')}</div>
    ${m.items?.length ? `<div class="bsec"><div class="lbl">Abilità del mercenario scelte · ${m.items.length}</div><div class="chips">${m.items.map(x => `<div class="chipi ${x.l ? 'big' : ''}">${icoImg(x.i, '', x.n)}<span>${esc(TN(view, x.n, ''))}</span></div>`).join('')}</div></div>` : ''}`;
}
const WARIT = { Whispers: 'Sussurri', 'Nightmare Dungeons': 'Spedizioni dell\'incubo', Helltides: 'Maree infernali', Undercity: 'Sottocittà', 'Boss Lairs': 'Covi dei boss', 'Infernal Hordes': 'Orde infernali', Pits: 'Fosse' };
function warView(view, war) {
  const ui = S.ui.build;
  const list = (war || []).filter(w => w.tree);
  if (!list.length) return '<div class="muted">Nessun piano di guerra indicato.</div>';
  if (!list[ui.war || 0]) ui.war = 0;
  const w = list[ui.war || 0];
  return `<div class="wtabs seg">${list.map((x, i) => `<button data-w="${i}" class="${i === (ui.war || 0) ? 'on' : ''}">${icoImg(x.i, 'ti', x.n)}${esc(lang() === 'en' ? x.n : (WARIT[x.n] || x.n))}</button>`).join('')}</div>${treeView(view, w.tree, 'war')}`;
}

// ---------------------------------------------------------------- extra della scheda build
// finestrelle col dettaglio (come su D4Builds) al passaggio del mouse
let TIPS = [];
function regTip(t) { TIPS.push(t); return TIPS.length - 1; }
function tipHTML(t) {
  const { x, view } = t, p = x.tip || {};
  const T = s => esc(TN(view, s, ''));
  if (t.kind === 'charm') {
    return `<div class="th"><b class="${p.my ? 'k-mythic' : 'tl'}">${esc(TN(view, p.n || x.n, x.it))}</b><small>${p.set?.n ? T(p.set.n) : (p.my ? 'Talismano mitico' : 'Talismano')}</small></div>${(p.vals || []).map(v => `<div class="tv">${v.b ? `<em>${T(v.b)}:</em> ` : ''}${T(v.t)}</div>`).join('')}${(p.eff || []).map(e => `<p>${T(e)}</p>`).join('')}`;
  }
  const kind = /aspect/i.test(x.n || '') ? 'legendary' : (view.tr['~m:' + x.n] ? 'mythic' : 'unique');
  const sub = (p.sub || '').replace(/^Mythic Unique/, 'Unico mitico').replace(/^Unique/, 'Unico').replace(/^Legendary/, 'Leggendario').replace(/^Runeword/, 'Parola runica');
  return `<div class="th"><b class="k-${kind}">${esc(TN(view, x.n, x.it))}</b><small>${esc(lang() === 'en' ? (p.sub || x.slot) : (sub.replace(/(Helm|Chest Armor|Gloves|Pants|Boots|Amulet|Ring \d|Ring|Offhand|Weapon.*|Bludgeoning Weapon|Slashing Weapon|Dual-Wield Weapon \d)$/, m => TN(view, m, '')) || TN(view, x.slot, x.slotIt)))}</small></div>
    ${p.desc ? `<p>${T(p.desc)}</p>` : ''}
    ${(p.stats || []).length ? `<div class="tsec">${p.stats.map(s => `<div class="tv">${s.ga ? '<i class="ga">★</i> ' : '• '}${T(s.t)}</div>`).join('')}</div>` : ''}
    ${(p.temper || []).length ? `<div class="tsec">${p.temper.map(s => `<div class="tv tp">⚒ ${T(s)}</div>`).join('')}</div>` : ''}
    ${(p.eff || []).map(e => `<p class="eff">${T(e)}</p>`).join('')}
    ${(p.hor || []).map(e => `<div class="tv hor">${T(e)}</div>`).join('')}
    ${p.drop ? `<div class="tdrop">Cade da: <b>${esc(p.drop)}</b></div>` : ''}
    ${!x.tip && view.src === 'd4b' ? `<div class="muted" style="font-size:11.5px;margin-top:6px">Dettagli in arrivo col prossimo aggiornamento dei dati.</div>` : ''}${x.asp ? `<div class="tv">${esc(TN(view, x.asp.n, x.asp.it))}</div>` : ''}`;
}
(function tipEvents() {
  let box;
  document.addEventListener('mouseover', e => {
    const el = e.target.closest?.('[data-tip]');
    if (!el || e.target.closest('.own')) { if (box) box.style.display = 'none'; return; }
    const t = TIPS[+el.dataset.tip]; if (!t) return;
    if (!box) { box = document.createElement('div'); box.className = 'tipbox glass'; document.body.appendChild(box); }
    box.innerHTML = tipHTML(t); box.style.display = 'block';
    const r = el.getBoundingClientRect(), bw = box.offsetWidth, bh = box.offsetHeight;
    let x = r.right + 10, y = r.top;
    if (x + bw > innerWidth - 8) x = r.left - bw - 10;
    if (x < 8) x = 8;
    if (y + bh > innerHeight - 8) y = innerHeight - bh - 8;
    box.style.left = x + 'px'; box.style.top = Math.max(8, y) + 'px';
  });
  document.addEventListener('scroll', () => { if (box) box.style.display = 'none'; }, true);
})();

// pezzi che hai già (salvati con la build)
const ownedOf = url => savedOf(url)?.owned || [];
function toggleOwned(b, slot) {
  if (!isSaved(b.url)) toggleSave(b);
  const cur = ownedOf(b.url);
  touchSaved(b.url, { owned: cur.includes(slot) ? cur.filter(x => x !== slot) : [...cur, slot] });
}

// rotazione presa dalle note della build (D4Builds) o, se manca, dalla barra delle abilità
function rotationLines(view, v) {
  const out = [];
  const notes = view.notes || [];
  let on = false;
  for (const line of notes) {
    const head = Array.isArray(line) && line.length === 1 && line[0]?.b ? line[0].b : null;
    if (head) { if (on) break; on = /rotation|rotazione/i.test(head); continue; }
    if (!on || !Array.isArray(line)) continue;
    const txt = line.map(x => typeof x === 'string' ? x : (x.b || TN(view, x.n, ''))).join('').replace(/^\s*[-•]\s*/, '').trim();
    if (txt) out.push(txt);
  }
  if (out.length) return { lines: out.slice(0, 12), from: 'note' };
  const keys = ['LMB', 'RMB', '1', '2', '3', '4'];
  return { lines: (v.skills || []).slice(0, 6).map((s, i) => `${keys[i]}: ${TN(view, s.n, s.it)}`), from: 'barra' };
}
function rotationHTML(view, v) {
  const r = rotationLines(view, v);
  if (!r.lines.length) return '';
  return `<div class="bsec"><div class="row"><div class="lbl">Rotazione${r.from === 'barra' ? ' · dalla barra delle abilità' : ''}</div><span class="sp"></span><button class="btn sm" id="rotov">Rotazione nell'overlay</button></div><ol class="rotlist">${r.lines.map(l => `<li>${esc(l)}</li>`).join('')}</ol></div>`;
}

// cosa manca e dove farmarlo
function farmHTML(view, v, b) {
  const own = ownedOf(b.url);
  const miss = (v.gear || []).filter(x => x.n && !/^empty$/i.test(x.slot) && !own.includes(x.slot));
  const have = (v.gear || []).filter(x => own.includes(x.slot)).length;
  const rows = miss.map(x => {
    const asp = /aspect/i.test(x.n);
    let where = asp ? 'Aspetto: dal Codex del Potere (sotterranei) o da estrarre da un leggendario' : (x.tip?.drop ? x.tip.drop : '');
    if (!asp && !where) { const it = (S.lootIdx || {})[norm(x.n)]; if (it) where = [...new Set(it.sources.map(s => S.lootBoss?.[s.boss]?.name).filter(Boolean))].slice(0, 3).join(' · '); }
    return `<div class="frow"><span class="fslot">${esc(TN(view, x.slot, x.slotIt))}</span><b class="${asp ? 'k-legendary' : 'k-unique'}">${esc(TN(view, x.n, x.it))}</b><small>${esc(where || (asp ? '' : 'drop casuale (qualsiasi attività di fine gioco)'))}</small></div>`;
  }).join('');
  return `<div class="bsec"><div class="lbl">Cosa ti manca · ${have}/${(v.gear || []).filter(x => x.n && !/^empty$/i.test(x.slot)).length} pezzi trovati</div><div class="muted" style="font-size:12px;margin-bottom:8px">Segna con ✓ sull'icona i pezzi che hai già: qui restano quelli da farmare, con dove cercarli.</div>${rows ? `<div class="flist">${rows}</div>` : '<div class="muted">Hai tutto! 🎉</div>'}</div>`;
}
async function loadLootIdx() {
  if (S.lootIdx) return;
  try { const L = await getLoot(S.season); S.lootIdx = {}; (L.items || []).forEach(it => { S.lootIdx[norm(it.en)] = it; S.lootIdx[norm(it.name)] = it; }); S.lootBoss = Object.fromEntries((L.bosses || []).map(x => [x.id, x])); } catch { }
}

// filtro loot del gioco creato dalla build
function lootFilterHTML(view, v) {
  return `<div class="bsec"><div class="lbl">Filtro loot del gioco</div><div class="lfbox glass">
    <div class="muted" style="font-size:12.5px;line-height:1.5">Crea un filtro da importare in Diablo IV (Opzioni › Gameplay › Filtro bottino › Nuovo filtro › Importa): mette in risalto mitici, unici della build, i talismani dei set e i pezzi con gli affissi giusti per ogni slot.</div>
    <div class="row wrap" style="gap:12px;margin-top:10px"><label class="field" style="flex-direction:row;align-items:center;gap:8px"><span>Affissi giusti richiesti</span><select class="inp" id="lfmin"><option value="2">almeno 2</option><option value="3" selected>almeno 3</option><option value="4">almeno 4</option></select></label><label class="tog"><input type="checkbox" id="lfhide"><i></i><span><b>Nascondi il resto dell'equipaggiamento</b></span></label><span class="sp"></span><button class="btn pri" id="lfmake">Crea filtro</button></div>
    <div id="lfout"></div></div></div>`;
}
function lootReq(view, v, b) {
  const uniques = [], sets = new Set();
  (v.gear || []).forEach(x => { if (x.n && !/aspect/i.test(x.n) && !/^empty$/i.test(x.slot)) uniques.push(x.n); });
  (v.charms || []).forEach(c => { if (c.tip?.set?.n) sets.add(c.tip.set.n); else if (c.tip?.my || /mythic|unique/i.test(c.i || '')) uniques.push(c.n); });
  let slots = (v.stats || []).map(st => ({ slot: st.slot, stats: st.list.filter(x => !x.tf && !/^weapon type$/i.test(x.t)).map(x => x.t) }));
  if (!slots.length) slots = (v.gear || []).filter(x => x.tip?.stats?.length).map(x => ({ slot: x.slot, stats: x.tip.stats.map(s => s.t) }));
  return { name: b.title, uniques, sets: [...sets], slots, minAffix: +($('#lfmin')?.value || 3), hide: !!$('#lfhide')?.checked };
}

function bindGearExtras(box, view, v, b, redraw) {
  $$('.own', box).forEach(el => el.onclick = ev => { ev.stopPropagation(); toggleOwned(b, el.dataset.own); setTimeout(redraw, 50); });
  $('#rotov', box)?.addEventListener('click', async () => {
    const r = rotationLines(view, v);
    const o = { ...(S.st.settings.overlay || {}), rotation: { ...(S.st.settings.overlay?.rotation || {}), on: true }, rotTitle: b.title, rotLines: r.lines, enabled: true };
    try { S.st.settings = await post('/api/settings', { overlay: o }); toast('Rotazione impostata nell\'overlay'); } catch (e) { toast(e.message); }
  });
  $('#lfmake', box)?.addEventListener('click', async () => {
    const out = $('#lfout', box);
    out.innerHTML = '<div class="skel" style="height:60px;margin-top:10px"></div>';
    try {
      const r = await post('/api/lootfilter', lootReq(view, v, b));
      out.innerHTML = `<div class="lfres"><div class="row"><b>Filtro pronto · ${r.rules.length} regole</b><span class="sp"></span><button class="btn sm pri" id="lfcopy">${IC.copy} Copia codice</button></div><div class="chips" style="margin:8px 0">${r.rules.map(x => `<span class="badge">${esc(x)}</span>`).join('')}</div><textarea class="inp lfcode" readonly>${esc(r.code)}</textarea>${r.missing?.length ? `<div class="muted" style="font-size:11.5px;margin-top:6px">Non messe nel filtro (il gioco non le riconosce come affissi): ${esc(r.missing.join(', '))}</div>` : ''}${!(v.stats || []).length && !(v.gear || []).some(x => x.tip?.stats?.length) ? `<div class="muted" style="font-size:11.5px;margin-top:6px">Questa fonte non indica gli affissi per slot: il filtro mette in risalto solo mitici, unici e set.</div>` : ''}</div>`;
      $('#lfcopy', out).onclick = () => copy(r.code, 'Codice del filtro');
    } catch (e) { out.innerHTML = `<div class="muted" style="margin-top:8px">${esc(e.message)}</div>`; }
  });
  if (!S.lootIdx) loadLootIdx().then(() => { if ($('.flist', box)) redraw(); });
}

async function paragonToOverlay(planner, title, variant, step) {
  const o = { ...(S.st.settings.overlay || {}), planner, title, variant: variant || 0, step: step || 0, board: 0, enabled: true, paragon: { ...(S.st.settings.overlay?.paragon || {}), on: true } };
  try { S.st.settings = await post('/api/settings', { overlay: o }); toast('Paragon impostato nell\'overlay'); } catch (e) { toast(e.message); }
}

// note personali sulle build salvate
function bindNote(b) {
  const el = $('#bnote'); if (!el) return;
  const sv = savedOf(b.url);
  if (!sv) { el.innerHTML = ''; return; }
  el.innerHTML = `<textarea class="inp bnote" placeholder="Le tue note su questa build (es. “manca il Tuskhelm, uso Writhing al posto”)">${esc(sv.note || '')}</textarea>`;
  const ta = $('textarea', el);
  ta.oninput = () => touchSaved(b.url, { note: ta.value });
}

// ---------------------------------------------------------------- confronto tra due build
const cmpList = () => { try { return JSON.parse(localStorage.getItem('hr.cmp') || '[]'); } catch { return []; } };
const cmpHas = url => cmpList().some(x => x.url === url);
function cmpToggle(b) {
  let l = cmpList();
  if (cmpHas(b.url)) l = l.filter(x => x.url !== b.url);
  else { l.push({ url: b.url, src: b.src, title: b.title, cls: b.cls, type: b.type, profile: b.profile || '', season: b.fromSeason || S.season }); if (l.length > 2) l = l.slice(-2); }
  localStorage.setItem('hr.cmp', JSON.stringify(l));
  cmpBar();
  toast(l.length === 2 ? 'Pronte per il confronto' : (cmpHas(b.url) ? 'Aggiunta al confronto: scegline un\'altra' : 'Tolta dal confronto'));
}
function cmpBar() {
  let el = $('#cmpbar'); const l = cmpList();
  if (!l.length || S.page === 'cmp') { el?.remove(); return; }
  if (!el) { el = document.createElement('div'); el.id = 'cmpbar'; el.className = 'glass'; document.body.appendChild(el); }
  el.innerHTML = `<span>⇄ <b>${l.map(x => esc(x.title)).join(' · ')}</b>${l.length < 2 ? ' <span class="muted">· scegli un\'altra build</span>' : ''}</span>${l.length === 2 ? '<button class="btn pri sm" id="cmpgo">Confronta</button>' : ''}<button class="btn sm ghost" id="cmpx">✕</button>`;
  $('#cmpgo', el)?.addEventListener('click', () => { location.hash = '#/cmp'; });
  $('#cmpx', el).onclick = () => { localStorage.removeItem('hr.cmp'); cmpBar(); };
}
async function loadView(b) { return viewOf(b); }
const pickVariant = (view, b) => { if (!/level/i.test(b.type)) { const i = view.variants.findIndex(v => /endgame/i.test(v.name || '')); if (i >= 0) return view.variants[i]; const j = view.variants.findIndex(v => !/level/i.test(v.name || '')); if (j >= 0) return view.variants[j]; } return view.variants[0]; };
const slotNorm = s => { s = String(s || '').toLowerCase(); if (/^ring/.test(s)) return s.replace(/\s*\d$/, '') + (/(2)$/.test(s) ? ' 2' : ' 1'); if (isWeapon(s)) return 'weapon'; return s; };
PAGES.cmp = async main => {
  const l = cmpList();
  cmpBar();
  main.innerHTML = `<div class="page"><div class="row" style="gap:8px;margin-bottom:12px"><button class="btn" id="cback">← Torna alle build</button><h1 class="pt" style="margin:0 0 0 6px;font-size:24px">Confronto</h1></div><div id="cbody"><div class="skel" style="height:400px"></div></div></div>`;
  $('#cback').onclick = () => history.length > 1 ? history.back() : (location.hash = '#/build');
  if (l.length < 2) { $('#cbody').innerHTML = `<div class="glass empty">${IC.search}<span>Scegli due build con “⇄ Confronta” nella pagina di una build.</span></div>`; return; }
  let A, B;
  try { [A, B] = await Promise.all(l.map(loadView)); } catch (e) { $('#cbody').innerHTML = noData(e.message); return; }
  const va = pickVariant(A, l[0]), vb = pickVariant(B, l[1]);
  const by = v => { const m = {}; let w = 0; (v.gear || []).forEach(g => { if (!g.n || /^empty$/i.test(g.slot)) return; let k = slotNorm(g.slot); if (k === 'weapon') k = 'weapon ' + (++w); if (k === 'ring 1' && m[k]) k = 'ring 2'; if (m[k]) k += '+'; m[k] = g; }); return m; };
  const ga = by(va), gb = by(vb);
  const order = ['helm', 'chest armor', 'gloves', 'pants', 'boots', 'amulet', 'ring 1', 'ring 2', 'weapon 1', 'weapon 2', 'weapon 3', 'weapon 4', 'offhand'];
  const keys = [...new Set([...order.filter(k => ga[k] || gb[k]), ...Object.keys(ga), ...Object.keys(gb)])];
  const cell = (view, g) => g ? `<div class="ccell">${icoImg(g.i, 'ci', g.n, g.fb || slotIcon(g.slot))}<b>${esc(TN(view, g.n, g.it))}</b></div>` : '<span class="muted">—</span>';
  const same = (x, y) => x && y && norm(x.n) === norm(y.n);
  const listCmp = (title, xa, xb) => { const na = new Set(xa.map(x => norm(x.n))), nb = new Set(xb.map(x => norm(x.n))); return `<tr class="sub"><td colspan="3">${title}</td></tr><tr><td></td><td>${xa.map(x => `<span class="badge ${nb.has(norm(x.n)) ? '' : 'diff'}">${esc(TN(A, x.n, x.it))}</span>`).join(' ') || '—'}</td><td>${xb.map(x => `<span class="badge ${na.has(norm(x.n)) ? '' : 'diff'}">${esc(TN(B, x.n, x.it))}</span>`).join(' ') || '—'}</td></tr>`; };
  const shared = keys.filter(k => same(ga[k], gb[k])).length;
  $('#cbody').innerHTML = `<div class="glass card"><div class="muted" style="margin-bottom:10px">${shared} pezzi su ${keys.length} uguali · le differenze sono evidenziate</div><table class="ctable"><thead><tr><th></th><th>${clsIcon(l[0].cls, 16)} ${esc(l[0].title)}<small>${esc(l[0].src)} · ${esc(va.name || '')}</small></th><th>${clsIcon(l[1].cls, 16)} ${esc(l[1].title)}<small>${esc(l[1].src)} · ${esc(vb.name || '')}</small></th></tr></thead><tbody>
    <tr class="sub"><td colspan="3">Equipaggiamento</td></tr>
    ${keys.map(k => `<tr class="${same(ga[k], gb[k]) ? '' : 'diff'}"><td class="sl">${esc(TN(A, (ga[k] || gb[k]).slot, (ga[k] || gb[k]).slotIt))}</td><td>${cell(A, ga[k])}</td><td>${cell(B, gb[k])}</td></tr>`).join('')}
    ${listCmp('Abilità', va.skills || [], vb.skills || [])}
    ${listCmp('Talismani e sigillo', [...(va.seal ? [va.seal] : []), ...(va.charms || [])], [...(vb.seal ? [vb.seal] : []), ...(vb.charms || [])])}
    ${listCmp('Rune', va.runes || [], vb.runes || [])}
  </tbody></table></div>`;
};

// ---------------------------------------------------------------- patch: novità per le tue build
async function patchForYou(el) {
  if (!el) return;
  const saved = savedList();
  const classes = [...new Set([...saved.map(x => x.cls), S.cls])];
  const names = new Set();
  for (const sv of saved.slice(0, 6)) {
    try { const b = { ...sv, profile: sv.profile }; const view = await loadView(b); const v = view.variants[sv.variant] || pickVariant(view, b); (v.skills || []).forEach(x => names.add(x.n)); (v.gear || []).forEach(x => x.n && !/^empty$/i.test(x.slot) && names.add(x.n)); } catch { }
  }
  let p; try { p = await api('/api/patches'); } catch { el.innerHTML = '<div class="muted">Note delle patch non disponibili.</div>'; return; }
  const a = p.articles?.[0], x = a?.patches?.[0];
  if (!x) { el.innerHTML = '<div class="muted">Nessuna patch.</div>'; return; }
  const doc = new DOMParser().parseFromString(x.html || '', 'text/html');
  const clsWords = { barbarian: 'Barbarian', druid: 'Druid', necromancer: 'Necromancer', rogue: 'Rogue', sorcerer: 'Sorcerer', spiritborn: 'Spiritborn', paladin: 'Paladin', warlock: 'Warlock' };
  const hits = [];
  let h3 = '', h4 = '';
  doc.body.querySelectorAll('h2,h3,h4,li').forEach(n => {
    if (n.tagName === 'H2') { h3 = ''; h4 = ''; return; }
    if (n.tagName === 'H3') { h3 = n.textContent.trim(); h4 = ''; return; }
    if (n.tagName === 'H4') { h4 = n.textContent.trim(); return; }
    if (n.querySelector('li')) return;
    const t = n.textContent.trim();
    const byName = [...names].find(nm => nm.length > 3 && t.toLowerCase().includes(nm.toLowerCase()));
    const byCls = classes.find(c => new RegExp('\\b' + clsWords[c] + '\\b', 'i').test(h3));
    if (byName || byCls) hits.push({ t, why: byName ? TN({ tr: {} }, byName, '') : clsName(byCls) + (h4 ? ' · ' + h4 : '') });
  });
  el.innerHTML = hits.length ? `<div class="muted" style="font-size:12px;margin-bottom:6px">${esc(x.version || '')} · ${hits.length} modifiche che toccano le tue classi e build</div><div class="pfy">${hits.slice(0, 12).map(h => `<div class="pfyi"><span class="badge">${esc(h.why)}</span><span>${esc(h.t.length > 220 ? h.t.slice(0, 220) + '…' : h.t)}</span></div>`).join('')}</div>${hits.length > 12 ? `<a class="link" href="#/patch" style="font-size:12.5px">…e altre ${hits.length - 12} nella pagina Patch →</a>` : ''}` : `<div class="muted">L'ultima patch (${esc(x.version || '')}) non tocca le tue classi o le build salvate.</div>`;
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
      <div class="seg" id="dcls"><button data-v="" class="${!ui.cls ? 'on' : ''}">Tutte le classi</button>${CLASSES.map(([c]) => `<button data-v="${c}" class="${ui.cls === c ? 'on' : ''}" title="${esc(clsName(c))}">${clsIcon(c, 16)}</button>`).join('')}</div>
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
    $('#dboss').innerHTML = `<div class="lbl" style="padding:6px 12px 8px">Boss e fonti</div>` + (L.bosses || []).filter(b => (b.items || []).length).map(b => `<div class="bossrow ${ui.boss === b.id ? 'on' : ''}" data-b="${b.id}"><span class="ic" style="width:30px;height:30px;border-radius:9px;display:grid;place-items:center;background:${b.kind === 'boss' ? 'rgba(255,61,31,.16)' : (b.kind === 'mythic' ? 'rgba(217,160,255,.16)' : 'rgba(255,255,255,.07)')};color:${b.kind === 'boss' ? '#ff8a6a' : (b.kind === 'mythic' ? '#d9a0ff' : 'var(--ink2)')}">${(b.kind === 'boss' ? IC.boss : IC.star).replace('<svg', '<svg width="16" height="16"')}</span><div class="t"><b>${esc(b.name)}</b><small>${(b.items || []).length} oggetti${b.location ? ' · ' + esc(b.location) : ''}</small></div></div>`).join('');
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
      <div class="glass card"><h2>Rotazione delle abilità</h2>
        ${tg('rotation.on', 'Mostra la rotazione', o.rotLines?.length ? esc(o.rotTitle || '') + ' · ' + o.rotLines.length + ' passaggi' : 'Scegli una build e premi “Rotazione nell\'overlay”')}
        ${tg('rotation.lock', 'Riquadro rotazione bloccato')}
        ${o.rotLines?.length ? `<ol class="rotlist">${o.rotLines.map(l => `<li>${esc(l)}</li>`).join('')}</ol>` : ''}
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
  if (!o.planner) { box.innerHTML = `<div class="muted" style="margin-top:6px">Nessuna build scelta. Apri una build in <a class="link" href="#/build">Build</a>, scheda Paragon, e premi “Paragon nell'overlay in gioco”.</div>`; return; }
  box.innerHTML = `<div class="skel" style="height:120px"></div>`;
  let p; try { p = await api('/api/ovparagon?id=' + encodeURIComponent(o.planner)); } catch (e) { box.innerHTML = `<div class="muted">Paragon non disponibile: ${esc(e.message)}</div>`; return; }
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
        <div class="row" style="gap:8px"><button class="btn" id="srefresh">${IC.refresh} Aggiorna ora</button>${S.st.collector?.enabled ? `<button class="btn" id="scollect">Raccogli D4Builds e Mobalytics</button>` : ''}</div>
        ${S.st.collector?.enabled ? `<div class="muted" style="font-size:12.5px" id="scol">${collectTxt()}</div>` : ''}
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
  $('#scollect')?.addEventListener('click', async () => { try { await fetch('/api/collect'); toast('Raccolta avviata: va avanti in sottofondo'); } catch { } });
  const colT = setInterval(() => { const el = $('#scol'); if (!el) return clearInterval(colT); el.textContent = collectTxt(); }, 2000);
  $('#srefresh').onclick = async () => { const b = $('#srefresh'); b.disabled = true; try { await api('/api/refresh'); S.builds = {}; S.loot = {}; S.st = await api('/api/state'); renderSeasonPick(); toast('Dati aggiornati'); } catch (e) { toast(e.message); } b.disabled = false; renderDataStatus($('#sstat')); };
};

function collectTxt() {
  const c = S.st.collector || {};
  if (c.running) return `Raccolta in corso dal tuo PC · ${c.step || ''}${c.total ? ` (${c.done}/${c.total})` : ''}. Va avanti anche a finestra chiusa.`;
  return `Questo PC legge D4Builds e Mobalytics (che bloccano il server) e li carica sul server ogni 6 ore${c.at ? ' · ultima raccolta ' + new Date(c.at * 1000).toLocaleString('it-IT', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : ''}.`;
}

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
      (bosses.length ? `<div class="lbl" style="padding:6px 10px">Boss</div>` + bosses.map(b => `<div class="mrow" data-boss="${b.id}"><div class="t"><b>${esc(b.name)}</b><small>${(b.items || []).length} oggetti</small></div></div>`).join('') : '') +
      (builds.length ? `<div class="lbl" style="padding:6px 10px">Build</div>` + builds.map(b => `<div class="mrow" data-burl="${esc(b.url)}" data-bcls="${b.cls}">${tierBadge(b.tier, b.cls)}<div class="t"><b>${esc(b.title)}</b><small>${esc(b.src)} · ${esc(clsName(b.cls))}</small></div></div>`).join('') : '') ||
      `<div class="muted" style="padding:12px">Nessun risultato</div>`;
    box.classList.add('on');
    $$('.mrow', box).forEach(r => r.onclick = () => {
      box.classList.remove('on');
      if (r.dataset.q) { S.ui.drop = { ...(S.ui.drop || {}), q: r.dataset.q, boss: '' }; location.hash = '#/drop'; if (S.page === 'drop') route(); }
      else if (r.dataset.boss) { S.ui.drop = { ...(S.ui.drop || {}), q: '', boss: r.dataset.boss }; location.hash = '#/drop'; if (S.page === 'drop') route(); }
      else { setCls(r.dataset.bcls); const b = (B?.builds || []).find(x => x.url === r.dataset.burl); if (b) { const was = location.hash; openBuildPage(b); if (location.hash === was) route(); } }
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
// barra della raccolta in sottofondo (D4Builds e Mobalytics) con il tempo stimato
function collectBar(c) {
  let el = $('#colbar');
  if (!c?.running) { if (el) { el.classList.add('done'); setTimeout(() => el.remove(), 500); } return; }
  if (!el) { el = document.createElement('div'); el.id = 'colbar'; el.className = 'glass'; el.innerHTML = '<div class="cbt"></div><div class="cbtrack"><i></i></div>'; document.body.appendChild(el); }
  const pct = c.total ? Math.min(100, Math.round(c.done / c.total * 100)) : 3;
  const eta = c.eta > 0 ? (c.eta >= 90 ? `circa ${Math.round(c.eta / 60)} min` : `meno di 2 min`) : 'calcolo il tempo…';
  el.querySelector('.cbt').innerHTML = `<b>Scarico le build</b> · ${esc(c.step || '')}${c.total ? ` · ${c.done}/${c.total}` : ''}<span class="sp"></span><span class="muted">${eta}</span>`;
  el.querySelector('.cbtrack i').style.width = pct + '%';
}
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
