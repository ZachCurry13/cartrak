import { h, chip, when, duration, pct, num } from '../ui.js';
import { api } from '../api.js';

export default async function charging(root) {
  const cs = await api('/api/charges');
  root.append(h('h2', {}, 'Charging sessions'));
  if (!cs.length) return root.append(h('p', { class: 'muted' }, 'No charging sessions recorded yet.'));
  root.append(...cs.map((c) => h('section', { class: 'card' },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, when(c.startTs)), c.open ? chip('ok', 'charging') : null),
    h('div', { class: 'muted' }, `${pct(c.startSoc)} → ${pct(c.endSoc)} · ${duration(c.startTs, c.endTs)} · peak ${num(c.peakKw, 1, ' kW')}`))));
}
