import { h } from './ui.js';
import { api } from './api.js';

// A route of the route planner kept on the server ("kind": "route" among the saved trips), so a route made on a computer opens
// on the phone. The server clears one two weeks after it was saved.
export const isRoute = (t) => { try { return JSON.parse(t.body).kind === 'route'; } catch { return false; } };

const dayName = (d) => d.toLocaleDateString([], { weekday: 'long', month: 'short', day: 'numeric' });

// st: { stops, when, best, back, draw(), note(text) } from the route planner.
export function savedRoutes(st) {
  const box = h('div');
  async function load() {
    try {
      const l = (await api('/api/saved-trips')).filter(isRoute);
      box.replaceChildren(l.length ? h('details', { class: 'card', style: 'margin:8px 0;padding:10px', open: true }, h('summary', {}, h('b', {}, `Saved routes (${l.length})`)),
        ...l.map(row), h('p', { class: 'muted', style: 'margin:6px 0 0' }, 'Saved routes open on any of your devices. Each is cleared two weeks after it was saved.')) : '');
    } catch { box.replaceChildren(); }
  }
  const row = (t) => {
    let n = 0;
    try { n = JSON.parse(t.body).stops.length - 1; } catch { /* counted as none */ }
    return h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap;align-items:center;margin-top:6px' },
      h('div', { style: 'flex:1 1 10em;min-width:0;overflow-wrap:anywhere' }, h('b', {}, t.name), h('span', { class: 'muted' }, ` · ${n} stop${n === 1 ? '' : 's'}`)),
      h('div', { class: 'seg' }, h('button', { class: 'sec', type: 'button', onclick: () => open(t) }, 'Open'),
        h('button', { class: 'text sm', type: 'button', onclick: async () => { if (confirm(`Delete ${t.name}?`)) { await api(`/api/saved-trips/${t.id}`, { method: 'DELETE' }).catch(() => {}); load(); } } }, 'Delete')));
  };
  function open(t) {
    try {
      const b = JSON.parse(t.body);
      if (!Array.isArray(b.stops) || b.stops.length < 2) throw new Error('empty');
      st.stops.splice(0, st.stops.length, ...b.stops.map((s) => ({ name: s.name || '', label: s.label || s.name || '', lat: s.lat, lon: s.lon })));
      if (b.when && new Date(b.when).getTime() > Date.now()) st.when.value = b.when; // a time already gone stays at now
      st.best.checked = !!b.best; st.back.checked = !!b.back;
      st.draw(); st.note(`Opened ${t.name}. Plan the route to see today's battery and weather.`);
    } catch { st.note('That saved route could not be opened.'); }
  }
  async function save() {
    if (st.stops.some((s) => s.lat == null)) { st.note('Find every stop before saving the route.'); return; }
    const d = new Date(st.when.value);
    const name = prompt('Name this route', dayName(Number.isNaN(d.getTime()) ? new Date() : d));
    if (!name || !name.trim()) return;
    const body = { kind: 'route', stops: st.stops.map((s) => ({ name: s.name, label: s.label, lat: s.lat, lon: s.lon })), when: st.when.value, best: st.best.checked, back: st.back.checked };
    try { await api('/api/saved-trips', { method: 'POST', body: { name: name.trim().slice(0, 80), body } }); st.note(`Saved as ${name.trim()}. Open it on any of your devices from Saved routes.`); load(); } catch (e) { st.note(e.message); }
  }
  load();
  return { el: box, save };
}
