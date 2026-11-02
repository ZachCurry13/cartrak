import { h } from './ui.js';

const KEY = 'cartrak.hideupd';
const hidden = () => { try { return new Set(JSON.parse(localStorage.getItem(KEY)) || []); } catch (e) { return new Set(); } };
const save = (set) => { try { localStorage.setItem(KEY, JSON.stringify([...set])); } catch (e) { /* ignore */ } };

// What needs updating: CarTrak itself and any part that has a newer version. Each has an id that includes the new version, so a
// notice that was hidden comes back when something newer than the hidden version is out.
export function pendingUpdates(about) {
  if (!about) return [];
  const out = [];
  const u = about.update;
  if (u && u.available) out.push({ id: `cartrak@${u.latest}`, text: `CarTrak v${u.latest} is available (this is v${about.version}). It updates through your TrueNAS app: Stop, then Start.` });
  for (const c of about.components || []) {
    if (c.update) out.push({ id: `${c.name}@${c.latest}`, text: `${c.name} ${c.latest} is available${c.version ? ` (this is ${c.version})` : ''}.${c.note ? ` ${c.note}` : ''}` });
  }
  return out;
}

// A notice at the top of every page, above the page itself, for as long as something needs updating, until it is hidden (on this
// device). Admins only: nobody else can do anything about it.
export function showUpdateNotes(session, view) {
  let box = document.getElementById('updnotes');
  if (!box) { box = h('div', { id: 'updnotes' }); view.before(box); }
  const admin = session.me && session.me.role === 'admin';
  const all = admin ? pendingUpdates(session.about) : [];
  const gone = hidden();
  const shown = all.filter((x) => !gone.has(x.id));
  box.replaceChildren(...shown.map((x) => h('div', { class: 'banner notice', role: 'status', style: 'display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin:8px 0' },
    h('span', { style: 'flex:1 1 14em;min-width:0' }, x.text),
    h('button', { class: 'sec', 'aria-label': `Hide: ${x.text}`, onclick: () => { gone.add(x.id); save(gone); showUpdateNotes(session, view); } }, 'Hide'))),
  all.length > shown.length ? h('p', { class: 'muted', style: 'margin:4px 0' }, `${all.length - shown.length} update notice${all.length - shown.length === 1 ? ' is' : 's are'} hidden. `,
    h('a', { href: '#/about', onclick: () => { save(new Set()); } }, 'Show them again')) : null);
}
