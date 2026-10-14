import { h, num } from './ui.js';
import { api } from './api.js';

const money = (cur, v) => `${cur}${v.toFixed(2)}`;

// "What would a charge cost?": pick a start and an end battery level and see it at your home and public prices.
export function forecastCard() {
  const from = h('input', { type: 'number', min: 0, max: 99, step: 1, inputmode: 'numeric', 'aria-label': 'Start battery percent', style: 'max-width:6em' });
  const to = h('input', { type: 'number', min: 1, max: 100, step: 1, inputmode: 'numeric', 'aria-label': 'End battery percent', value: 80, style: 'max-width:6em' });
  const out = h('div', { role: 'status' });
  let timer = 0;
  async function run() {
    const q = new URLSearchParams();
    if (from.value !== '') q.set('from', from.value);
    if (to.value !== '') q.set('to', to.value);
    let d;
    try { d = await api(`/api/charge-forecast?${q}`); } catch (e) { out.replaceChildren(h('p', { class: 'muted' }, e.message)); return; }
    if (from.value === '' && d.socNow != null) {
      from.value = Math.round(d.socNow);
      if (d.socNow >= +to.value) { to.value = 100; return run(); } // already past 80: show a full charge
    }
    if (d.capacityKwh == null) return out.replaceChildren(h('p', { class: 'muted' }, 'CarTrak does not know the battery size yet. It learns it from the car.'));
    if (!d.ok) return out.replaceChildren(h('p', { class: 'muted' }, 'Choose an end level higher than the start.'));
    const f = d.forecast, cur = d.currency;
    out.replaceChildren(...[
      h('p', {}, h('b', {}, num(f.batteryKwh, 1, ' kWh')), ' into the battery'),
      h('div', { class: 'row' }, h('span', { class: 'grow' }, 'At home', h('div', { class: 'muted' }, `${num(f.homeKwh, 1, ' kWh')} from the wall, with ${Math.round((1 - d.efficiency) * 100)}% lost as heat`)),
        h('b', {}, f.homeCost != null ? money(cur, f.homeCost) : '—')),
      f.offPeakCost != null ? h('div', { class: 'row' }, h('span', { class: 'grow' }, 'At home in the cheaper hours', f.homeCost != null && f.offPeakCost < f.homeCost ? h('div', { class: 'muted' }, `Saves ${money(cur, f.homeCost - f.offPeakCost)}`) : null),
        h('b', {}, money(cur, f.offPeakCost))) : null,
      h('div', { class: 'row' }, h('span', { class: 'grow' }, 'Fast charger', h('div', { class: 'muted' }, 'No heat loss counted')),
        h('b', {}, f.publicCost != null ? money(cur, f.publicCost) : '—')),
      f.homeCost == null && f.publicCost == null ? h('p', { class: 'muted' }, 'Add your price per kWh in More, Settings to see the cost.') : null].filter(Boolean));
  }
  const later = () => { clearTimeout(timer); timer = setTimeout(run, 250); };
  from.addEventListener('input', later); to.addEventListener('input', later);
  const quick = (v) => h('button', { class: 'sec', onclick: () => { to.value = v; run(); } }, `${v}%`);
  run();
  return h('section', { class: 'card' }, h('h2', {}, 'What would a charge cost?'),
    h('div', { style: 'display:flex;gap:8px;align-items:center;flex-wrap:wrap' },
      h('label', { style: 'margin:0' }, 'From'), from, h('label', { style: 'margin:0' }, 'to'), to, quick(80), quick(100)), out);
}
