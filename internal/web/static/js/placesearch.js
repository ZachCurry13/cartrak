import { h, shortDist } from './ui.js';
import { api } from './api.js';
import { matchDetail } from './placematch.js';

const SIZES = [[100, 'Small'], [150, 'Normal'], [300, 'Large'], [600, 'Very large']];

// Add a place by typing its address: search, pick the right match, name it. Only the coordinates are kept, never the address.
export function addressCard(onSaved) {
  const q = h('input', { id: 'pa', type: 'search', placeholder: 'A business name, or street and town', maxlength: 200, autocomplete: 'off' });
  const name = h('input', { id: 'pan', placeholder: 'For example Customer', maxlength: 40 });
  const size = h('select', { id: 'pas' }, ...SIZES.map(([v, n]) => h('option', { value: v, selected: v === 150 }, `${n} (${shortDist(v)})`)));
  const msg = h('p', { class: 'muted', role: 'status' });
  const list = h('div');
  let chosen = null;

  const pick = (m) => { chosen = m; if (!name.value) name.value = m.name; draw(); };
  const draw = () => list.replaceChildren(...found.map((m) => h('label', { class: 'row', style: 'gap:8px;align-items:center;margin:4px 0' },
    h('input', { type: 'radio', name: 'pam', checked: chosen === m, onchange: () => pick(m) }), h('span', { class: 'grow', style: 'min-width:0;overflow-wrap:anywhere' }, h('b', {}, m.name), h('div', { class: 'muted' }, matchDetail(m))))));
  let found = [];

  async function search() {
    msg.textContent = 'Searching...'; found = []; chosen = null; draw();
    try {
      found = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`);
      msg.textContent = found.length ? 'Pick the right one, name it and save.' : 'Nothing found. Add the town to the name, or type the street address.';
      if (found.length === 1) chosen = found[0];
      draw();
    } catch (e) { msg.textContent = e.message; }
  }
  async function save() {
    if (!chosen) { msg.textContent = 'Search and pick an address first.'; return; }
    try {
      await api('/api/named-places', { method: 'POST', body: { name: name.value, radiusM: Number(size.value), lat: chosen.lat, lon: chosen.lon } });
      onSaved();
    } catch (e) { msg.textContent = e.message; }
  }
  q.addEventListener('keydown', (e) => { if (e.key === 'Enter') search(); });
  return h('section', { class: 'card' }, h('h2', {}, 'Add a business or address'),
    h('label', { for: 'pa' }, 'Business or address'), q, h('p', {}, h('button', { class: 'sec', onclick: search }, 'Search')), list,
    h('label', { for: 'pan' }, 'Name'), name, h('label', { for: 'pas' }, 'How close counts as there'), size,
    h('p', {}, h('button', { onclick: save }, 'Save this place')), msg,
    h('p', { class: 'muted' }, 'The address is sent to OpenStreetMap to find it (the same service as city names, and off when City names is off). Only the spot is saved, not the address.'));
}
