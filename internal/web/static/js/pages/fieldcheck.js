import { h } from '../ui.js';
import { CHECKS, AUTO } from '../fieldchecks.js';
import { api } from '../api.js';

const KEY = 'cartrak.fieldcheck';
const load = () => { try { return JSON.parse(localStorage.getItem(KEY)) || {}; } catch (e) { return {}; } };
const save = (v) => { try { localStorage.setItem(KEY, JSON.stringify(v)); } catch (e) { /* the page works without it */ } };
const LABEL = { ok: 'Works', bad: 'Problem', skip: 'Skipped' };

// Field check: the things only a person with the car and a phone can try, one card each. Tap the result, add a note, then copy the
// whole list to send. It is kept on this device only until cleared.
export default async function fieldcheck(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const state = load();
  const found = [];
  let timer = null;
  // The answers are also kept on the server, so the Debug report and a tracking export carry them.
  const sync = () => {
    clearTimeout(timer);
    timer = setTimeout(() => api('/api/fieldcheck', { method: 'PUT', body: { state: { at: Math.floor(Date.now() / 1000), checks: state, auto: found.map(([n, r]) => [n, r.text]), titles: Object.fromEntries(CHECKS.map((c) => [c.id, c.title])) } } }).catch(() => {}), 700);
  };
  const persist = () => { save(state); sync(); };
  const autoBox = h('section', { class: 'card' }, h('h2', {}, 'Checked for you'), h('p', { class: 'muted' }, 'Looking…'));
  const list = h('div');
  const msg = h('p', { class: 'muted', role: 'status' });
  const box = h('textarea', { readonly: true, rows: 12, hidden: true, 'aria-label': 'Results', style: 'width:100%;box-sizing:border-box;font-family:ui-monospace,Menlo,Consolas,monospace;font-size:.75rem' });

  function report() {
    const lines = [`CarTrak field check, ${new Date().toLocaleString()}`, '', 'Checked for you:', ...found.map(([n, r]) => `- ${n}: ${r.text}${r.ok ? '' : '  <-- look at this'}`), '', 'My checks:'];
    for (const c of CHECKS) {
      const s = state[c.id];
      lines.push(`- ${c.title}: ${s && s.s ? LABEL[s.s] : 'not done'}${s && s.note ? ` — ${s.note}` : ''}`);
    }
    return lines.join('\n');
  }
  function draw() {
    const done = CHECKS.filter((c) => state[c.id] && state[c.id].s).length;
    list.replaceChildren(h('p', { class: 'muted' }, `${done} of ${CHECKS.length} done`), ...CHECKS.map((c) => {
      const s = state[c.id] || {};
      const set = (v) => { state[c.id] = { ...s, s: s.s === v ? '' : v }; persist(); draw(); };
      const note = h('input', { type: 'text', placeholder: s.s === 'bad' ? 'What happened?' : 'Note (optional)', value: s.note || '', 'aria-label': `Note for ${c.title}`,
        onchange: (e) => { state[c.id] = { ...(state[c.id] || {}), note: e.target.value.trim() }; persist(); } });
      return h('section', { class: 'card' }, h('h2', {}, c.title), h('p', {}, h('b', {}, 'Try: '), c.steps), h('p', { class: 'muted' }, h('b', {}, 'Should: '), c.expect),
        h('div', { class: 'seg' }, ...['ok', 'bad', 'skip'].map((v) => h('button', { class: s.s === v ? 'on' : '', onclick: () => set(v) }, LABEL[v]))), note);
    }));
  }
  const copy = h('button', { onclick: async () => {
    const text = report();
    try { await navigator.clipboard.writeText(text); msg.textContent = 'Copied. Paste it into the chat.'; } catch (e) { box.value = text; box.hidden = false; box.select(); msg.textContent = 'Select all and copy (the browser blocked the button).'; }
  } }, 'Copy results');
  const clear = h('button', { class: 'sec', onclick: () => { for (const k of Object.keys(state)) delete state[k]; persist(); draw(); msg.textContent = 'Cleared.'; } }, 'Start over');

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Field check')),
    h('p', { class: 'muted' }, 'Work down the list when you can. Tap Works, Problem or Skipped on each, add a note, then copy the results to send. The Debug report page has the rest.'),
    autoBox, list, h('div', { class: 'actions' }, copy, clear), msg, box);
  api('/api/fieldcheck').then((d) => { // answers saved from another device or an earlier visit
    const saved = (d.state && d.state.checks) || {};
    let changed = false;
    for (const [id, v] of Object.entries(saved)) if (!state[id] && v && v.s) { state[id] = v; changed = true; }
    if (changed) { save(state); draw(); }
  }).catch(() => {});
  draw();
  for (const [name, fn] of AUTO) found.push([name, await fn(session)]);
  sync();
  autoBox.replaceChildren(h('h2', {}, 'Checked for you'), ...found.map(([n, r]) => h('div', { class: 'row' }, h('span', { class: 'grow' }, n), h('span', { class: r.ok ? '' : 'warn' }, `${r.ok ? '✓ ' : '! '}${r.text}`))));
}
