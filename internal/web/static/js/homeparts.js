import { tripPlaces } from './tripplaces.js';
import { h, stat, num, dist, ago } from './ui.js';
import { effSpan, kwhText } from './energy.js';
import { pick } from './pick.js';

// Small pieces of the Home page, kept here so dashboard.js stays short.

// The charger's current. A DC fast charge reports 0 A there (that field is the car's own charger), so the figure is worked out from the
// power and the voltage and marked as approximate.
export function chargeAmps(latest, x) {
  const a = pick(x, 'XEV_BATTERY_CHARGER_CURRENT_OUTPUT'), v = pick(x, 'XEV_BATTERY_CHARGER_VOLTAGE_OUTPUT');
  if (a != null && a > 0.5) return num(a, 0, ' A');
  if (latest.chargeKw > 1 && v > 50) return `about ${Math.round((latest.chargeKw * 1000) / v)} A`;
  return null; // nothing flowing: no figure, rather than 0 A
}

// The last seven days at a glance. Tap the efficiency to change its unit.
export function weekCard(w) {
  const all = Object.values(w);
  const km = all.reduce((n, x) => n + x.distanceKm, 0);
  const trips = all.reduce((n, x) => n + x.trips, 0);
  const kwh = all.reduce((n, x) => n + x.kwh, 0);
  return card('Last 7 days', h('div', { class: 'grid' }, stat('Distance', dist(km, 0)), stat('Trips', trips),
    ...(kwh > 0 ? [stat('Energy used', kwhText(kwh)), stat('Efficiency', effSpan(all.reduce((n, x) => n + x.energyKm, 0), kwh))] : [])),
    h('p', { style: 'margin:10px 0 0' }, h('a', { href: '#/trips' }, 'See all trips')));
}

// "Around Buffalo · ", "Home → Gym · " or nothing
export function placesLine(t) {
  const p = tripPlaces(t);
  return p.around ? `Around ${p.to} · ` : p.from || p.to ? `${p.from || '…'} → ${p.to || '…'} · ` : '';
}

export function card(title, ...kids) { return h('section', { class: 'card' }, h('h2', {}, title), ...kids); }

export function wrappedTeaser() {
  const d = new Date(new Date().getFullYear(), new Date().getMonth() - 1, 1);
  const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
  return h('a', { class: 'wteaser w-teal', href: `#/wrapped/${key}`, style: 'min-height:110px;margin-bottom:12px' }, h('span', { class: 'kick' }, 'Wrapped'), h('b', {}, `Your ${d.toLocaleDateString([], { month: 'long' })} is ready ✨`), h('span', {}, 'Tap to play ▶'));
}

// Ford sends some values only when the car wakes, so say when a value is old.
export function oldNote(x, metric) {
  const t = Date.parse(pick(x, `${metric}.updateTime`) || '') / 1000;
  return t && Date.now() / 1000 - t > 7200 ? h('p', { class: 'muted' }, `These battery values are from ${ago(t)}; the car reports them when it wakes up.`) : null;
}

export function kv(rows) {
  return h('div', { class: 'kv' }, rows.filter((r) => r[1] != null).flatMap(([k, v]) => [h('span', {}, k), h('span', {}, v)]));
}
