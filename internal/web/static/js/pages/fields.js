import { h, chip, emptyState } from '../ui.js';
import { api } from '../api.js';

const FILTERS = [['all', 'All'], ['changing', 'Changing'], ['still', 'Never changed'], ['unused', 'Not used']];
const WINDOWS = [[24, '24 h'], [48, '2 days'], [168, '7 days']];

const ago = (ts, now) => {
  if (!ts) return 'never';
  const m = Math.max(0, Math.round((now - ts) / 60));
  return m < 90 ? `${m} min ago` : m < 2880 ? `${Math.round(m / 60)} h ago` : `${Math.round(m / 1440)} d ago`;
};
const pretty = (s) => s.replace(/([a-z0-9])([A-Z])/g, '$1 $2').replace(/^./, (c) => c.toUpperCase());

// Everything the car reports and what each thing has been doing: so nothing goes unnoticed.
export default async function fields(root) {
  let hours = 48, filter = 'all', q = '';
  const box = h('div'), segF = h('div', { class: 'seg' }), segW = h('div', { class: 'seg' });
  const search = h('input', { type: 'search', placeholder: 'Find a field', 'aria-label': 'Find a field' });
  search.addEventListener('input', () => { q = search.value.trim().toLowerCase(); draw(); });
  root.append(h('p', {}, h('a', { href: '#/data' }, '← Car data')), h('div', { class: 'page-title' }, h('h2', {}, 'Field watcher')),
    h('p', { class: 'muted' }, 'Every kind of reading the car sends, how often it changes, and what CarTrak does with it. A field marked "Not used" is data nobody is looking at yet. Tap one for its range.'), segW, segF, search, box);
  let data = null;

  async function load() { data = await api(`/api/fields?hours=${hours}`); draw(); }

  function draw() {
    segW.replaceChildren(...WINDOWS.map(([v, t]) => h('button', { class: hours === v ? 'on' : '', onclick: () => { hours = v; load(); } }, t)));
    segF.replaceChildren(...FILTERS.map(([k, t]) => h('button', { class: filter === k ? 'on' : '', onclick: () => { filter = k; draw(); } }, t)));
    if (!data || !data.fields.length) return box.replaceChildren(emptyState('No readings yet', 'Fields appear once CarTrak has stored a reading from the car.'));
    const all = data.fields;
    const used = all.filter((f) => f.usedFor).length, onlyShown = all.filter((f) => !f.usedFor && f.shown).length, none = all.filter((f) => !f.usedFor && !f.shown).length;
    const list = all.filter((f) => (filter === 'all' || (filter === 'changing' && f.changes > 0) || (filter === 'still' && f.changes === 0) || (filter === 'unused' && !f.usedFor))
      && (!q || f.metric.toLowerCase().includes(q) || pretty(f.metric).toLowerCase().includes(q)));
    box.replaceChildren(h('p', { class: 'muted' }, `${all.length} kinds of reading in ${data.readings} stored readings. ${used} used for a feature, ${onlyShown} only shown on Car data, ${none} not shown or used. ${all.filter((f) => f.changes > 0).length} changed.`),
      ...(list.length ? list.map((f) => row(f, data.now)) : [h('p', { class: 'muted' }, 'Nothing matches.')]));
  }

  function row(f, now) {
    const more = h('div', { class: 'muted', hidden: true, style: 'margin-top:6px;font-size:.85rem' },
      [f.min != null ? `Range ${f.min} to ${f.max}.` : null, f.parts > 1 ? `Made of ${f.parts} values.` : null, f.reportedAt ? `The car's own time for it: ${f.reportedAt}.` : null, `Seen in ${f.samples} readings.`].filter(Boolean).join(' '));
    const tag = f.usedFor ? chip('ok', f.usedFor) : f.shown ? chip('work', 'Car data only') : chip('bad', 'Not used');
    const el = h('button', { class: 'row', style: 'width:100%;text-align:left;background:none;border:0;border-top:1px solid var(--line);color:inherit;display:block;padding:10px 0', 'aria-expanded': 'false' },
      h('div', { style: 'display:flex;justify-content:space-between;gap:10px;flex-wrap:wrap' }, h('b', {}, pretty(f.metric)), h('span', { class: 'muted' }, f.last || '—')),
      h('div', { style: 'display:flex;justify-content:space-between;gap:10px;flex-wrap:wrap;margin-top:2px' }, h('span', { class: 'muted' }, f.changes ? `changed ${f.changes} time${f.changes === 1 ? '' : 's'} · last ${ago(f.lastChangeTs, now)}` : 'did not change'), tag), more);
    el.addEventListener('click', () => { more.hidden = !more.hidden; el.setAttribute('aria-expanded', String(!more.hidden)); });
    return el;
  }
  await load();
}
