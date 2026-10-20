import { h, stat, dist, speed, num, pct } from './ui.js';
import { api } from './api.js';
import { effSpan } from './energy.js';
import { curveBox } from './curve.js';
import { whyCard } from './tripwhy.js';

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
  return [tiles, t.category ? h('p', { class: 'muted' }, `${t.category === 'work' ? 'Work' : 'Personal'}${t.notes ? `: ${t.notes}` : ''}`) : null, why, h('p', {}, h('a', { href: `#/trip/${e.id}` }, 'Open the whole trip, with its map and edit options'))];
}

// What a charging session adds: where, how much and what it cost, and the power curve.
function chargePanel(e, cur) {
  const where = { home: 'at home', public: 'away from home' }[e.where];
  const lines = [e.addedKwh != null ? `${num(e.addedKwh, 1, ' kWh')} added` : null, e.cost != null ? `${cur}${e.cost.toFixed(2)}` : null, where, `battery ${pct(e.startSoc)} to ${pct(e.endSoc)}`].filter(Boolean);
  const curve = e.end == null ? h('p', { class: 'muted' }, 'Still charging. The power curve appears when it ends.') : curveBox(e.id);
  if (curve.tagName === 'DETAILS') curve.open = true;
  return [h('p', {}, lines.join(' · ')), curve, h('p', {}, h('a', { href: '#/charging' }, 'Open Charging, to enter what you paid'))];
}

// Turns a row of the day into one that opens its details just below it, so nothing takes you off the day.
export function expandable(row, e, cur) {
  if (e.kind === 'parked') return [row];
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
      const kids = e.kind === 'driving' ? await tripPanel(e) : chargePanel(e, cur);
      panel.replaceChildren(...kids.filter(Boolean));
    } catch (err) { loaded = false; panel.replaceChildren(h('p', { class: 'muted' }, err.message || 'Could not load the details.')); }
  };
  row.addEventListener('click', toggle);
  row.addEventListener('keydown', (ev) => { if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); toggle(); } });
  return [row, panel];
}
