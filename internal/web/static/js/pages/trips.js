import { h, stat, catChip, dist, when, duration, pct } from '../ui.js';
import { api } from '../api.js';
import { createMap, mapConfig } from '../map.js';

const RANGES = { week: 7, month: 30, year: 365, all: 0 };

export default async function trips(root) {
  const mc = await mapConfig();
  let cat = '', range = 'month', view = 'list';
  const list = h('div', { class: 'cols' }), sum = h('div', { class: 'card' });
  const segCat = h('div', { class: 'seg' }), segRange = h('div', { class: 'seg' }), segView = h('div', { class: 'seg' }), mapBox = h('div');
  const q = () => {
    const p = new URLSearchParams();
    if (cat) p.set('category', cat);
    if (RANGES[range]) p.set('from', Math.floor(Date.now() / 1000) - RANGES[range] * 86400);
    return p.toString();
  };
  function seg(el, items, get, set) {
    el.replaceChildren(...items.map(([k, label]) => h('button', { class: get() === k ? 'on' : '', onclick: () => { set(k); draw(); } }, label)));
  }
  async function draw() {
    seg(segCat, [['', 'All'], ['work', 'Work'], ['personal', 'Personal']], () => cat, (v) => { cat = v; });
    seg(segRange, [['week', '7 days'], ['month', '30 days'], ['year', 'Year'], ['all', 'All']], () => range, (v) => { range = v; });
    if (mc.enabled) seg(segView, [['list', 'List'], ['map', 'Map']], () => view, (v) => { view = v; });
    const [ts, s] = await Promise.all([api(`/api/trips?${q()}`), api(`/api/summary?${q().replace(/category=\w+&?/, '')}`)]);
    sum.replaceChildren(h('div', { class: 'grid' }, stat('Work', dist(s.work.distanceKm, 0)), stat('Personal', dist(s.personal.distanceKm, 0)),
      stat('Work trips', s.work.trips), stat('Personal trips', s.personal.trips)),
    h('p', {}, h('a', { class: 'btn sec', href: `/api/export/trips.csv?${q()}` }, 'Export CSV')));
    list.replaceChildren(...(ts.length ? ts.map(row) : [h('p', { class: 'muted' }, 'No trips in this range yet.')]));
    list.hidden = view === 'map';
    mapBox.hidden = view !== 'map';
    if (view === 'map') mapBox.replaceChildren(overview(await api(`/api/trips/routes?${q()}`)));
  }
  // Every trip in the range on one map: work in amber, personal in blue. Tap a route to open the trip.
  const COLOR = { work: '#b45309', personal: '#1b5fc4' };
  function overview(routes) {
    if (!routes.length) return h('p', { class: 'muted' }, 'No trips with a GPS route in this range.');
    const m = createMap(mc, { height: 420 });
    m.show({ routes: routes.map((r) => ({ points: r.points, color: COLOR[r.category] || COLOR.personal, width: 4, opacity: 0.85, onClick: () => { location.hash = `#/trip/${r.id}`; } })) });
    return h('div', {}, m.el, h('p', { class: 'muted' }, h('span', { class: 'chip work' }, 'work'), ' ', h('span', { class: 'chip personal' }, 'personal'), '  Tap a route to open the trip.'));
  }
  const row = (t) => h('a', { class: 'card', href: `#/trip/${t.id}`, style: 'display:block;color:inherit;text-decoration:none' },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, t.open ? 'In progress' : dist(t.distanceKm)), catChip(t)),
    h('div', { class: 'muted' }, `${when(t.startTs, t.tz)} · ${duration(t.startTs, t.endTs)} · ${pct(t.startSoc)} → ${pct(t.endSoc)}`),
    t.notes ? h('div', { class: 'muted' }, t.notes) : null);
  root.append(segCat, segRange, ...(mc.enabled ? [segView] : []), sum, list, mapBox);
  await draw();
}
