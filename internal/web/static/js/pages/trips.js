import { h, emptyState, stat, catChip, chip, dist, when, duration, pct } from '../ui.js';
import { api } from '../api.js';
import { categoryList, catLabel } from '../categories.js';
import { tripCard, colorKey } from '../tripcard.js';
import { isSaver } from '../mode.js';
import { createMap, mapConfig } from '../map.js';
import { effSpan, kwhText } from '../energy.js';
import { monthsCard } from '../rangeband.js';
import { tripSelect } from '../tripselect.js';
import { routeEstimateBar } from '../routeestimatebar.js';

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
    seg(segCat, [['', 'All'], ...categoryList().map((c) => [c.key, c.label])], () => cat, (v) => { cat = v; });
    seg(segRange, [['week', '7 days'], ['month', '30 days'], ['year', 'Year'], ['all', 'All']], () => range, (v) => { range = v; });
    if (mc.enabled) seg(segView, [['list', 'List'], ['map', 'Map']], () => view, (v) => { view = v; });
    const [ts, s] = await Promise.all([api(`/api/trips?${q()}`), api(`/api/summary?${q().replace(/category=[\w-]+&?/, '')}`)]);
    const keys = categoryList().map((c) => c.key).filter((k) => k === 'work' || k === 'personal' || (s[k] && s[k].trips > 0));
    const all = Object.values(s);
    const kwh = all.reduce((n, x) => n + x.kwh, 0), energyKm = all.reduce((n, x) => n + x.energyKm, 0);
    sum.replaceChildren(h('div', { class: 'grid' }, ...keys.map((k) => stat(catLabel(k), dist(s[k] ? s[k].distanceKm : 0, 0))), ...keys.map((k) => stat(`${catLabel(k)} trips`, s[k] ? s[k].trips : 0)),
      ...(kwh > 0 ? [stat('Energy used', kwhText(kwh)), stat('Efficiency', effSpan(energyKm, kwh))] : [])),
    h('p', {}, h('a', { class: 'btn sec', href: `/api/export/trips.csv?${q()}` }, 'Export CSV')));
    lastTs = ts;
    const unlabelled = ts.filter((x) => !x.open && x.categorySource === 'default').length;
    review.replaceChildren(...(admin && unlabelled ? [h('a', { class: 'banner notice', href: '#/review', style: 'display:block' }, `${unlabelled} trip${unlabelled === 1 ? '' : 's'} to review. Tap to label them.`)] : []));
    paintList();
    if (!isSaver()) { // the little route pictures arrive a moment later, so the list is there at once
      api(`/api/trips/routes?${q()}&limit=100`).then((rs) => { thumbs = new Map(rs.map((r) => [r.id, r])); if (lastTs === ts) paintList(); }).catch(() => {});
    }
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
  let thumbs = null;
  const avgEfficiency = () => { let km = 0, kwh = 0; for (const x of lastTs) if (x.energyKwh > 0 && x.distanceKm > 1) { km += x.distanceKm; kwh += x.energyKwh; } return kwh > 0 ? km / kwh : null; };
  const row = (x) => tripCard(x, { sel, admin, thumbs, avgKmPerKwh: avgEfficiency() });
  const selectRow = h('p');
  const review = h('div');
  const estBar = h('div');
  if (admin) routeEstimateBar().then((b) => { if (b) estBar.append(b); }).catch(() => {});
  root.append(estBar, h('nav', { class: 'linkrow', 'aria-label': 'More about trips' }, h('a', { href: '#/day' }, 'Day view'), h('a', { href: '#/history' }, 'History'), h('a', { href: '#/stats' }, 'Statistics'), h('a', { href: '#/reports' }, 'Reports'), h('a', { href: '#/plan' }, 'Plan a trip'), admin ? h('a', { href: '#/rules' }, 'Trip rules') : null), review, journeys, segCat, segRange, ...(mc.enabled ? [segView] : []), sum, colorKey(), selectRow, sel.bar, list, mapBox, monthsCard());
  paintJourneys();
  await draw();
}
