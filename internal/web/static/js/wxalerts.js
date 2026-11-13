import { h } from './ui.js';
import { api } from './api.js';

// Weather warnings from the National Weather Service (when switched on under Integrations): where the car is, on Home, and at the
// stops of a planned route.

const level = (a) => (a.severity === 'Extreme' || a.severity === 'Severe' ? 'bad' : 'work');
const until = (a) => (a.ends ? ` until ${new Date(a.ends * 1000).toLocaleString([], { weekday: 'short', hour: 'numeric', minute: '2-digit' })}` : '');

export const warningLine = (a, where) => h('p', { class: `banner ${level(a) === 'bad' ? '' : 'notice'}`, style: 'overflow-wrap:anywhere' },
  h('b', {}, a.event), where ? ` at ${where}` : '', until(a), a.headline ? h('div', { class: 'muted' }, a.headline) : null);

// carWarnings is the warnings where the car is, or [] when they are off or there are none.
export async function carWarnings() {
  const r = await api('/api/weather-alerts').catch(() => null);
  if (!r || !r.on || !r.places.length) return [];
  return r.places[0].alerts;
}

// routeWarnings fills in the road warnings at a route's stops (each warning once, at the first stop it covers).
export function routeWarnings(stops) {
  const box = h('div');
  const pts = stops.filter((s) => s.lat != null).slice(0, 12);
  if (!pts.length) return box;
  api(`/api/weather-alerts?points=${pts.map((s) => `${s.lat.toFixed(3)},${s.lon.toFixed(3)}`).join(';')}`).then((r) => {
    if (!r || !r.on) return;
    const seen = new Set(), out = [];
    r.places.forEach((p, i) => p.alerts.filter((a) => a.drive && !seen.has(a.event)).forEach((a) => { seen.add(a.event); out.push(warningLine(a, pts[i].name)); }));
    box.replaceChildren(...out);
  }).catch(() => {});
  return box;
}
