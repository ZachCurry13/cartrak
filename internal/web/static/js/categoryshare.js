import { h, dist } from './ui.js';
import { catColor, catLabel } from './categories.js';
import { hm } from './format.js';

const share = (part, total) => (total > 0 ? (part / total) * 100 : 0);

// How the driving was split between work, personal and the other categories: one bar and a row for each, by distance, with the
// share of trips and of time next to it. `s` is the summary the Trips page already has (a category to its trips, km and seconds).
export function categoryShare(s) {
  const rows = Object.entries(s).filter(([, v]) => v.trips > 0).map(([k, v]) => ({ key: k, ...v })).sort((a, b) => b.distanceKm - a.distanceKm);
  const km = rows.reduce((n, r) => n + r.distanceKm, 0), trips = rows.reduce((n, r) => n + r.trips, 0), sec = rows.reduce((n, r) => n + (r.seconds || 0), 0);
  if (rows.length < 1 || km <= 0) return null;
  const bar = h('div', { role: 'img', 'aria-label': rows.map((r) => `${catLabel(r.key)} ${Math.round(share(r.distanceKm, km))} percent`).join(', '),
    style: 'display:flex;height:14px;border-radius:7px;overflow:hidden;margin:8px 0;background:var(--card2)' },
    ...rows.map((r) => h('div', { title: catLabel(r.key), style: `flex:${r.distanceKm} 0 0;background:${catColor(r.key)}` })));
  return h('section', { class: 'card' }, h('h2', {}, 'Where your driving goes'), bar,
    ...rows.map((r) => h('div', { class: 'row', style: 'align-items:baseline;gap:8px' },
      h('span', { style: `flex:0 0 10px;height:10px;border-radius:50%;background:${catColor(r.key)}` }),
      h('div', { class: 'grow', style: 'min-width:0' }, h('b', {}, `${catLabel(r.key)} ${Math.round(share(r.distanceKm, km))}%`),
        h('div', { class: 'muted' }, `${dist(r.distanceKm, 0)} · ${r.trips} trip${r.trips === 1 ? '' : 's'} (${Math.round(share(r.trips, trips))}%)${sec > 0 && r.seconds ? ` · ${hm(r.seconds)} driving (${Math.round(share(r.seconds, sec))}%)` : ''}`)))),
    h('p', { class: 'muted' }, 'The percentage is the share of the distance in the range you chose above. The brackets are the share of trips and of the time spent driving.'));
}
