import { h } from '../ui.js';
import { api } from '../api.js';

// Places you have named, such as Home or the office. Trips show the names, and you can get an alert on
// arriving and leaving (More, Notifications).
export default async function places(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Places')),
    h('p', { class: 'muted' }, 'Name the places you go to often. Trips then read "Home → Office" instead of just a distance, and you can get an alert when the car arrives or leaves (turn them on under More, Notifications). You can also name a place from any trip.'), box);

  async function load() {
    const ps = await api('/api/named-places');
    const name = h('input', { id: 'pn', placeholder: 'For example Home', maxlength: 40 });
    const size = h('select', { id: 'pr' }, ...[[100, 'Small (100 m)'], [150, 'Normal (150 m)'], [300, 'Large (300 m)'], [600, 'Very large (600 m)']].map(([v, t]) => h('option', { value: v, selected: v === 150 }, t)));
    const msg = h('p', { class: 'muted', role: 'status' });
    async function here() {
      try { await api('/api/named-places', { method: 'POST', body: { name: name.value, radiusM: Number(size.value), here: true } }); load(); } catch (e) { msg.textContent = e.message; }
    }
    box.replaceChildren(...[
      h('section', { class: 'card' }, h('h2', {}, 'Your places'),
        ...(ps.length ? ps.map((p) => h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, p.name), h('div', { class: 'muted' }, `within about ${p.radiusM} m`)),
          admin ? h('button', { class: 'sec', onclick: async () => { if (confirm(`Remove ${p.name}?`)) { await api(`/api/named-places/${p.id}`, { method: 'DELETE' }); load(); } } }, 'Remove') : null))
          : [h('p', { class: 'muted' }, 'No places yet.')])),
      admin ? h('section', { class: 'card' }, h('h2', {}, 'Add where the car is now'),
        h('label', { for: 'pn' }, 'Name'), name, h('label', { for: 'pr' }, 'How close counts as there'), size,
        h('p', {}, h('button', { onclick: here }, 'Save this place')), msg,
        h('p', { class: 'muted' }, 'Uses the car\'s last reported position, so do it while the car is there. Or open a trip and name its start or end.')) : null].filter(Boolean));
  }
  await load();
}
