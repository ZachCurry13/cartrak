import { h, num, dist } from './ui.js';
import { api } from './api.js';
import { tapBars } from './chart.js';

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
  const bars = tapBars(d.months.slice(-12).map((m) => ({ label: monthName(m.month), value: m.kwh - low * 0.9, text: `${num(m.kwh, 1, ' kWh')} (average that month)` })), { summary: 'Estimated battery size each month' });
  return h('section', { class: 'card' }, h('h2', {}, 'Battery health (estimate)'),
    d.ready ? h('p', {}, h('b', {}, `${num(d.startKwh, 1)} → ${num(d.nowKwh, 1, ' kWh')}`), ' ', h('span', { class: 'muted' }, `${d.changePct > 0 ? '+' : ''}${d.changePct.toFixed(1)}% since the start`)) : null,
    d.ready ? h('p', {}, VERDICT[d.verdict]) : h('p', { class: 'muted' }, 'CarTrak needs a couple of months of readings before it can say anything about the trend.'),
    bars,
    h('p', { class: 'muted' }, 'This is the size of the battery as the car itself estimates it. The estimate moves with temperature and how full the battery is, so a small dip in winter is normal. It is not a lab test of the battery.'));
}

// The battery's own figures next to what the car shows, and the usable size worked out from them over time.
export async function truthCard() {
  const d = await api('/api/battery-truth').catch(() => null);
  if (!d || !d.now || d.now.actualSoc == null) return null;
  const n = d.now, u = d.usable;
  const row = (l, v, note) => h('div', { class: 'row' }, h('span', { class: 'grow' }, l, note ? h('div', { class: 'muted' }, note) : null), h('b', {}, v));
  return h('section', { class: 'card' }, h('h2', {}, 'True battery level'),
    h('p', { class: 'muted' }, 'The percentage on the car\'s display is not the battery\'s own level. The car reports both.'),
    row('The car shows', n.displayedSoc != null ? `${num(n.displayedSoc, 1)}%` : '—'),
    row('The battery\'s own level', `${num(n.actualSoc, 1)}%`),
    n.usableKwh != null ? row('Usable size now', num(n.usableKwh, 1, ' kWh'), `Energy left ${n.energyKwh != null ? num(n.energyKwh, 1, ' kWh') : '—'} divided by the true level${n.fordKwh != null ? `. Ford's own estimate is ${num(n.fordKwh, 1, ' kWh')}` : ''}.`) : null,
    n.maxRangeKm != null ? row('Range on a full charge', dist(n.maxRangeKm, 0), `The car's own estimate for 100%${d.fullRange && d.fullRange.ready ? `. ${dist(d.fullRange.startKwh, 0)} → ${dist(d.fullRange.nowKwh, 0)} over the history CarTrak has (${d.fullRange.changePct > 0 ? '+' : ''}${d.fullRange.changePct.toFixed(1)}%). It moves with the weather and how you have been driving, so read it as a slow trend.` : '. A trend appears after a couple of months.'}`) : null,
    u && u.ready ? h('p', {}, `${num(u.startKwh, 1)} → ${num(u.nowKwh, 1, ' kWh')} over the history CarTrak has (${u.changePct > 0 ? '+' : ''}${u.changePct.toFixed(1)}%). ${({ steady: 'Steady.', gentle: 'A gentle decline, which is normal.', watch: 'A bigger drop than usual; worth watching.' })[u.verdict] || ''}`) : h('p', { class: 'muted' }, 'A trend appears after a couple of months of readings between 20 and 95 percent.'));
}
