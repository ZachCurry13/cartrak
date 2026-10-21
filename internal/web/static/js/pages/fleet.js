import { h, dist, pct, chip, ago, when } from '../ui.js';
import { api } from '../api.js';
import { setCar } from '../car.js';
import { isCharging } from '../chargestate.js';

// All cars (#/fleet, chosen in the car picker): every car side by side and the household's driving together. Each car's own
// pages stay as they were; tap a car to open it.

const COLOURS = ['#3b82f6', '#f59e0b', '#22c55e', '#a855f7', '#ef4444', '#14b8a6'];
const TABS = [['', 'Overview'], ['year', 'This year'], ['trips', 'Trips']];

// One car's numbers, asked of that car.
async function carData(c, from) {
  const ask = (path) => api(path, { car: c.id }).catch(() => null);
  const [vehicle, latest, trips, recap] = await Promise.all([ask('/api/vehicle'), ask('/api/latest?summary=1'), ask('/api/trips?limit=1'), ask(`/api/analytics/recap?month=${from}`)]);
  return { car: c, vehicle, latest, last: trips && trips[0], recap };
}

const open = (id, hash = '#/') => { setCar(String(id)); location.hash = hash; location.reload(); };

function carCard(x, colour) {
  const v = x.vehicle || {}, l = x.latest || {};
  const level = v.battery && l.socPct != null ? pct(l.socPct) : l.fuelPct != null ? `${pct(l.fuelPct)} fuel` : '—';
  const state = l.speedKph > 1 ? ['personal', 'Driving'] : isCharging(l) ? ['ok', 'Charging'] : ['work', 'Parked'];
  const kind = v.label || (v.battery ? 'Electric' : v.engine ? 'Gas' : '');
  return h('a', { class: 'card fleetcar', href: '#', style: `--cc:${colour}`, onclick: (e) => { e.preventDefault(); open(x.car.id); } },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow', style: 'font-size:1.15rem' }, x.car.name), chip(state[0], state[1])),
    h('div', { class: 'bigfig', style: 'margin-top:8px' }, h('div', {}, h('b', {}, level), h('span', {}, kind)), h('div', {}, h('b', {}, l.rangeKm != null ? dist(l.rangeKm, 0) : '—'), h('span', {}, 'range'))),
    x.last ? h('p', { class: 'muted', style: 'margin:8px 0 0' }, `Last trip ${when(x.last.startTs, x.last.tz)}: ${dist(x.last.distanceKm)}${x.last.endPlace ? ` to ${x.last.endPlace}` : ''}`) : null,
    l.ts ? h('p', { class: 'muted', style: 'margin:2px 0 0' }, `reported ${ago(l.ts)}`) : null);
}

// A month or a year together: distance and trips, how much of it was electric, and each car's share.
function togetherCard(xs, period) {
  const km = xs.reduce((a, x) => a + ((x.recap && x.recap.km) || 0), 0);
  if (km <= 0) return null;
  const trips = xs.reduce((a, x) => a + ((x.recap && x.recap.trips) || 0), 0);
  const evKm = xs.reduce((a, x) => a + (x.vehicle && x.vehicle.battery && !x.vehicle.engine && x.recap ? x.recap.km : 0), 0);
  const cost = xs.reduce((a, x) => a + ((x.recap && x.recap.cost) || 0), 0);
  const cur = (xs.find((x) => x.recap && x.recap.currency) || {}).recap?.currency || '$';
  const title = period.includes('-') ? new Date(`${period}-15`).toLocaleDateString([], { month: 'long' }) : period;
  return h('section', { class: 'card' }, h('h2', {}, `${title} together`),
    h('div', { class: 'bigfig' }, h('div', {}, h('b', {}, dist(km, 0)), h('span', {}, `${trips} trips`)), h('div', {}, h('b', {}, `${Math.round((evKm / km) * 100)}%`), h('span', {}, 'of the miles electric')),
      h('div', {}, h('b', {}, `${cur}${cost.toFixed(0)}`), h('span', {}, 'charging'))),
    h('div', { class: 'socbar', style: 'margin-top:12px' }, ...xs.map((x, i) => h('i', { style: `width:${(((x.recap && x.recap.km) || 0) / km) * 100}%;background:${COLOURS[i % COLOURS.length]}`, title: x.car.name }))),
    ...xs.map((x, i) => h('div', { class: 'row', style: 'flex-wrap:nowrap' }, h('span', { class: 'grow', style: 'min-width:0' }, h('i', { class: 'sw', style: `background:${COLOURS[i % COLOURS.length]}` }), x.car.name),
      h('span', { style: 'white-space:nowrap' }, h('b', {}, dist((x.recap && x.recap.km) || 0, 0)), h('span', { class: 'muted' }, ` · ${Math.round((((x.recap && x.recap.km) || 0) / km) * 100)}%`)))));
}

async function overview(body, cars, year) {
  const now = new Date(), period = year ? String(now.getFullYear()) : `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  const xs = await Promise.all(cars.map((c) => carData(c, period)));
  const together = togetherCard(xs, period) || h('p', { class: 'muted' }, 'No driving recorded in this stretch yet.');
  body.replaceChildren(together, h('div', { class: 'cols' }, ...xs.map((x, i) => carCard(x, COLOURS[i % COLOURS.length]))));
}

// Every car's trips in one list, newest first, each marked with its car.
async function trips(body, cars) {
  const lists = await Promise.all(cars.map((c) => api('/api/trips?limit=60', { car: c.id }).then((ts) => ts.map((t) => ({ ...t, car: c }))).catch(() => [])));
  const all = lists.flat().sort((a, b) => b.startTs - a.startTs).slice(0, 120);
  const colour = new Map(cars.map((c, i) => [c.id, COLOURS[i % COLOURS.length]]));
  body.replaceChildren(h('section', { class: 'card' }, ...all.map((t) => h('a', { class: 'row', href: '#', style: 'color:inherit;text-decoration:none;align-items:flex-start', onclick: (e) => { e.preventDefault(); open(t.car.id, `#/trip/${t.id}`); } },
    h('i', { class: 'sw', style: `background:${colour.get(t.car.id)};margin-top:6px` }),
    h('span', { class: 'grow', style: 'flex-basis:12em' }, h('b', {}, dist(t.distanceKm)), ' ', h('span', { class: 'muted' }, t.car.name),
      h('div', { class: 'muted' }, `${when(t.startTs, t.tz)}${t.startPlace || t.endPlace ? ` · ${t.startPlace || '…'} → ${t.endPlace || '…'}` : ''}`))))));
}

export default async function fleet(root, args, session) {
  const cars = session.cars || [];
  const tab = args[0] || '';
  const body = h('div', {}, h('p', { class: 'muted' }, 'Asking each car…'));
  root.append(h('div', { class: 'page-title' }, h('h2', {}, 'All cars')),
    h('nav', { class: 'atabs', 'aria-label': 'All cars' }, ...TABS.map(([k, t]) => h('a', { href: `#/fleet${k ? `/${k}` : ''}`, class: tab === k ? 'on' : '' }, t))), body);
  if (cars.length < 2) { body.replaceChildren(h('p', { class: 'muted' }, 'Add a second car under More, Cars to see them together.')); return; }
  await (tab === 'trips' ? trips(body, cars) : overview(body, cars, tab === 'year'));
}
