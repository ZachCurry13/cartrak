import { h, chip, dist, isMetric } from '../ui.js';
import { api } from '../api.js';
import { planTrip } from '../tripplan.js';
import { volText } from '../fuelunits.js';

// The road-trip planner for a gas or hybrid car: how many fuel stops a distance needs and about where, from how far a percent of the
// tank goes in your own driving. Nothing about charging or a battery.

const RESERVE = 15; // the same level the low fuel alert uses
const toKm = (v) => (isMetric() ? v : v * 1.609344);
const median = (v) => { const s = [...v].sort((a, b) => a - b); return s.length ? s[Math.floor(s.length / 2)] : null; };

export default async function planFuel(root) {
  const [latest, trips, fuel] = await Promise.all([api('/api/latest?summary=1').catch(() => null), api('/api/trips?limit=60').catch(() => null), api('/api/fuel').catch(() => null)]);
  const list = Array.isArray(trips) ? trips : (trips && trips.trips) || [];
  const perTank = median(list.filter((t) => t.kmPerTank > 20 && t.distanceKm >= 16 && !t.distanceSrc).map((t) => t.kmPerTank));
  const x = (latest && latest.extra) || {};
  const carRange = Object.entries(x).find(([k]) => /fuelRange$/i.test(k));
  const fromCar = carRange && latest && latest.fuelPct > 10 ? Number(carRange[1]) / latest.fuelPct : null; // the car's own range figure for one percent
  const kmPerPct = perTank ? perTank / 100 : fromCar;
  const unit = isMetric() ? 'km' : 'mi';
  const distance = h('input', { id: 'pd', type: 'number', min: 1, inputmode: 'decimal', placeholder: unit === 'mi' ? '420' : '680' });
  const start = h('input', { id: 'ps', type: 'number', min: 1, max: 100, inputmode: 'numeric', value: latest && latest.fuelPct != null ? Math.round(latest.fuelPct) : 100 });
  const mins = h('input', { id: 'pm', type: 'number', min: 1, max: 60, inputmode: 'numeric', value: 10 });
  const out = h('div');
  const field = (id, label, input, help) => h('div', {}, h('label', { for: id }, label), input, help ? h('p', { class: 'muted' }, help) : null);

  function show() {
    const r = planTrip({ distanceKm: toKm(Number(distance.value)), startPct: Number(start.value), reservePct: RESERVE, targetPct: 100, kmPerPct });
    if (!r.ok) return out.replaceChildren(h('p', { class: 'muted' }, r.reason.replace('battery', 'tank')));
    const each = Number(mins.value) || 10;
    out.replaceChildren(h('section', { class: 'card' }, h('h2', {}, 'The plan'),
      h('p', {}, r.stops.length ? chip('work', `${r.stops.length} fuel stop${r.stops.length === 1 ? '' : 's'}`) : chip('ok', 'No fuel stop needed'), ' ',
        h('span', { class: 'muted' }, r.stops.length ? `About ${each * r.stops.length} min at the pump in all.` : `You arrive with about ${Math.round(r.arrivePct)}% in the tank.`)),
      ...r.stops.map((s, i) => h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, `Stop ${i + 1}, about ${dist(s.atKm, 0)} in`),
        h('div', { class: 'muted' }, `Arrive with ${Math.round(s.arrivePct)}%, fill to ${Math.round(s.chargeToPct)}%`)), h('b', {}, `${each} min`))),
      r.stops.length ? h('p', { class: 'muted' }, `You arrive with about ${Math.round(r.arrivePct)}%.`) : null,
      h('p', { class: 'muted' }, `A guide, not a route. Hills, highway speed, wind and cold use more fuel. Stops keep ${RESERVE}% in the tank, the level of the low fuel alert, and pick real stations near each mark.`)));
  }
  for (const i of [distance, start, mins]) i.addEventListener('input', show);

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Road-trip planner')),
    h('section', { class: 'card' }, h('h2', {}, 'Your trip'),
      kmPerPct ? h('p', { class: 'muted' }, `Your tank goes about ${dist(kmPerPct * 100, 0)}${perTank ? ', from your recent driving' : ", from the car's own range figure"}${fuel && fuel.tankGal ? ` (a ${volText(fuel.tankGal)} tank)` : ''}.`)
        : h('p', { class: 'banner' }, 'CarTrak does not know how far your tank goes yet. It needs a reading with the range, or a few trips.'),
      field('pd', `Trip distance (${unit})`, distance), field('ps', 'Fuel in the tank now (%)', start), field('pm', 'Minutes for each fuel stop', mins)),
    out);
  show();
}
