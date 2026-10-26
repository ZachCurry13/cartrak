import { carWords } from '../carwords.js';
import { h } from '../ui.js';
import { api } from '../api.js';

const RANGES = [[7, '7 days'], [30, '30 days'], [365, 'Year']];
const COLORS = { driving: 'var(--personal)', charging: 'var(--ok)', parked: 'var(--card2)' };
const pct = (part, whole) => (whole > 0 ? `${Math.round((part / whole) * 100)}%` : '0%');
const hours = (sec) => (sec >= 86400 ? `${(sec / 86400).toFixed(1)} days` : sec >= 3600 ? `${(sec / 3600).toFixed(1)} h` : `${Math.round(sec / 60)} min`);

// How the car's time is spent: parked, driving or charging.
export default async function timeuse(root, _a, session) {
  const fuelCar = carWords(session).fuel;
  let days = 30;
  const seg = h('div', { class: 'seg' });
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Time')),
    h('p', { class: 'muted' }, fuelCar ? 'How the car\'s time is spent: parked or driving, from the trips CarTrak recorded.' : 'How the car\'s time is spent: parked, driving or charging, from the trips and charging sessions CarTrak recorded.'), seg, box);

  const bar = (d, h2 = 14) => {
    const total = d.driving + d.charging + d.parked;
    const part = (k) => h('i', { style: `display:block;height:100%;width:${total ? (d[k] / total) * 100 : 0}%;background:${COLORS[k]}` });
    return h('div', { style: `display:flex;height:${h2}px;border-radius:7px;overflow:hidden;background:var(--card2)` }, part('driving'), part('charging'), part('parked'));
  };

  async function load() {
    seg.replaceChildren(...RANGES.map(([n, t]) => h('button', { class: n === days ? 'on' : '', onclick: () => { days = n; load(); } }, t)));
    const d = await api(`/api/time-use?days=${days}`);
    const total = d.driving + d.charging + d.parked;
    if (total <= 0) return box.replaceChildren(h('p', { class: 'muted' }, 'Not enough history yet.'));
    const legend = (k, label) => h('div', { class: 'row' }, h('span', { class: 'grow' }, h('span', { style: `display:inline-block;width:12px;height:12px;border-radius:3px;margin-right:8px;background:${COLORS[k]};border:1px solid var(--line)` }), label),
      h('span', {}, h('b', {}, pct(d[k], total)), ' ', h('span', { class: 'muted' }, hours(d[k]))));
    const daily = d.days.slice(-30);
    box.replaceChildren(...[
      h('section', { class: 'card' }, h('h2', {}, 'Where the time went'), bar(d, 22), h('div', { style: 'height:8px' }),
        legend('parked', 'Parked'), legend('driving', 'Driving'), fuelCar ? null : legend('charging', 'Charging')),
      d.places.length ? h('section', { class: 'card' }, h('h2', {}, 'Parked at'),
        ...d.places.map((p) => h('div', { class: 'row' }, h('span', { class: 'grow' }, p.name), h('span', {}, h('b', {}, pct(p.sec, d.parked)), ' ', h('span', { class: 'muted' }, hours(p.sec))))),
        h('p', { class: 'muted' }, 'Name places under More, Places to see Home and the office here instead of Elsewhere.')) : null,
      daily.length > 1 ? h('section', { class: 'card' }, h('h2', {}, `Day by day${d.days.length > 30 ? ' (last 30 days)' : ''}`),
        ...daily.map((x) => h('div', { style: 'display:flex;gap:8px;align-items:center;padding:3px 0' },
          h('span', { class: 'muted', style: 'width:4.5em;font-size:.8rem' }, new Date(`${x.date}T12:00:00`).toLocaleDateString([], { weekday: 'short', day: 'numeric' })),
          h('div', { style: 'flex:1' }, bar(x, 10)), h('span', { class: 'muted', style: 'width:3.2em;text-align:right;font-size:.8rem' }, hours(x.driving + x.charging)))),
        h('p', { class: 'muted' }, fuelCar ? 'The number on the right is the time driving that day.' : 'The number on the right is the time driving or charging that day.')) : null].filter(Boolean));
  }
  await load();
}
