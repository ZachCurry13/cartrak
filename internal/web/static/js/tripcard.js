import { h, catChip, chip, dist, when, duration, pct } from './ui.js';
import { effSpan } from './energy.js';
import { catColor } from './categories.js';

const NS = 'http://www.w3.org/2000/svg';
const svgEl = (tag, attrs) => { const e = document.createElementNS(NS, tag); for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v); return e; };

// A small picture of where the drive went: the route in the category's colour, a green dot where it began and a red one where it ended.
function thumb(points, color) {
  const box = h('div', { class: 'tthumb', 'aria-hidden': 'true' });
  if (!points || points.length < 2) { box.textContent = '🚗'; box.classList.add('empty'); return box; }
  const lats = points.map((p) => p[0]), lons = points.map((p) => p[1]);
  const minLat = Math.min(...lats), maxLat = Math.max(...lats), minLon = Math.min(...lons), maxLon = Math.max(...lons);
  const k = Math.cos(((minLat + maxLat) / 2) * Math.PI / 180);
  const w = Math.max((maxLon - minLon) * k, 1e-5), ht = Math.max(maxLat - minLat, 1e-5);
  const sc = 44 / Math.max(w, ht), pad = 8;
  const W = w * sc + pad * 2, H = ht * sc + pad * 2;
  const xy = (p) => [pad + (p[1] - minLon) * k * sc, pad + (maxLat - p[0]) * sc];
  const svg = svgEl('svg', { viewBox: `0 0 ${W.toFixed(1)} ${H.toFixed(1)}`, preserveAspectRatio: 'xMidYMid meet' });
  svg.append(svgEl('polyline', { points: points.map((p) => xy(p).map((n) => n.toFixed(1)).join(',')).join(' '), fill: 'none', stroke: color, 'stroke-width': 2.6, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }));
  const [sx, sy] = xy(points[0]), [ex, ey] = xy(points[points.length - 1]);
  svg.append(svgEl('circle', { cx: sx, cy: sy, r: 3.2, fill: '#16a34a' }), svgEl('circle', { cx: ex, cy: ey, r: 3.2, fill: '#dc2626' }));
  box.append(svg);
  return box;
}

// How much of the battery the drive used: the level it ended at, then the stretch it spent.
function battery(t) {
  if (t.startSoc == null || t.endSoc == null) return null;
  const end = Math.max(0, Math.min(100, t.endSoc)), start = Math.max(end, Math.min(100, t.startSoc));
  return h('div', { class: 'sbar', title: `${pct(t.startSoc)} to ${pct(t.endSoc)}` }, h('i', { class: 'used', style: `left:${end}%;width:${start - end}%` }), h('i', { class: 'left', style: `width:${end}%` }));
}

// One drive in the Trips list: its route, how far and how long, where, how much battery it took and how efficient it was against the rest.
export function tripCard(t, { sel, admin, thumbs, avgKmPerKwh }) {
  const color = catColor(t.category);
  const effText = effSpan(t.distanceKm, t.energyKwh, t.energyEstimated ? '≈ ' : '');
  const kmPerKwh = t.energyKwh > 0 && t.distanceKm > 1 ? t.distanceKm / t.energyKwh : null;
  const grade = kmPerKwh && avgKmPerKwh ? (kmPerKwh >= avgKmPerKwh * 1.08 ? 'good' : kmPerKwh <= avgKmPerKwh * 0.9 ? 'bad' : '') : '';
  const picked = sel.picked.has(t.id);
  const places = t.startPlace || t.endPlace ? `${t.startPlace || '…'} → ${t.endPlace || '…'}` : '';
  const share = admin && !t.open && !sel.state.on
    ? h('button', { class: 'ico sm', type: 'button', 'aria-label': 'Share this trip', title: 'Share this trip', onclick: (e) => { e.preventDefault(); e.stopPropagation(); location.hash = `#/trip/${t.id}/share`; } },
      (() => { const s = svgEl('svg', { viewBox: '0 0 24 24' }); s.append(svgEl('path', { d: 'M12 3v12M8 7l4-4 4 4M5 12v8h14v-8' })); return s; })()) : null;
  return h('a', { class: `tcard${picked ? ' picked' : ''}`, style: `--cc:${color}${sel.state.on && !sel.eligible(t) ? ';opacity:.5' : ''}`, href: sel.state.on ? '#' : `#/trip/${t.id}`,
    onclick: sel.state.on ? (e) => { e.preventDefault(); if (sel.eligible(t)) sel.toggle(t); } : null },
  thumb(thumbs && thumbs.get(t.id), color),
  h('div', { class: 'tmain' },
    h('div', { class: 'tline' }, h('span', { class: 'tbig' }, picked ? '✓ ' : '', t.open ? 'In progress' : dist(t.distanceKm)), h('span', { class: 'muted' }, duration(t.startTs, t.endTs)),
      t.distanceSrc ? chip('work', 'corrected') : null, t.journeyId ? chip('work', 'journey') : null),
    h('div', { class: 'muted' }, when(t.startTs, t.tz)),
    places ? h('div', { class: 'muted tplaces' }, places) : null,
    h('div', { class: 'tline' }, battery(t), kmPerKwh ? h('span', { class: `pill ${grade}` }, effText) : null),
    t.notes ? h('div', { class: 'muted' }, t.notes) : null),
  h('div', { class: 'tside' }, catChip(t), share));
}
