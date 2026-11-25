(async function () {
  // Estrattore D4Builds: gira dentro la pagina della build (lettore nascosto) e mette il risultato in window.__hrOut
  if (window.__hrRun) return;
  window.__hrRun = 1;
  const sleep = ms => new Promise(r => setTimeout(r, ms));
  const T = el => (el ? el.textContent.replace(/\s+/g, ' ').trim() : '');
  const A = (el, s) => [...(el || document).querySelectorAll(s)];
  const img = el => { if (el && el.tagName !== 'IMG') el = el.querySelector('img'); return el ? { n: el.getAttribute('alt') || '', i: el.getAttribute('src') || '' } : null; };
  const num = v => Math.round(parseFloat(v) || 0);
  const content = () => document.querySelector('.builder__content');
  const tabs = () => A(document, '.builder__navigation__link');
  async function waitFor(fn, ms) { const end = Date.now() + ms; while (Date.now() < end) { try { const v = fn(); if (v) return v; } catch (e) { } await sleep(150); } return null; }
  async function openTab(i) {
    const t = tabs()[i];
    if (!t) return false;
    if (!t.classList.contains('active')) { t.click(); await sleep(400); }
    await waitFor(() => tabs()[i] && tabs()[i].classList.contains('active'), 4000);
    await sleep(900);
    return true;
  }
  function notes(root) {
    const out = [];
    let line = [];
    const flush = () => { while (line.length && typeof line[line.length - 1] === 'string' && !line[line.length - 1].trim()) line.pop(); if (line.length) out.push(line); line = []; };
    const walk = n => {
      if (n.nodeType === 3) { const s = n.textContent.replace(/[ \t]+/g, ' '); const parts = s.split(/\n/); parts.forEach((p, i) => { if (i) flush(); if (p) line.push(p); }); return; }
      if (n.nodeType !== 1) return;
      const tag = n.tagName, cl = (n.className && n.className.toString()) || '';
      if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'svg' || tag === 'SVG') return;
      if (/(^|\s)(unique|aspect|build__skill|paragon__tile|rune|gem|item)__wrapper/.test(cl) || /__wrapper--formatted-notes-inline/.test(cl)) {
        const im = n.querySelector('img');
        const nmEl = n.querySelector('[class*="__name"]');
        const nm = T(nmEl) || (im ? im.getAttribute('alt') || '' : '') || T(n);
        line.push({ n: nm, i: im ? im.getAttribute('src') || '' : '', t: /build__skill/.test(cl) ? 's' : (/aspect/.test(cl) ? 'a' : (/paragon/.test(cl) ? 'p' : 'u')) });
        return;
      }
      if (tag === 'IMG') { const s = n.getAttribute('src') || ''; if (/sunderarmor/.test(s) && !/ga\.png|tempering|transfigured/.test(s)) line.push({ n: n.getAttribute('alt') || '', i: s, t: 'u' }); return; }
      if (tag === 'BR') { flush(); return; }
      if (tag === 'DIV' && !n.childNodes.length) { flush(); return; }
      if (tag === 'STRONG' || tag === 'B') { const t = T(n); if (t) line.push({ b: t }); return; }
      if (/^H[1-6]$/.test(tag)) { flush(); const t = T(n); if (t) out.push([{ b: t }]); return; }
      const block = /^(P|LI|UL|OL|TR)$/.test(tag);
      if (block) flush();
      if (tag === 'LI') line.push('• ');
      [...n.childNodes].forEach(walk);
      if (block) flush();
    };
    walk(root);
    flush();
    return out;
  }
  function gearTab() {
    const c = content();
    const v = { gear: [], charms: [], runes: [], prime: [], skills: [], shards: [], stats: [] };
    A(c, '.builder__gear__item').forEach(it => {
      const nameEl = it.querySelector('.builder__gear__name');
      if (!nameEl) return;
      const kind = (nameEl.className.match(/builder__gear__name--(\w+)/) || [])[1] || '';
      v.gear.push({ slot: T(it.querySelector('.builder__gear__slot')), n: T(nameEl), k: kind, i: (img(it.querySelector('.builder__gear__icon')) || {}).i || '', gems: A(it, '.builder__gem__icon').map(img).filter(x => x && x.n) });
    });
    const seal = c.querySelector('.builder__seal__icon');
    if (seal) v.seal = img(seal);
    v.charms = A(c, '.builder__charm__icon').map(img).filter(Boolean);
    const rg = A(c, '.builder__gems').find(x => /rune/i.test(T(x.querySelector('.builder__gems__title'))));
    if (rg) v.runes = A(rg, '.builder__gem__icon').map(img).filter(x => x && x.n);
    v.prime = A(c, '.builder__prime__power img').map(img).filter(Boolean);
    A(c, '.builder__skills__section').forEach(sec => {
      const title = T(sec.querySelector('.builder__skills__title'));
      const boons = A(sec, '.builder__skills__boons');
      if (boons.length) {
        v.shardsTitle = title;
        boons.forEach(b => {
          const items = A(b, '.builder__skills__boon');
          const cat = items[0] && img(items[0].querySelector('.builder__skills__boon__icon'));
          const sel = x => { const s = x.querySelector('.selected'); return s && parseFloat(getComputedStyle(s).opacity) > 0.5 ? 1 : 0; };
          const nodes = items.slice(1).map(x => { const ims = A(x, 'img'); const last = ims[ims.length - 1]; const on = sel(x); return { i: last ? last.getAttribute('src') : '', n: (last && (last.getAttribute('src') || '').split('/').pop().replace(/\.png$/, '').replace(/_/g, ' ')) || '', a: on ? 1 : 0 }; });
          v.shards.push({ cat, nodes, a: items[0] ? sel(items[0]) : 0 });
        });
      } else {
        const list = A(sec, '.build__skill__wrapper img, .builder__skill img').map(img).filter(x => x && x.n && !/empty/i.test(x.n) && !/\/UI\//.test(x.i));
        const seen = new Set();
        const uniq = list.filter(x => !seen.has(x.n) && seen.add(x.n));
        if (/skill/i.test(title) || !v.skills.length) { v.skills = uniq; v.skillsTitle = title; }
        else v.extra = (v.extra || []).concat([{ title, items: uniq }]);
      }
    });
    A(c, '.builder__stats__group').forEach(g => {
      const slot = T(g.querySelector('.builder__stats__slot'));
      const list = A(g, '.builder__stat').map(s => ({ t: T(s), ga: s.querySelector('img[src*="ga.png"]') ? 1 : 0, tp: s.querySelector('img[src*="tempering"]') ? 1 : 0, tf: s.querySelector('img[src*="transfigured"]') ? 1 : 0 })).filter(s => s.t && !/^(Stat \d|Tempering Stat \d)$/.test(s.t));
      if (list.length) v.stats.push({ slot, i: (img(g.querySelector('.builder__stats__slot__icon')) || {}).i || '', list });
    });
    const n = c.querySelector('.builder__notes__display');
    return { v, notes: n ? notes(n) : null, notesTitle: T(c.querySelector('.builder__notes__title')) };
  }
  function viewer() {
    const c = content();
    const cv = c && c.querySelector('.viewer-canvas');
    if (!cv) return null;
    const nodes = A(cv, '.viewer-node').map(n => {
      const st = n.getAttribute('style') || '';
      const x = num((st.match(/left:\s*([-\d.]+)px/) || [])[1]), y = num((st.match(/top:\s*([-\d.]+)px/) || [])[1]);
      const ic = n.querySelector('.node-icon') || n.querySelector('.category-icon');
      const bg = n.querySelector('.node-bg') || n.querySelector('.category-icon-bg');
      const rk = T(n.querySelector('.node-rank-badge'));
      const cl = n.className;
      return [x, y, cl.includes('category') ? 'c' : (cl.includes('large') ? 'l' : (cl.includes('diamond') ? 'd' : (cl.includes('connector') ? 'k' : 's'))), ic ? ic.getAttribute('alt') || '' : '', ic ? ic.getAttribute('src') || '' : '', bg ? bg.getAttribute('src') || '' : '', rk, /allocated|unlocked|active/.test(cl) ? 1 : 0];
    });
    const lines = A(cv, 'svg.viewer-lines g').map(g => { const p = g.querySelector('path'); const d = p ? p.getAttribute('d') || '' : ''; const m = d.match(/M\s*([-\d.]+)\s+([-\d.]+)\s*L\s*([-\d.]+)\s+([-\d.]+)/); return m ? [num(m[1]), num(m[2]), num(m[3]), num(m[4]), g.querySelectorAll('path').length > 2 ? 1 : 0] : null; }).filter(Boolean);
    const uniq = {}; lines.forEach(l => { const k = l.slice(0, 4).join(','); if (!uniq[k] || l[4]) uniq[k] = l; });
    return { nodes, lines: Object.values(uniq), remaining: T(c.querySelector('.skill__tree__remaining__total')) };
  }
  function paragon() {
    const c = content();
    if (!c || !c.querySelector('.paragon__board')) return null;
    const boards = A(c, '.paragon__board').map(b => {
      const st = b.getAttribute('style') || '';
      const nameEl = b.querySelector('.paragon__board__name');
      let name = '';
      if (nameEl) [...nameEl.childNodes].forEach(n => { if (n.nodeType === 3) name += n.textContent; });
      const tiles = A(b, '.paragon__board__tile').map(t => {
        const r = +((t.className.match(/\br(\d+)/) || [])[1] || 0), cc = +((t.className.match(/\bc(\d+)/) || [])[1] || 0);
        const ic = t.querySelector('.paragon__board__tile__icon');
        const bg = t.querySelector('.paragon__board__tile__bg');
        return [r, cc, /\bactive\b/.test(t.className) ? 1 : 0, ic ? ic.getAttribute('alt') || '' : '', ic ? (ic.getAttribute('src') || '').replace(/^https:\/\/sunderarmor\.com\/DIABLO4\/Paragon\//, '') : '', bg ? (bg.getAttribute('alt') || '') : ''];
      });
      const gates = A(b, '.paragon__board__gate').map(g => { const s = g.getAttribute('style') || ''; return [num((s.match(/left:\s*([-\d.]+)px/) || [])[1]), num((s.match(/top:\s*([-\d.]+)px/) || [])[1]), /active/.test(g.className) ? 1 : 0]; });
      return { n: name.replace(/\s+/g, ' ').trim(), g: T(b.querySelector('.paragon__board__name__glyph')).replace(/[()]/g, ''), s: A(b, '.paragon__board__stat').map(T), x: num((st.match(/left:\s*([-\d.]+)px/) || [])[1]), y: num((st.match(/top:\s*([-\d.]+)px/) || [])[1]), r: num((st.match(/rotate\(([-\d.]+)deg/) || [])[1]), tiles, gates };
    });
    const glyphs = A(c, '.paragon__selected__glyphs .aspect__name.glyph').map(T);
    const glyphIcons = A(c, '.paragon__selected__glyphs img.glyph').map(x => x.getAttribute('src'));
    return { boards, glyphs: glyphs.map((g, i) => ({ n: g, i: glyphIcons[i] || '' })), points: T(c.querySelector('.skill__tree__remaining__level')), remaining: T(c.querySelector('.skill__tree__remaining__total')) };
  }
  function mercs() {
    const c = content();
    if (!c || !c.querySelector('.skill__tree__merc')) return null;
    const slots = A(c, '.skill__tree__merc__button__wrapper').map(w => { const im = w.querySelector('img'); let n = im ? im.getAttribute('alt') || '' : ''; const src = im ? im.getAttribute('src') || '' : ''; if (!n || /empty/i.test(n)) n = decodeURIComponent(src.split('/').pop().replace(/\.png$/, '')).replace(/_/g, ' ').replace(/(^|\s)\w/g, m => m.toUpperCase()); return { l: T(w.querySelector('.skill__tree__merc__label')), n, i: src }; });
    const items = A(c, '.skill__tree__item').map(it => { const st = it.getAttribute('style') || ''; const im = it.querySelector('img'); return [num((st.match(/left:\s*([-\d.]+)px/) || [])[1]), num((st.match(/top:\s*([-\d.]+)px/) || [])[1]), im ? im.getAttribute('alt') || '' : '', im ? im.getAttribute('src') || '' : '', /--active/.test(it.className) ? 1 : 0, /\blarge\b/.test(it.className) ? 1 : 0]; });
    const seen = new Set();
    return { slots, items: items.filter(x => x[4] && x[2] && !seen.has(x[2]) && seen.add(x[2])).map(x => ({ n: x[2].replace(/-v\d+$/i, ''), i: x[3], l: x[5] })) };
  }
  async function warplans() {
    const c = content();
    const wt = A(c, '.warplan-tab');
    if (!wt.length) return null;
    const out = [];
    for (let i = 0; i < wt.length; i++) {
      const t = A(content(), '.warplan-tab')[i];
      if (!t) break;
      t.click();
      await sleep(700);
      const v = viewer();
      const im = t.querySelector('img');
      const alloc = v ? v.nodes.filter(n => n[2] !== 'c' && n[6] && !/^0\//.test(n[6])) : [];
      out.push({ n: im ? im.getAttribute('alt') : '', i: im ? im.getAttribute('src') : '', tree: alloc.length ? v : null });
    }
    return out;
  }
  try {
    await waitFor(() => document.querySelector('.builder__gear__name'), 20000);
    await sleep(800);
    const head = {
      title: T(document.querySelector('.builder__header__name')), desc: T(document.querySelector('.builder__header__description')),
      updated: T(document.querySelector('.builder__last__updated')), cls: ((img(document.querySelector('.builder__header__icon')) || {}).n || '').replace(/^Diablo 4 /, ''),
      season: T(document.querySelector('.builder__dropdown__wrapper'))
    };
    const vbtn = document.querySelector('.item__arrow__icon--variant');
    let vnames = [];
    if (vbtn) { vbtn.click(); await sleep(500); vnames = A(document, '.build-variant-dropdown .dropdown__option').map(T); vbtn.click(); await sleep(200); }
    if (!vnames.length) vnames = [(document.querySelector('.builder__variant__input') || {}).value || 'Build'];
    const variants = [];
    let notesOut = null, notesTitle = '';
    for (let vi = 0; vi < vnames.length && vi < 8; vi++) {
      if (vi > 0) {
        await openTab(0);
        const b = document.querySelector('.item__arrow__icon--variant');
        if (!b) break;
        b.click(); await sleep(400);
        const opt = A(document, '.build-variant-dropdown .dropdown__option')[vi];
        if (!opt) break;
        opt.click();
        await waitFor(() => ((document.querySelector('.builder__variant__input') || {}).value || '') === vnames[vi], 5000);
        await sleep(1000);
      }
      await openTab(0);
      const g = gearTab();
      if (!notesOut && g.notes) { notesOut = g.notes; notesTitle = g.notesTitle; }
      const v = g.v;
      v.name = vnames[vi];
      if (await openTab(1)) v.tree = viewer();
      if (await openTab(2)) v.paragon = paragon();
      if (await openTab(3)) v.merc = mercs();
      if (await openTab(4)) v.war = await warplans();
      variants.push(v);
    }
    await openTab(0);
    window.__hrOut = JSON.stringify({ ok: 1, head, variants, notes: notesOut, notesTitle });
  } catch (e) {
    window.__hrOut = JSON.stringify({ ok: 0, err: String(e && e.message || e) });
  }
})();
