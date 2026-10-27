import { h } from './ui.js';
import { api } from './api.js';
import { stopRow, minutesText } from './stopcard.js';

// The parked time on one day, under the day's tiles: each stop with its place and length, and for an admin the label and note.
export async function dayStopsCard(date, admin) {
  const get = () => api(`/api/stops?from=${date}&to=${date}`);
  const d = await get().catch(() => null);
  if (!d || !d.stops.length) return null;
  const card = h('section', { class: 'card' });
  const draw = (x) => card.replaceChildren(h('h2', {}, 'Parked time'), h('p', { class: 'muted' }, `${minutesText(x.minutes)} parked between trips, ${minutesText(x.workMinutes)} of it work.`),
    ...x.stops.map((s) => stopRow(s, admin, x.tz, () => get().then(draw))), h('p', {}, h('a', { href: '#/stops' }, 'All work stops')));
  draw(d);
  return card;
}
