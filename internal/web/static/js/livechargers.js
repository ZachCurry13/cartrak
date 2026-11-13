import { h, dist, ago } from './ui.js';
import { api } from './api.js';

// How many plugs are free at a charger right now, from Google (which gets it from the networks that report it). Asked only when
// someone taps, since each question to Google is billed; the server keeps each answer for five minutes.

const plugLine = (k) => {
  const kind = `${k.type}${k.kw ? ` ${Math.round(k.kw)} kW` : ''}`;
  if (k.available == null) return `${kind}: ${k.count} plug${k.count === 1 ? '' : 's'} (the network does not report which are free)`;
  const out = k.outOfService ? `, ${k.outOfService} out of service` : '';
  return `${kind}: ${k.available} of ${k.count} free${out}`;
};

function chargerRow(c) {
  const fresh = c.connectors.map((k) => k.updated || 0).reduce((a, b) => Math.max(a, b), 0);
  const free = c.connectors.some((k) => k.available > 0), known = c.connectors.some((k) => k.available != null);
  return h('div', { style: 'border-top:1px solid var(--line);padding:8px 0' },
    h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap;align-items:baseline' },
      h('b', { style: 'flex:1 1 10em;min-width:0;overflow-wrap:anywhere' }, c.name),
      known ? h('span', { class: `chip ${free ? 'ok' : 'bad'}` }, free ? 'Free plugs' : 'All busy') : null),
    h('div', { class: 'muted', style: 'overflow-wrap:anywhere' }, [dist(c.distKm, 1), c.address].filter(Boolean).join(' · ')),
    ...c.connectors.map((k) => h('div', {}, plugLine(k))),
    fresh ? h('div', { class: 'muted' }, `Reported ${ago(fresh)}`) : null);
}

// liveCheck is a "Free plugs now?" button for one place (a planned charging stop): the chargers within radius meters of it.
export function liveCheck(lat, lon, radius = 400, most = 2) {
  const out = h('div');
  const btn = h('button', { class: 'text sm', type: 'button' }, 'Free plugs now?');
  btn.onclick = async () => {
    btn.disabled = true;
    out.replaceChildren(h('p', { class: 'muted', role: 'status' }, 'Asking...'));
    try {
      const r = await api(`/api/chargers/live?lat=${lat}&lon=${lon}&r=${radius}`);
      out.replaceChildren(...(r.chargers.length ? r.chargers.slice(0, most).map(chargerRow) : [h('p', { class: 'muted' }, 'Google lists no charger right there.')]));
    } catch (e) { out.replaceChildren(h('p', { class: 'muted' }, e.message)); }
    btn.disabled = false;
    btn.textContent = 'Check again';
  };
  return h('div', {}, btn, out);
}

// nearCarCard is the Charging page's "Chargers near the car": up to ten within about 5 miles, on a tap.
export function nearCarCard(latest) {
  if (!latest || latest.lat == null || latest.lon == null) return null;
  const out = h('div');
  const btn = h('button', { class: 'sec', type: 'button' }, 'Check now');
  btn.onclick = async () => {
    btn.disabled = true;
    out.replaceChildren(h('p', { class: 'muted', role: 'status' }, 'Asking Google...'));
    try {
      const r = await api(`/api/chargers/live?lat=${latest.lat}&lon=${latest.lon}&r=8000`);
      out.replaceChildren(...(r.chargers.length ? r.chargers.map(chargerRow) : [h('p', { class: 'muted' }, 'Google lists no chargers within about 5 miles.')]));
    } catch (e) { out.replaceChildren(h('p', { class: 'muted' }, e.message)); }
    btn.disabled = false;
    btn.textContent = 'Check again';
  };
  return h('section', { class: 'card' }, h('h2', {}, 'Chargers near the car'),
    h('p', { class: 'muted' }, 'Which chargers nearby have a free plug right now, as their networks report it to Google. Needs the Google Places key (More, Integrations); each check is one billed lookup.'),
    h('p', {}, btn), out);
}
