import { h, pressure, num } from './ui.js';
import { api } from './api.js';

// Each tire against the placard pressure, with a warning when one is slowly going down.
export async function tireCard() {
  const d = await api('/api/tires').catch(() => null);
  if (!d || !d.tires.some((t) => t.kpa != null)) return null;
  const target = (i) => (i < 2 ? d.placardFront : d.placardRear);
  const rows = d.tires.map((t, i) => {
    const p = target(i);
    const diff = t.kpa != null && p ? Math.round(((t.kpa - p) / p) * 100) : null;
    return h('div', { class: 'row' }, h('span', { class: 'grow' }, t.name, t.leak ? h('span', { class: 'chip bad', style: 'margin-left:6px' }, 'losing air') : null,
      h('div', { class: 'muted' }, [p ? `target ${pressure(p)}` : null, diff != null ? `${diff > 0 ? '+' : ''}${diff}% from target` : null, t.perDay != null ? `${t.perDay > 0 ? '+' : ''}${num(t.perDay * 0.145038, 2)} psi a day` : null].filter(Boolean).join(' · '))),
      h('b', {}, t.kpa != null ? pressure(t.kpa) : '—'));
  });
  return h('section', { class: 'card' }, h('h2', {}, 'Tire pressure and trend'),
    d.note ? h('p', { class: 'banner' }, d.note) : null,
    ...rows,
    h('p', { class: 'muted' }, d.days >= 5 ? `The daily trend uses ${d.days} days of readings, adjusted for the outside temperature so a cold morning is not mistaken for a leak. Readings can be warmer than the cold placard value.` : 'A slow-leak check needs about five days of readings. The target is the car\'s placard value; readings can be warmer than the cold placard.'));
}
