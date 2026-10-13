import { h } from './ui.js';
import { api } from './api.js';

// On a trip: name where it started or ended, if that is not a named place yet (admins).
export function placeCard(t) {
  const needStart = !t.startPlace && t.startLat != null;
  const needEnd = !t.endPlace && t.endLat != null;
  if (!needStart && !needEnd) return null;
  const name = h('input', { id: 'pcn', placeholder: 'For example Office', maxlength: 40, 'aria-label': 'Place name' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const add = (end) => async () => {
    try { await api('/api/named-places', { method: 'POST', body: { name: name.value, tripId: t.id, end } }); location.reload(); } catch (e) { msg.textContent = e.message; }
  };
  return h('section', { class: 'card' }, h('h2', {}, 'Name a place'),
    h('p', { class: 'muted' }, 'Give this trip\'s start or end a name. Trips from or to it will show the name, and you can get alerts when the car arrives or leaves.'),
    name,
    h('p', { class: 'seg' }, needStart ? h('button', { class: 'sec', onclick: add('start') }, 'Name the start') : null, needEnd ? h('button', { class: 'sec', onclick: add('end') }, 'Name the end') : null), msg);
}
