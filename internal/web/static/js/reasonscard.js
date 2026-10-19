import { h, num } from './ui.js';
import { api } from './api.js';

const NAMES = { speed: 'Speed', temperature: 'Weather', other: 'Everything else (wind, hills, climate control, braking)' };

// Across your recent trips, how much energy each reason cost or saved compared with your usual driving.
export async function reasonsCard() {
  const d = await api('/api/efficiency-reasons').catch(() => null);
  if (!d || d.trips < 3 || !d.usualKwh) return null;
  const rows = d.items.map((i) => {
    const bad = i.kwh > 0.05, good = i.kwh < -0.05;
    return h('div', { class: 'row' }, h('span', { class: 'grow' }, NAMES[i.key] || i.key,
      h('div', { class: 'muted' }, bad ? 'cost you extra energy' : good ? 'saved you energy' : 'about neutral')),
      h('b', { class: bad ? 'bad' : '' }, `${i.kwh > 0 ? '+' : i.kwh < 0 ? '−' : ''}${num(Math.abs(i.kwh), 1, ' kWh')}`));
  });
  const diff = d.usedKwh - d.usualKwh;
  return h('section', { class: 'card' }, h('h2', {}, 'What costs you the most'),
    h('p', { class: 'muted' }, `Your last ${d.trips} trips in ${d.days} days used ${num(d.usedKwh, 0, ' kWh')}, which is ${num(Math.abs(diff), 1, ' kWh')} ${diff > 0 ? 'more' : 'less'} than your usual driving would have for the same distance. Here is where the difference came from.`),
    ...rows,
    h('p', { class: 'muted' }, 'Speed and weather are measured against your own history. Everything else is what is left over, so it includes wind, hills, heating and cooling the cabin, and how hard you brake and accelerate.'));
}
