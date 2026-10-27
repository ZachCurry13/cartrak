import { h, chip } from './ui.js';
import { api } from './api.js';
import { categoryList, catLabel } from './categories.js';

// A stop is the time the car sat parked between two trips, such as time in a customer's parking lot. It takes the category of the trip
// that arrived there; an admin can change it and add a note (a customer's name) here.

export const minutesText = (m) => (m >= 60 ? `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, '0')} min` : `${m} min`);
const clock = (ts, tz) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', timeZone: tz });
export const dayText = (ts, tz) => new Date(ts * 1000).toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric', timeZone: tz });

function editor(s, reload) {
  const cat = h('select', { 'aria-label': 'Category' }, h('option', { value: '' }, 'Same as the trip'), ...categoryList().map((c) => h('option', { value: c.key }, c.label)));
  cat.value = s.manual ? s.category : '';
  const note = h('input', { type: 'text', maxlength: '200', placeholder: 'Customer or note', value: s.note || '', 'aria-label': 'Note' });
  const msg = h('span', { class: 'muted', role: 'status' });
  const save = () => api(`/api/stops/${s.tripId}`, { method: 'PUT', body: { category: cat.value, note: note.value } }).then(reload).catch((e) => { msg.textContent = e.message; });
  return h('details', {}, h('summary', { style: 'min-height:44px;display:flex;align-items:center;cursor:pointer' }, 'Label this stop'), h('div', { class: 'cols2', style: 'margin-top:6px' }, cat, note),
    h('p', { class: 'seg' }, h('button', { type: 'button', onclick: save }, 'Save'), msg));
}

export function stopRow(s, admin, tz, reload) {
  const where = s.place || 'an unnamed place';
  return h('div', { class: 'row', style: 'align-items:flex-start;flex-wrap:wrap' },
    h('span', { class: 'grow', style: 'flex-basis:12em;min-width:0' }, h('b', {}, minutesText(s.minutes)), s.open ? h('span', { class: 'muted' }, ' · still here') : null,
      h('div', { class: 'muted' }, `${dayText(s.start, tz)} · ${clock(s.start, tz)} to ${s.open ? 'now' : clock(s.end, tz)}`),
      h('div', {}, where), s.note ? h('div', { class: 'muted' }, s.note) : null),
    chip(s.category === 'work' ? 'work' : 'info', catLabel(s.category)),
    admin ? h('div', { style: 'flex-basis:100%' }, editor(s, reload)) : null);
}
