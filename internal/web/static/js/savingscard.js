import { h, isMetric, num } from './ui.js';
import { api } from './api.js';

const fmt = (v, d) => Math.abs(v).toLocaleString(undefined, { minimumFractionDigits: d, maximumFractionDigits: d });
const money = (cur, v) => `${v < 0 ? '−' : ''}${cur}${fmt(v, 2)}`;
// per distance a few more decimals show the difference between 5.7 and 6.4 cents
export const perDistance = (cur, costPerKm) => `${cur}${fmt(isMetric() ? costPerKm : costPerKm * 1.609344, 3)} per ${isMetric() ? 'km' : 'mi'}`;
const co2 = (kg) => (isMetric() ? `${num(kg, 0)} kg` : `${num(kg * 2.20462, 0)} lb`);

// One period in a sentence or two: what charging cost per distance, and the saving against the gas car when one is set.
export function periodLines(cur, r, gasSet, before) {
  const out = [];
  if (r.costPerKm != null) out.push(h('p', {}, h('b', {}, perDistance(cur, r.costPerKm))));
  else if (r.partial && r.km > 0) out.push(h('p', { class: 'muted' }, 'Some charges have no price yet, so there is no cost per distance. Add your price per kWh in Charging settings.'));
  if (gasSet && r.saved != null) out.push(h('p', {}, r.saved >= 0 ? 'You saved ' : 'It cost ', h('b', {}, money(cur, Math.abs(r.saved))), r.saved >= 0 ? ' against the gas car' : ' more than the gas car', r.gasCost != null ? h('span', { class: 'muted' }, ` (gas would have been ${money(cur, r.gasCost)})`) : null));
  if (gasSet && r.saved != null && r.sources && r.sources.length) out.push(h('p', { class: 'muted' }, `Worked out charge by charge: the energy added, turned into the gas that goes as far, at ${r.sources.join(', ')}.`));
  if (gasSet && r.skipped > 0) out.push(h('p', { class: 'muted' }, `${r.skipped} ${r.skipped === 1 ? 'charge is' : 'charges are'} left out until there are trips to judge the car's efficiency by.`));
  if (gasSet && r.saved != null && before != null && Math.abs(before - r.saved) >= 0.005) out.push(h('p', { class: 'muted' }, `Counting the distance driven instead, as earlier versions did, this was ${money(cur, before)}.`));
  if (gasSet && r.co2AvoidedKg != null && r.co2AvoidedKg > 0) out.push(h('p', { class: 'muted' }, `About ${co2(r.co2AvoidedKg)} of CO2 avoided.`));
  return out;
}

// Gas today: the live price where the car is, how it moved this week, and what a mile costs electric against the gas car.
export function gasTodayLines(g) {
  if (!g || !g.live || !g.ready) return [];
  const cents = g.change == null ? '' : `, ${g.change >= 0 ? 'up' : 'down'} ${Math.round(Math.abs(g.change) * 100)}¢ since last week`;
  const out = [h('p', {}, 'Gas today: ', h('b', {}, `${g.currency}${fmt(g.perGallon, 2)} a gallon`), h('span', { class: 'muted' }, ` (${g.area}, week of ${g.week}${cents})`))];
  if (g.electricPerKm != null && g.gasPerKm != null) out.push(h('p', {}, 'Per distance: electric ', h('b', {}, perDistance(g.currency, g.electricPerKm)), ', gas car ', h('b', {}, perDistance(g.currency, g.gasPerKm))));
  return out;
}

// Costs and savings, for the Charging page: this month and the year so far.
export async function savingsCard(admin) {
  const [v, gn] = await Promise.all([api('/api/savings').catch(() => null), api('/api/gasnow').catch(() => null)]);
  if (!v || !(v.month.km > 0 || v.year.km > 0)) return null;
  const cur = v.currency;
  const month = periodLines(cur, v.month, v.gasSet, v.before && v.before.month), year = periodLines(cur, v.year, v.gasSet, v.before && v.before.year);
  if (!month.length && !year.length && v.gasSet) return null;
  return h('section', { class: 'card' }, h('h2', {}, 'Cost per distance and savings'),
    v.gasSet ? null : h('p', { class: 'muted' }, 'Set the gas price and your gas car\'s economy in Charging settings to see what you save against it and the CO2 you avoid.', admin ? [' ', h('a', { href: '#/charging-settings' }, 'Open Charging settings')] : ''),
    v.gasSet && !v.live && admin ? h('p', { class: 'muted' }, 'Add a free EIA key in Charging settings to use the real average gas price where you charged, week by week.') : null,
    ...gasTodayLines(gn),
    month.length ? h('div', {}, h('h3', {}, 'This month'), ...month) : null,
    year.length ? h('div', {}, h('h3', {}, 'This year so far'), ...year) : null);
}
