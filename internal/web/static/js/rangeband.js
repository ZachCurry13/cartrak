import { h, dist } from './ui.js';
import { api } from './api.js';
import { getReserve, getTarget } from './charge.js';

// How far the car goes between two battery levels (your reserve and your charging target), worked out
// from how it has really been driven. It follows the season, so it differs from the car's own guess.
const cache = { at: 0, key: '', data: null };

async function band(days) {
  const key = `${getReserve()}-${getTarget()}-${days}`;
  if (cache.key !== key || Date.now() - cache.at > 300000) {
    cache.data = await api(`/api/range-band?low=${getReserve()}&high=${getTarget()}&days=${days}`);
    Object.assign(cache, { at: Date.now(), key });
  }
  return cache.data;
}

// A line under the battery on Home, for example "20 to 80%: 140 mi".
export function bandRow() {
  const row = h('p', { style: 'margin:10px 0 0' });
  row.hidden = true;
  band(30).then((b) => {
    if (b.bandKm == null) return;
    row.replaceChildren(h('span', { class: 'muted' }, `${b.low} to ${b.high}%: `), h('b', {}, dist(b.bandKm, 0)),
      h('span', { class: 'muted' }, b.source === 'trips' ? ` from your last ${b.days} days of driving` : " from the car's own range figure"));
    row.hidden = false;
  }).catch(() => { /* offline or no data: the line stays hidden */ });
  return row;
}

const monthName = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short', year: 'numeric' }); };

// A table on the Trips page: the same stretch of battery, month by month.
export function monthsCard() {
  const box = h('section', { class: 'card' });
  box.hidden = true;
  band(30).then((b) => {
    if (!b.months || !b.months.length) return;
    box.replaceChildren(h('h2', {}, 'Range by month'),
      h('p', { class: 'muted' }, `How far ${b.low} to ${b.high}% of battery went each month, from your driving. Cold weather and highway speeds shorten it.`),
      ...b.months.map((m) => h('div', { class: 'row' }, h('span', { class: 'grow' }, monthName(m.month), ' ', h('span', { class: 'muted' }, `${m.trips} trips`)), h('b', {}, dist(m.bandKm, 0)))));
    box.hidden = false;
  }).catch(() => { /* leave it hidden */ });
  return box;
}
