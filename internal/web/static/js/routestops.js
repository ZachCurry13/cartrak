// "Add a stop" on a trip with an estimated road: the car recorded too few positions to show a detour, so say where it went (Elk River,
// a customer, a store) and the road is worked out again through those places. Up to five stops, kept with the trip.
import { h } from './ui.js';
import { api } from './api.js';

export function routeStopsBox(t, onChanged) {
  const list = h('div', { class: 'tags' }), found = h('div'), msg = h('p', { class: 'muted', role: 'status' });
  let stops = [];
  const q = h('input', { type: 'search', placeholder: 'A town, business or address', maxlength: 200, autocomplete: 'off', 'aria-label': 'Search for a stop' });

  async function save(next) {
    msg.textContent = 'Working out the road through your stops…';
    try {
      await api(`/api/trips/${t.id}/route-stops`, { method: 'POST', body: { stops: next } });
      stops = next;
      const d = await api(`/api/trips/${t.id}/route-options`); // the road closest to the odometer comes first
      await api(`/api/trips/${t.id}/route-choice`, { method: 'POST', body: { kind: d.options[0].kind } });
      onChanged();
    } catch (e) { msg.textContent = e.message; draw(); }
  }
  const draw = () => list.replaceChildren(...stops.map((s, i) => h('span', { class: 'chip' }, s.name || 'Stop', ' ',
    h('button', { type: 'button', class: 'text sm', 'aria-label': `Remove ${s.name || 'stop'}`, onclick: () => save(stops.filter((_, j) => j !== i)) }, '×'))));

  async function search() {
    if (!q.value.trim()) return;
    msg.textContent = 'Searching…';
    found.replaceChildren();
    try {
      const r = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`);
      msg.textContent = r.length ? 'Tap the place the car went to.' : 'Not found. Add the town, or type the address.';
      found.replaceChildren(...r.map((m) => h('button', { type: 'button', class: 'sec', style: 'display:block;width:100%;text-align:left;margin:4px 0', onclick: () => save([...stops, { name: m.name, lat: m.lat, lon: m.lon }]) }, m.name)));
    } catch (e) { msg.textContent = e.message; }
  }
  q.addEventListener('keydown', (e) => { if (e.key === 'Enter') search(); });

  const box = h('details', { class: 'curvebox', style: 'margin:8px 0 0' }, h('summary', { class: 'muted' }, 'Add a stop the car did not report'),
    h('p', { class: 'muted' }, 'Name a place the trip went through (up to five) and the road is worked out again through it.'),
    list, h('div', { class: 'row', style: 'border:0;padding:0' }, q, h('button', { type: 'button', class: 'sec', onclick: search }, 'Search')), found, msg);
  api(`/api/trips/${t.id}/route-stops`).then((d) => { stops = d.stops || []; draw(); if (stops.length) box.open = true; }).catch(() => { /* not available: the box still opens */ });
  return box;
}
