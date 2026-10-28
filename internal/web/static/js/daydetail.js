import { h, stat, dist, speed, num, pct } from './ui.js';
import { api } from './api.js';
import { effSpan } from './energy.js';
import { curveBox } from './curve.js';
import { whyCard } from './tripwhy.js';
import { catLabel } from './categories.js';

const hm = (sec) => (sec < 3600 ? `${Math.max(1, Math.round(sec / 60))} min` : `${Math.floor(sec / 3600)} h ${Math.round((sec % 3600) / 60)} min`);

// What a drive adds to its row: the numbers worth a second look, and why the efficiency was what it was.
async function tripPanel(e) {
  const { trip: t } = await api(`/api/trips/${e.id}?samples=0`);
  const secs = (t.endTs || Date.now() / 1000) - t.startTs;
  const avg = secs > 0 ? (t.distanceKm / secs) * 3600 : null;
  const tiles = h('div', { class: 'dtiles day' },
    stat('Distance', dist(t.distanceKm, 1)),
    stat('Average speed', speed(avg)),
    stat('Top speed', speed(t.maxSpeedKph)),
    t.energyKwh != null ? stat('Energy used', `${num(t.energyKwh, 1, ' kWh')}${t.energyEstimated ? ' (estimate)' : ''}`) : null,
    t.energyKwh != null && t.distanceKm > 0 ? stat('Efficiency', effSpan(t.distanceKm, t.energyKwh)) : null,
    stat('Battery', `${pct(t.startSoc)} to ${pct(t.endSoc)}`));
  const why = await whyCard(e.id).catch(() => null);
  return [tiles, t.category ? h('p', { class: 'muted' }, `${catLabel(t.category)}${t.notes ? `: ${t.notes}` : ''}`) : null, why, h('p', { class: 'seg' }, h('a', { class: 'btn sec', href: `#/trip/${e.id}` }, 'Open the whole trip'), h('a', { class: 'btn sec', href: `#/trip/${e.id}/share` }, 'Share this trip'))];
}

// "What does a kWh cost here?" for a public charge priced from the general price. Saving it gives the place its own price (a place
// you named, or a new one at this charger), and every charge there, past and future, is worked out with it.
function placePrice(e, cur) {
  const kwh = e.wallKwh || e.addedKwh;
  const now = e.cost != null && kwh > 0 ? (e.cost / kwh).toFixed(2) : '';
  const input = h('input', { type: 'number', step: '0.01', min: 0, inputmode: 'decimal', value: now, 'aria-label': 'Price per kWh at this charger', style: 'width:7em' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const save = async () => {
    const price = Number(input.value);
    if (!(price >= 0) || input.value === '') { msg.textContent = 'Enter the price per kWh, for example 0.48.'; return; }
    try {
      const named = await api('/api/named-places');
      const mine = named.find((p) => p.name === e.place);
      if (mine) await api(`/api/named-places/${mine.id}`, { method: 'PATCH', body: { chargePrice: price } });
      else await api('/api/named-places', { method: 'POST', body: { name: `${e.place || 'Charger'} charger`, lat: e.lat, lon: e.lon, radiusM: 150, chargePrice: price } });
      msg.textContent = `Saved. Charging here now costs ${cur}${price.toFixed(2)} a kWh; the day is worked out again.`;
      setTimeout(() => dispatchEvent(new Event('hashchange')), 900);
    } catch (err) { msg.textContent = err.message; }
  };
  return h('div', { class: 'card', style: 'margin:8px 0' }, h('b', {}, 'What does a kWh cost here?'),
    h('p', { class: 'muted' }, `This charge is an estimate from your general price. Give this charger its own price and every charge here uses it.`),
    h('div', { style: 'display:flex;flex-wrap:wrap;align-items:center;gap:8px;margin:8px 0' }, h('span', {}, cur), input, h('button', { type: 'button', onclick: (ev) => { ev.stopPropagation(); save(); } }, 'Save for this place')), msg);
}

// What a charging session adds: where, how much and what it cost, and the power curve.
function chargePanel(e, cur, admin) {
  const where = { home: 'at home', public: 'away from home' }[e.where];
  const lines = [e.addedKwh != null ? `${num(e.addedKwh, 1, ' kWh')} added` : null, e.cost != null ? `${cur}${e.cost.toFixed(2)}` : null, where, `battery ${pct(e.startSoc)} to ${pct(e.endSoc)}`].filter(Boolean);
  const curve = e.end == null ? h('p', { class: 'muted' }, 'Still charging. The power curve appears when it ends.') : curveBox(e.id);
  if (curve.tagName === 'DETAILS') curve.open = true;
  const g = e.gas && e.cost != null ? e.gas : null; // what the same range would have cost in the gas car
  const gasLine = g ? h('p', {}, `The same ${dist(g.km, 0)} in the gas car would have cost about ${cur}${g.cost.toFixed(2)}${g.source ? ` (${g.source})` : ''}, so ${g.cost >= e.cost ? `this charge saved about ${cur}${(g.cost - e.cost).toFixed(2)}` : `this charge cost about ${cur}${(e.cost - g.cost).toFixed(2)} more`}.`) : null;
  return [h('p', {}, lines.join(' · ')), gasLine, admin && e.where !== 'home' && !e.costActual && e.lat != null ? placePrice(e, cur) : null, curve, h('p', {}, h('a', { href: '#/charging' }, 'Open Charging, to enter what you paid'))];
}

// Turns a row of the day into one that opens its details just below it, so nothing takes you off the day.
export function expandable(row, e, cur, admin) {
  if (e.kind === 'parked' || e.kind === 'remote') return [row];
  row.removeAttribute('href');
  row.setAttribute('role', 'button');
  row.setAttribute('tabindex', '0');
  row.setAttribute('aria-expanded', 'false');
  row.classList.add('hopen');
  const panel = h('div', { class: 'hpanel', hidden: true });
  let loaded = false;
  const toggle = async () => {
    const open = panel.hidden;
    panel.hidden = !open;
    row.setAttribute('aria-expanded', String(open));
    if (!open || loaded) return;
    loaded = true;
    panel.replaceChildren(h('p', { class: 'muted' }, 'Loading…'));
    try {
      const kids = e.kind === 'driving' ? await tripPanel(e) : chargePanel(e, cur, admin);
      panel.replaceChildren(...kids.filter(Boolean));
    } catch (err) { loaded = false; panel.replaceChildren(h('p', { class: 'muted' }, err.message || 'Could not load the details.')); }
  };
  row.addEventListener('click', toggle);
  row.addEventListener('keydown', (ev) => { if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); toggle(); } });
  return [row, panel];
}
