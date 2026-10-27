import { h } from '../ui.js';
import { api } from '../api.js';
import { withCar } from '../car.js';
import { stopRow, minutesText, dayText } from '../stopcard.js';

const RANGES = [[7, '7 days'], [30, '30 days'], [90, '90 days']];
const ymd = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

// Work stops: the time the car sat parked between trips (a customer's parking lot, say), with what each stop was and how long, so the
// time between drives is on record too.
export default async function stops(root, _a, session) {
  const admin = session.me.role === 'admin';
  let days = 30;
  const seg = h('div', { class: 'seg' });
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Work stops')),
    h('p', { class: 'muted' }, 'The time the car sat parked between trips, such as in a customer\'s parking lot. A stop takes the category of the trip that arrived there; change it or add a note under Label this stop. Stops at Home, while charging, or over 8 hours are left out.'), seg, box);

  async function load() {
    seg.replaceChildren(...RANGES.map(([n, t]) => h('button', { class: n === days ? 'on' : '', onclick: () => { days = n; load(); } }, t)));
    const from = new Date(); from.setDate(from.getDate() - days + 1);
    const q = `from=${ymd(from)}&to=${ymd(new Date())}`;
    const d = await api(`/api/stops?${q}`);
    if (!d.stops.length) return box.replaceChildren(h('p', { class: 'muted' }, 'No stops in this time yet. They appear on their own after a trip, once the car has been parked for 10 minutes.'));
    const tz = d.tz;
    box.replaceChildren(
      h('section', { class: 'card' }, h('h2', {}, 'Time parked'),
        h('div', { class: 'bigfig' }, h('div', {}, h('b', {}, minutesText(d.workMinutes)), h('span', {}, 'work')), h('div', {}, h('b', {}, minutesText(d.minutes)), h('span', {}, `all, in ${d.stops.length} stop${d.stops.length === 1 ? '' : 's'}`))),
        ...d.days.slice(0, 14).map((x) => h('div', { class: 'row' }, h('span', { class: 'grow' }, dayText(new Date(`${x.date}T12:00:00`).getTime() / 1000, tz)), h('span', {}, h('b', {}, minutesText(x.work)), h('span', { class: 'muted' }, ` work of ${minutesText(x.minutes)}`)))),
        h('p', {}, h('a', { class: 'btn sec', href: withCar(`/api/stops.csv?${q}`) }, 'Export CSV'))),
      h('section', { class: 'card' }, h('h2', {}, 'Stops'), ...d.stops.map((s) => stopRow(s, admin, tz, load))));
  }
  await load();
}
