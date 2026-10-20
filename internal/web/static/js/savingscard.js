import { h, isMetric, num } from './ui.js';
import { api } from './api.js';

const fmt = (v, d) => Math.abs(v).toLocaleString(undefined, { minimumFractionDigits: d, maximumFractionDigits: d });
const money = (cur, v) => `${v < 0 ? '−' : ''}${cur}${fmt(v, 2)}`;
// per distance a few more decimals show the difference between 5.7 and 6.4 cents
export const perDistance = (cur, costPerKm) => `${cur}${fmt(isMetric() ? costPerKm : costPerKm * 1.609344, 3)} per ${isMetric() ? 'km' : 'mi'}`;
const co2 = (kg) => (isMetric() ? `${num(kg, 0)} kg` : `${num(kg * 2.20462, 0)} lb`);

// One period in a sentence or two: what charging cost per distance, and the saving against the petrol car when one is set.
export function periodLines(cur, r, petrolSet) {
  const out = [];
  if (r.costPerKm != null) out.push(h('p', {}, h('b', {}, perDistance(cur, r.costPerKm))));
  else if (r.partial && r.km > 0) out.push(h('p', { class: 'muted' }, 'Some charges have no price yet, so there is no cost per distance. Add your price per kWh in Charging settings.'));
  if (petrolSet && r.saved != null) out.push(h('p', {}, r.saved >= 0 ? 'You saved ' : 'It cost ', h('b', {}, money(cur, Math.abs(r.saved))), r.saved >= 0 ? ' against the petrol car' : ' more than the petrol car', r.petrolCost != null ? h('span', { class: 'muted' }, ` (petrol would have been ${money(cur, r.petrolCost)})`) : null));
  if (petrolSet && r.co2AvoidedKg != null && r.co2AvoidedKg > 0) out.push(h('p', { class: 'muted' }, `About ${co2(r.co2AvoidedKg)} of CO2 avoided.`));
  return out;
}

// Costs and savings, for the Charging page: this month and the year so far.
export async function savingsCard(admin) {
  const v = await api('/api/savings').catch(() => null);
  if (!v || !(v.month.km > 0 || v.year.km > 0)) return null;
  const cur = v.currency;
  const month = periodLines(cur, v.month, v.petrolSet), year = periodLines(cur, v.year, v.petrolSet);
  if (!month.length && !year.length && v.petrolSet) return null;
  return h('section', { class: 'card' }, h('h2', {}, 'Cost per distance and savings'),
    v.petrolSet ? null : h('p', { class: 'muted' }, 'Set the petrol price and your petrol car\'s economy in Charging settings to see what you save against it and the CO2 you avoid.', admin ? [' ', h('a', { href: '#/charging-settings' }, 'Open Charging settings')] : ''),
    month.length ? h('div', {}, h('h3', {}, 'This month'), ...month) : null,
    year.length ? h('div', {}, h('h3', {}, 'This year so far'), ...year) : null);
}
