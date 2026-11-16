import { h, dist, isMetric } from './ui.js';
import { api } from './api.js';
import { toDisplayVol, volUnit } from './fuelunits.js';

// "Since the car was new": the energy, cost and gas saved over the odometer's whole life, estimated from the driving CarTrak has seen.
// Every figure is an estimate and says so. Nothing at all until there is enough driving to go on.

const round = (v, to) => Math.round(v / to) * to;
const kwh = (v) => `${round(v, v >= 10000 ? 100 : 10).toLocaleString()} kWh`;
const money = (cur, v) => `${cur}${round(v, v >= 1000 ? 10 : 1).toLocaleString()}`;
const eff = (kmPerKwh) => (isMetric() ? `${kmPerKwh.toFixed(1)} km/kWh` : `${(kmPerKwh * 0.621371).toFixed(1)} mi/kWh`);
const day = (ts) => new Date(ts * 1000).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });

export function lifetimeCard() {
  const box = h('div');
  api('/api/lifetime').then((r) => {
    if (!r || !r.ok) return;
    const l = r.lifetime, cur = r.currency || '$';
    const line = (label, value, note) => h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap;align-items:baseline;padding:4px 0' },
      h('span', { style: 'flex:1 1 9em;min-width:0' }, label), h('b', {}, value), note ? h('div', { class: 'muted', style: 'flex-basis:100%' }, note) : null);
    box.replaceChildren(h('section', { class: 'card' }, h('h2', {}, 'Since the car was new (estimated)'),
      h('p', {}, h('b', {}, dist(l.odoKm, 0)), ' on the clock. CarTrak has recorded ', dist(l.trackedKm, 0), ` of them since ${day(l.since)}.`),
      line('Electricity into the battery', `about ${kwh(l.batteryKwh)}`, `between ${kwh(l.batteryLow)} and ${kwh(l.batteryHigh)}`),
      line('From chargers, with charging losses', `about ${kwh(l.wallKwh)}`),
      l.cost != null ? line('What that cost', `about ${money(cur, l.cost)}`, 'at what a kWh has cost you since CarTrak began') : null,
      l.gasCost != null ? line('The same miles in your gas car', `about ${money(cur, l.gasCost)}`, l.gasLiters != null ? `${Math.round(toDisplayVol(l.gasLiters / 3.785411784)).toLocaleString()} ${volUnit()} of gas` : null) : null,
      l.cost != null && l.gasCost != null && l.gasCost > l.cost ? line('Saved against gas', h('span', { class: 'chip ok' }, `about ${money(cur, l.gasCost - l.cost)}`)) : null,
      h('p', { class: 'muted', style: 'margin-top:8px' }, `Worked out from the odometer and your own driving: ${eff(l.trackedEff)} on the trips CarTrak recorded, `,
        l.mildMeasured ? `${eff(l.mildEff)} on its mild-weather trips. ` : l.mildEff > l.trackedEff ? `and about ${eff(l.mildEff)} assumed for mild weather, since so far it has seen mostly cold days. ` : '',
        'The range runs from one to the other; the more seasons CarTrak sees, the closer it gets. Prices are today\'s, not what electricity and gas cost in past years.')));
  }).catch(() => {});
  return box;
}
