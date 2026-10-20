import { h, emptyState, shortDist } from '../ui.js';
import { api } from '../api.js';

// What charging costs at a place: its own price per kWh (0 when it is free), or counts as home. Left alone, a place uses the
// usual home and public prices from Charging settings.
function chargeBox(p, reload) {
  const price = h('input', { type: 'number', step: '0.01', min: 0, inputmode: 'decimal', value: p.chargePrice == null ? '' : p.chargePrice, placeholder: 'usual price', style: 'width:9em', 'aria-label': `Price per kWh at ${p.name}` });
  const home = h('input', { type: 'checkbox', checked: p.chargeHome, id: `ph${p.id}` });
  const msg = h('span', { class: 'muted', role: 'status' });
  const save = async () => {
    try {
      const body = { chargeHome: home.checked };
      if (price.value === '') body.clearPrice = true; else body.chargePrice = Number(price.value);
      await api(`/api/named-places/${p.id}`, { method: 'PATCH', body });
      reload();
    } catch (e) { msg.textContent = e.message; }
  };
  const summary = p.chargePrice != null ? `Charging here: ${p.chargePrice === 0 ? 'free' : `${p.chargePrice.toFixed(2)} per kWh`}` : p.chargeHome ? 'Charging here counts as home' : 'Charging here: the usual price';
  return h('details', { class: 'curvebox', style: 'margin:0 0 8px' }, h('summary', { class: 'muted' }, summary),
    h('p', { class: 'muted' }, 'Leave the price empty to use your usual prices (Charging, Charging settings). Enter 0 if charging here is free.'),
    h('label', {}, 'Price per kWh here'), price,
    h('label', { for: `ph${p.id}`, style: 'display:flex;gap:8px;align-items:center;margin-top:8px' }, home, 'Counts as home charging (home price, and home in the statistics)'),
    h('p', {}, h('button', { onclick: save }, 'Save'), ' ', msg));
}

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
    const size = h('select', { id: 'pr' }, ...[[100, 'Small'], [150, 'Normal'], [300, 'Large'], [600, 'Very large']].map(([m, n]) => [m, `${n} (${shortDist(m)})`]).map(([v, t]) => h('option', { value: v, selected: v === 150 }, t)));
    const msg = h('p', { class: 'muted', role: 'status' });
    async function here() {
      try { await api('/api/named-places', { method: 'POST', body: { name: name.value, radiusM: Number(size.value), here: true } }); load(); } catch (e) { msg.textContent = e.message; }
    }
    box.replaceChildren(...[
      h('section', { class: 'card' }, h('h2', {}, 'Your places'),
        ...(ps.length ? ps.map((p) => h('div', {}, h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, p.name), h('div', { class: 'muted' }, `within about ${shortDist(p.radiusM)}`)),
          admin ? h('button', { class: 'sec', onclick: async () => { if (confirm(`Remove ${p.name}?`)) { await api(`/api/named-places/${p.id}`, { method: 'DELETE' }); load(); } } }, 'Remove') : null), admin ? chargeBox(p, load) : null))
          : [emptyState('No places yet', 'Drive to Home or the office and use the form below while the car is there.')])),
      admin ? h('section', { class: 'card' }, h('h2', {}, 'Add where the car is now'),
        h('label', { for: 'pn' }, 'Name'), name, h('label', { for: 'pr' }, 'How close counts as there'), size,
        h('p', {}, h('button', { onclick: here }, 'Save this place')), msg,
        h('p', { class: 'muted' }, 'Uses the car\'s last reported position, so do it while the car is there. Or open a trip and name its start or end.')) : null].filter(Boolean));
  }
  await load();
}
