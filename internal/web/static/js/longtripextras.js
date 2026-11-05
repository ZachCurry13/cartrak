import { h } from './ui.js';
import { createMap } from './map.js';
import { hmMin as hm } from './format.js';

const COLORS = ['#1b5fc4', '#d97706', '#7c3aed', '#0d9488', '#be185d', '#475569'];
const KIND = { start: '#059669', charge: '#0d9488', overnight: '#7c3aed', dest: '#dc2626', via: '#d97706' };

// The whole trip on a map: a line through each day's stops (straight between them: the planner draws the real roads) and the stops.
export function routeMap(mc, days) {
  if (!mc || !mc.enabled) return null;
  const m = createMap(mc, { height: 260 });
  const routes = days.map((d, i) => ({ points: d.stops.map((s) => [s.lat, s.lon]), color: COLORS[i % COLORS.length], width: 4, dash: '1 8', cap: 'round' }));
  let n = 0;
  const markers = days.flatMap((d, i) => d.stops.map((s, j) => (i && !j ? null : { lat: s.lat, lon: s.lon, color: KIND[s.kind] || KIND.charge, ...(s.kind === 'charge' ? { label: ++n } : {}) }))).filter(Boolean);
  m.show({ routes, markers });
  return h('div', {}, m.el, h('p', { class: 'muted' }, 'Straight lines between the stops. Plan a day in the planner to draw the real roads.'));
}

// Sunrise and sunset at a place on a day (NOAA's approximation), as Dates; null where the sun does not rise or set.
export function sunTimes(day, lat, lon) {
  const rad = Math.PI / 180;
  const midnight = Date.UTC(day.getFullYear(), day.getMonth(), day.getDate());
  const n = Math.floor((midnight - Date.UTC(day.getFullYear(), 0, 0)) / 86400000);
  const g = (2 * Math.PI / 365) * (n - 1);
  const eqt = 229.18 * (0.000075 + 0.001868 * Math.cos(g) - 0.032077 * Math.sin(g) - 0.014615 * Math.cos(2 * g) - 0.040849 * Math.sin(2 * g));
  const decl = 0.006918 - 0.399912 * Math.cos(g) + 0.070257 * Math.sin(g) - 0.006758 * Math.cos(2 * g) + 0.000907 * Math.sin(2 * g) - 0.002697 * Math.cos(3 * g) + 0.00148 * Math.sin(3 * g);
  const c = Math.cos(90.833 * rad) / (Math.cos(lat * rad) * Math.cos(decl)) - Math.tan(lat * rad) * Math.tan(decl);
  if (c < -1 || c > 1) return null;
  const ha = Math.acos(c) / rad;
  return { rise: new Date(midnight + (720 - 4 * (lon + ha) - eqt) * 60000), set: new Date(midnight + (720 - 4 * (lon - ha) - eqt) * 60000) };
}

const clock = (d) => d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });

// What the day's driving looks like against the daylight: when the sun sets and how much of the day is after dark.
export function daylightNote(d, i, n) {
  const a = d.stops[0], b = d.stops[d.stops.length - 1];
  const start = new Date(d.departTs * 1000), end = new Date((d.departTs + (d.driveMin + d.chargeMin) * 60) * 1000);
  const sun = sunTimes(start, (a.lat + b.lat) / 2, (a.lon + b.lon) / 2);
  if (!sun) return null;
  const dark = Math.max(0, (end - sun.set) / 60000) + Math.max(0, (sun.rise - start) / 60000);
  return h('li', {}, `${n > 1 ? `Day ${i + 1}: ` : ''}sunrise ${clock(sun.rise)}, sunset ${clock(sun.set)}. `, dark > 5 ? `About ${hm(Math.min(dark, d.driveMin + d.chargeMin))} of it is after dark.` : 'All in daylight.');
}

// When to leave to arrive by a time, for a trip that fits in one day: the arrival time minus the day's driving and charging.
export function leaveFor(arriveBy, days) {
  if (!arriveBy || days.length !== 1) return null;
  const d = days[0], at = new Date(arriveBy).getTime();
  return Number.isNaN(at) ? null : new Date(at - (d.driveMin + d.chargeMin) * 60000);
}
