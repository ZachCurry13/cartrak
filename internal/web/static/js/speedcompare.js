import { h, isMetric } from './ui.js';
import { api } from './api.js';
import { fitSpeedModel } from './bestspeed.js';
import { bestSpeedCard } from './bestspeedcard.js';
import { getReserve, getTarget } from './charge.js';

const KEY = 'cartrak.cmpdist';
const saved = () => { try { return Number(localStorage.getItem(KEY)) || 0; } catch (e) { return 0; } };

// Two speeds side by side on the efficiency page: energy per mile, and for a trip of the length you choose, the driving time
// saved against the charging time added. The charging speed is measured from your own fast charges between reserve and target.
export async function speedCompare(bands) {
  const model = fitSpeedModel(bands);
  const latest = await api('/api/latest?summary=1').catch(() => null);
  if (!model || !latest || !latest.capacityKwh) return null;
  const low = getReserve(), high = getTarget();
  const win = high - low >= 10 ? await api(`/api/charging/window?from=${low}&to=${high}`).catch(() => null) : null;
  const fast = await api('/api/charging/fast-average').catch(() => null);
  const kw = win && win.minutesPerPct ? (latest.capacityKwh / 100) * 60 / win.minutesPerPct : fast && fast.avgKw ? fast.avgKw : 100;
  const metric = isMetric();
  const dist = h('input', { type: 'number', min: 20, inputmode: 'numeric', value: saved() || (metric ? 960 : 600), 'aria-label': 'Trip length', style: 'max-width:6em' });
  const box = h('div');
  const show = () => {
    const v = Number(dist.value);
    try { localStorage.setItem(KEY, String(v)); } catch (e) { /* storage blocked */ }
    const km = metric ? v : v * 1.609344;
    const card = bestSpeedCard({ model, distanceKm: km, startPct: 90, reservePct: low, targetPct: high, chargeKw: kw, capacityKwh: latest.capacityKwh,
      cChosen: model.c(model.vTypical), compareOnly: true }, show);
    box.replaceChildren(...(card ? [card] : [h('p', { class: 'muted' }, 'Enter a longer trip to compare speeds.')]));
  };
  dist.addEventListener('change', show);
  show();
  return h('div', {}, h('h3', {}, 'Compare two speeds'),
    h('div', { style: 'display:flex;gap:8px;align-items:center;flex-wrap:wrap' }, h('label', { for: 'cd', style: 'margin:0' }, 'Trip length'), (dist.id = 'cd', dist), h('span', { class: 'muted' }, metric ? 'km' : 'mi')),
    box);
}
