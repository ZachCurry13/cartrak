import { h, num } from './ui.js';
import { api } from './api.js';

const monthName = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short' }); };
const VERDICT = {
  steady: 'Steady. The car\'s estimate of its battery size has not dropped in a way that matters.',
  gentle: 'A gentle decline, which is normal as a battery ages.',
  watch: 'A bigger drop than usual. It is worth keeping an eye on, and mentioning at your next service.',
};

// Battery health from the car's own battery-size estimate over time, or null when there is no estimate.
export async function batteryCard() {
  const d = await api('/api/battery-health').catch(() => null);
  if (!d || !d.months.length) return null;
  const top = Math.max(...d.months.map((m) => m.kwh));
  const low = Math.min(...d.months.map((m) => m.kwh));
  const bars = h('div', { class: 'ybars', role: 'img', 'aria-label': 'Estimated battery size each month' },
    ...d.months.slice(-12).map((m) => h('div', { class: 'ybar' },
      h('div', { class: 'yfill', style: `height:${Math.max(8, Math.round(((m.kwh - low * 0.9) / (top - low * 0.9 || 1)) * 100))}%`, title: `${monthName(m.month)}: ${num(m.kwh, 1, ' kWh')}` }),
      h('span', { class: 'muted' }, monthName(m.month)))));
  return h('section', { class: 'card' }, h('h2', {}, 'Battery health (estimate)'),
    d.ready ? h('p', {}, h('b', {}, `${num(d.startKwh, 1)} → ${num(d.nowKwh, 1, ' kWh')}`), ' ', h('span', { class: 'muted' }, `${d.changePct > 0 ? '+' : ''}${d.changePct.toFixed(1)}% since the start`)) : null,
    d.ready ? h('p', {}, VERDICT[d.verdict]) : h('p', { class: 'muted' }, 'CarTrak needs a couple of months of readings before it can say anything about the trend.'),
    bars,
    h('p', { class: 'muted' }, 'This is the size of the battery as the car itself estimates it. The estimate moves with temperature and how full the battery is, so a small dip in winter is normal. It is not a lab test of the battery.'));
}
