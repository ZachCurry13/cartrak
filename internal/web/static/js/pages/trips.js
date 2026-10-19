import { h, emptyState, stat, catChip, chip, dist, when, duration, pct } from '../ui.js';
import { api } from '../api.js';
import { createMap, mapConfig } from '../map.js';
import { effSpan, kwhText } from '../energy.js';
import { monthsCard } from '../rangeband.js';
import { tripSelect } from '../tripselect.js';

const RANGES = { week: 7, month: 30, year: 365, all: 0 };

export default async function trips(root, _a, session) {
  const mc = await mapConfig();
  const admin = session && session.me.role === 'admin';
  let lastTs = [];
  const journeys = h('div');
  const sel = tripSelect(paintList, (id) => { location.hash = `#/journey/${id}`; });
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
      stat('Work trips', s.work.trips), stat('Personal trips', s.personal.trips),
      ...(s.work.kwh + s.personal.kwh > 0 ? [stat('Energy used', kwhText(s.work.kwh + s.personal.kwh)), stat('Efficiency', effSpan(s.work.energyKm + s.personal.energyKm, s.work.kwh + s.personal.kwh))] : [])),
    h('p', {}, h('a', { class: 'btn sec', href: `/api/export/trips.csv?${q()}` }, 'Export CSV')));
    lastTs = ts;
    const unlabelled = ts.filter((x) => !x.open && x.categorySource === 'default').length;
    review.replaceChildren(...(admin && unlabelled ? [h('a', { class: 'banner notice', href: '#/review', style: 'display:block' }, `${unlabelled} trip${unlabelled === 1 ? '' : 's'} to review. Tap to label them.`)] : []));
    paintList();
    list.hidden = view === 'map';
    mapBox.hidden = view !== 'map';
    if (view === 'map') mapBox.replaceChildren(overview(await api(`/api/trips/routes?${q()}`)));
  }
  function paintList() {
    sel.state.shown = lastTs;
    list.replaceChildren(...(lastTs.length ? lastTs.map(row) : [emptyState('No trips in this range', 'Try a wider range. Trips appear after the car drives and Ford reports the trip.')]));
    selectRow.replaceChildren(sel.button(admin && lastTs.length > 1) || '');
    sel.paintBar();
  }
  async function paintJourneys() {
    const js = await api('/api/journeys').catch(() => []);
    journeys.replaceChildren(...(js.length ? [h('h2', {}, 'Journeys'), h('div', { class: 'cols' }, ...js.map((j) => h('a', { class: 'card', href: `#/journey/${j.id}`, style: 'display:block;color:inherit;text-decoration:none' },
      h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, j.name), chip('work', `${j.trips} trips`)),
      h('div', { class: 'muted' }, `${dist(j.distanceKm, 0)} · ${when(j.startTs)}`))))] : []));
  }
  // Every trip in the range on one map: work in amber, personal in blue. Tap a route to open the trip.
  const COLOR = { work: '#b45309', personal: '#1b5fc4' };
  function overview(routes) {
    if (!routes.length) return h('p', { class: 'muted' }, 'No trips with a GPS route in this range.');
    const m = createMap(mc, { height: 420 });
    m.show({ routes: routes.map((r) => ({ points: r.points, color: COLOR[r.category] || COLOR.personal, width: 4, opacity: 0.85, onClick: () => { location.hash = `#/trip/${r.id}`; } })) });
    return h('div', {}, m.el, h('p', { class: 'muted' }, h('span', { class: 'chip work' }, 'work'), ' ', h('span', { class: 'chip personal' }, 'personal'), '  Tap a route to open the trip.'));
  }
  // ' · 2.44 mi/kWh' as a tappable piece, or nothing when the trip has no usable energy figure.
  const effPart = (t) => { const e = effSpan(t.distanceKm, t.energyKwh, t.energyEstimated ? '≈ ' : ''); return e.nodeType === 3 ? [] : [' · ', e]; };
  const row = (t) => h('a', { class: 'card', href: sel.state.on ? '#' : `#/trip/${t.id}`, onclick: sel.state.on ? (e) => { e.preventDefault(); if (sel.eligible(t)) sel.toggle(t); } : null,
    style: `display:block;color:inherit;text-decoration:none${sel.picked.has(t.id) ? ';border-color:var(--accent)' : sel.state.on && !sel.eligible(t) ? ';opacity:.5' : ''}` },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, sel.picked.has(t.id) ? '✓ ' : '', t.open ? 'In progress' : dist(t.distanceKm)), t.journeyId ? chip('work', 'journey') : null, ' ', catChip(t)),
    h('div', { class: 'muted' }, `${when(t.startTs, t.tz)} · ${duration(t.startTs, t.endTs)} · ${pct(t.startSoc)} → ${pct(t.endSoc)}`, ...effPart(t)),
    t.startPlace || t.endPlace ? h('div', { class: 'muted' }, `${t.startPlace || '…'} → ${t.endPlace || '…'}`) : null,
    t.notes ? h('div', { class: 'muted' }, t.notes) : null);
  const selectRow = h('p');
  const review = h('div');
  root.append(h('p', {}, h('a', { class: 'btn sec', href: '#/history' }, 'History feed'), ' ', h('a', { class: 'btn sec', href: '#/stats' }, 'Statistics'), ' ', h('a', { class: 'btn sec', href: '#/plan' }, 'Plan a trip · 70 vs 75')), review, journeys, segCat, segRange, ...(mc.enabled ? [segView] : []), sum, selectRow, sel.bar, list, mapBox, monthsCard());
  paintJourneys();
  await draw();
}
