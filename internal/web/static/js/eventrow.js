import { h, dist, num, speed, pct, temp } from './ui.js';
import { effSpan } from './energy.js';

const GLYPH = { charging: '⚡', driving: '↗', parked: 'P' };

// The power symbol is drawn, not typed: the character for it (U+23FB) is missing from many phone fonts and shows as a box with an x.
function powerIcon() {
  const NS = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(NS, 'svg');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('width', '16');
  svg.setAttribute('height', '16');
  const path = document.createElementNS(NS, 'path');
  path.setAttribute('d', 'M12 3v8M6.4 6.6a8 8 0 1 0 11.2 0');
  for (const [k, v] of Object.entries({ fill: 'none', stroke: 'currentColor', 'stroke-width': '2.4', 'stroke-linecap': 'round' })) path.setAttribute(k, v);
  svg.append(path);
  return svg;
}
const WHERE = { home: 'Home', public: 'Public' };
export const money = (cur, v) => `${cur}${v.toFixed(2)}`;

export function span(sec) {
  const m = Math.round(sec / 60);
  if (m < 60) return `${m} min`;
  if (m < 2880) return `${Math.floor(m / 60)} h ${m % 60} min`;
  return `${Math.floor(m / 1440)} d ${Math.floor((m % 1440) / 60)} h`;
}

// a short duration for a row's header, where space is tight: 27 min, 1h 39m
const short = (sec) => { const m = Math.round(sec / 60); return m < 60 ? `${m} min` : m < 1440 ? `${Math.floor(m / 60)}h ${m % 60}m` : span(sec); };

const clock = (ts, tz) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', timeZone: tz });

// A thin bar of the battery level: what was left in a quiet color and the part that changed in the event's own color.
function levelBar(e) {
  const a = e.startSoc, b = e.endSoc;
  if (a == null && b == null) return null;
  const lo = Math.min(a == null ? b : a, b == null ? a : b), hi = Math.max(a == null ? b : a, b == null ? a : b);
  return h('div', { class: 'hbar', role: 'img', 'aria-label': `Battery ${Math.round(a == null ? b : a)}% to ${Math.round(b == null ? a : b)}%` },
    h('i', { class: 'base', style: `width:${lo}%` }), h('i', { class: `chg ${e.kind}`, style: `width:${Math.max(1, hi - lo)}%` }));
}

// The battery at the start and the end ("98 → 51%"): on a road trip what you arrive with matters more than what was used.
function levels(e) {
  if (e.startSoc != null && e.endSoc != null) return `${Math.round(e.startSoc)} → ${pct(e.endSoc)}`;
  return e.endSoc != null ? pct(e.endSoc) : null;
}

// Where it happened: a drive from → to, a charge as home or public and its place (once, when they are the same word).
function placeOf(e) {
  if (e.kind === 'driving') return [e.fromPlace, e.place].filter(Boolean).join(' → ');
  if (e.kind === 'remote') return 'Remote start';
  if (e.kind !== 'charging') return e.place;
  const kind = WHERE[e.where] || 'Charging';
  return e.place && e.place.toLowerCase() !== kind.toLowerCase() ? `${kind} · ${e.place}` : kind;
}

// A charge's average power: a top-up of less than a kWh is called that, and slow charging gets a decimal (7.4 kW).
function power(e, secs) {
  if (!(e.addedKwh > 0) || secs <= 300) return null;
  if (e.addedKwh < 1) return 'Top-up';
  const kw = (e.addedKwh / secs) * 3600;
  return `about ${num(kw, kw < 10 ? 1 : 0, ' kW')} on average`;
}

// One row of the feed. stopNumber, when given, is the stop's number on the day map and replaces the little icon. extra can carry
// what the Day view knows from the rows around it: stopSec (the whole stop at a charger) and leftNote (a top-up folded into a drive).
export function eventRow(e, tz, cur, stopNumber, extra = {}) {
  const now = Date.now() / 1000;
  const end = e.end == null ? now : e.end;
  const secs = end - e.start;
  const place = placeOf(e);
  const detail = [];
  if (e.kind === 'driving') {
    if (extra.leftNote) detail.push(extra.leftNote);
    if (e.end != null && e.distanceKm >= 0.5 && secs > 60) detail.push(speed((e.distanceKm / secs) * 3600) + ' average', e.energyKwh > 0.2 ? effSpan(e.distanceKm, e.energyKwh) : null);
  } else if (e.kind === 'remote') {
    // the car warming or cooling itself before a drive: how long, what it took, and the weather that called for it
    if (e.place) detail.push(e.place);
    if (e.tempC != null) detail.push(`${temp(e.tempC)} outside`);
    if (e.restarts) detail.push(`started again ${e.restarts === 1 ? 'once' : `${e.restarts} times`}`);
  } else if (e.kind === 'charging') {
    detail.push(power(e, secs));
    if (extra.stopSec > secs + 300) detail.push(`stopped ${span(extra.stopSec)} in all`);
    // what a kWh cost: from a receipt, or worked out from a price
    const kwh = e.wallKwh || e.addedKwh;
    if (e.cost != null && kwh > 0.5) detail.push(`${money(cur, e.cost / kwh)}/kWh ${e.costActual ? '(receipt)' : '(estimate)'}`);
  }
  const shown = detail.filter(Boolean);
  const used = e.kind === 'remote' && e.energyKwh > 0 ? num(e.energyKwh, 1, ' kWh') : '';
  const right = e.kind === 'remote' ? used : e.kind === 'charging' ? (e.cost != null ? money(cur, e.cost) : e.addedKwh != null ? num(e.addedKwh, 1, ' kWh') : '') : e.kind === 'driving' ? dist(e.distanceKm, 1) : '';
  const sub = e.kind === 'charging' && e.cost != null && e.addedKwh != null ? `${num(e.addedKwh, 1, ' kWh')}${e.gas ? ` · gas ${money(cur, e.gas.cost)}` : ''}` : '';
  const lv = levels(e);
  const body = [
    h('div', { class: 'hrow1' }, h('span', { class: `hico ${e.kind}${stopNumber != null ? ' num' : ''}`, 'aria-hidden': 'true' }, stopNumber != null ? stopNumber : e.kind === 'remote' ? powerIcon() : GLYPH[e.kind]),
      h('b', {}, `${clock(e.start, tz)} – ${e.end == null ? 'Now' : clock(e.end, tz)}`), h('span', { class: 'muted grow', title: span(secs) }, short(secs)),
      lv ? h('span', { class: 'hdelta' }, lv) : null),
    levelBar(e),
    h('div', { class: 'hrow2' }, h('span', { class: 'hplace' }, h('span', {}, place || (e.kind === 'parked' ? 'Parked' : 'Drive')), e.kind === 'parked' && place ? h('div', { class: 'muted' }, 'Parked') : null,
      shown.length ? h('div', { class: 'muted' }, ...shown.flatMap((x, i) => (i ? [' · ', x] : [x]))) : null), h('span', { class: 'hright' }, right, sub ? h('div', { class: 'muted' }, sub) : null))];
  const href = e.kind === 'driving' ? `#/trip/${e.id}` : e.kind === 'charging' ? '#/charging' : null;
  return h(href ? 'a' : 'div', { class: `hcard ${e.kind}`, ...(href ? { href } : {}) }, ...body.filter(Boolean));
}
