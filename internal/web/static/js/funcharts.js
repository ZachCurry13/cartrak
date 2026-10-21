import { h, num } from './ui.js';
import { valueText } from './funwords.js';

// The Insights pictures: where every kWh went, a typical day as a clock, and achievement badges with progress rings.

const NS = 'http://www.w3.org/2000/svg';
const S = (tag, a = {}, t) => { const e = document.createElementNS(NS, tag); for (const [k, v] of Object.entries(a)) e.setAttribute(k, v); if (t != null) e.textContent = t; return e; };

// Heights for the bars of one column, in proportion to the values but never too thin for their two-line label: the small ones
// get the minimum and the rest share what is left.
function fitHeights(values, room, least = 28) {
  const total = values.reduce((a, v) => a + v, 0) || 1;
  let h = values.map((v) => (v / total) * room);
  const small = h.map((x) => x < least);
  const taken = small.filter(Boolean).length * least;
  const bigTotal = values.reduce((a, v, i) => a + (small[i] ? 0 : v), 0) || 1;
  if (taken < room) h = values.map((v, i) => (small[i] ? least : (v / bigTotal) * (room - taken)));
  return h;
}

// Where the energy went: home and public charging on the left, into the battery, out to city and highway driving, the car's own
// use, losses and what is still in the battery. Bands are as thick as the energy they carry.
export function flowChart(f) {
  const ins = [['Home', f.homeKwh, '#22c55e'], ['Public', f.publicKwh, '#3b82f6']].filter((x) => x[1] > 0.5);
  const outs = [['Lost as heat', f.lostKwh, '#94a3b8'], ['City driving', f.cityKwh, '#f59e0b'], ['Highway', f.highwayKwh, '#f97316'], ['Climate, parked, the rest', f.otherKwh, '#a78bfa'],
    ...(f.storedKwh > 0.5 ? [['Still in the battery', f.storedKwh, '#14b8a6']] : [])].filter((x) => x[1] > 0.5);
  const total = ins.reduce((a, x) => a + x[1], 0);
  if (total < 1) return h('p', { class: 'muted' }, 'No charging in this stretch yet.');
  const outTotal = outs.reduce((a, x) => a + x[1], 0) || 1;
  const W = 350, H = 250, top = 8, gap = 8, x0 = 10, xm = 150, x1 = 300;
  const usable = (n) => H - top * 2 - gap * (n - 1);
  const svg = S('svg', { viewBox: `0 0 ${W} ${H}`, class: 'chart', role: 'img', 'aria-label': 'Where the energy went' });
  const band = (ax, a0, a1, bx, b0, b1, col) => { const m = (ax + bx) / 2; svg.append(S('path', { d: `M${ax},${a0} C${m},${a0} ${m},${b0} ${bx},${b0} L${bx},${b1} C${m},${b1} ${m},${a1} ${ax},${a1} Z`, fill: col, 'fill-opacity': '.45' })); };
  const label = (x, y, anchor, name, kwh) => { const t = S('text', { x, y, 'text-anchor': anchor, 'font-size': '11', fill: 'currentColor' }); t.append(S('tspan', { 'font-weight': '700' }, name), S('tspan', { x, dy: 13, 'fill-opacity': '.7' }, num(kwh, 0, ' kWh'))); svg.append(t); };
  // the middle bar is the energy in, scaled to the left column; the right column has its own scale
  const hIn = fitHeights(ins.map((x) => x[1]), usable(ins.length)), hOut = fitHeights(outs.map((x) => x[1]), usable(outs.length));
  let yl = top, ym = top + ((ins.length - 1) * gap) / 2;
  for (const [i, [name, v, col]] of ins.entries()) {
    const hgt = hIn[i];
    band(x0 + 8, yl, yl + hgt, xm, ym, ym + hgt, col);
    svg.append(S('rect', { x: x0, y: yl, width: 8, height: hgt, rx: 2, fill: col }));
    label(x0 + 14, yl + hgt / 2 - 2, 'start', name, v);
    yl += hgt + gap; ym += hgt;
  }
  const midTop = top + ((ins.length - 1) * gap) / 2, midH = usable(ins.length);
  svg.append(S('rect', { x: xm, y: midTop, width: 8, height: midH, rx: 2, fill: 'var(--muted)' }));
  let yr = top, ys = midTop;
  for (const [i, [name, v, col]] of outs.entries()) {
    const hgt = hOut[i], share = (v / outTotal) * midH;
    band(xm + 8, ys, ys + share, x1, yr, yr + hgt, col);
    svg.append(S('rect', { x: x1, y: yr, width: 8, height: Math.max(2, hgt), rx: 2, fill: col }));
    label(x1 - 6, yr + hgt / 2 - 2, 'end', name, v);
    yr += hgt + gap; ys += share;
  }
  return svg;
}

// A typical day as a clock: driving on the outer ring, charging on the inner one, darker where it happens on more days.
export function dayRing(r) {
  const svg = S('svg', { viewBox: '-14 -14 268 268', class: 'dayring', role: 'img', 'aria-label': 'A typical day' });
  const arc = (q1, q2, rad, w, col) => {
    const a1 = (q1 / 96) * 2 * Math.PI - Math.PI / 2, a2 = (q2 / 96) * 2 * Math.PI - Math.PI / 2;
    svg.append(S('path', { d: `M${120 + rad * Math.cos(a1)},${120 + rad * Math.sin(a1)} A${rad},${rad} 0 ${q2 - q1 > 48 ? 1 : 0} 1 ${120 + rad * Math.cos(a2)},${120 + rad * Math.sin(a2)}`, fill: 'none', stroke: col, 'stroke-width': w }));
  };
  for (const [rad, w] of [[92, 26], [62, 18]]) { arc(0, 48, rad, w, 'var(--card2)'); arc(48, 96, rad, w, 'var(--card2)'); }
  const top = Math.max(...r.driving, ...r.charging, 1e-9);
  const paint = (arr, rad, w, col) => arr.forEach((v, q) => { if (v > top * 0.04) arc(q, q + 1.05, rad, w, `color-mix(in srgb,${col} ${Math.round(25 + (v / top) * 75)}%,var(--card2))`); });
  paint(r.driving, 92, 26, '#f59e0b');
  paint(r.charging, 62, 18, '#22c55e');
  ['12a', '3a', '6a', '9a', '12p', '3p', '6p', '9p'].forEach((t, i) => { const a = (i / 8) * 2 * Math.PI - Math.PI / 2; svg.append(S('text', { x: 120 + 122 * Math.cos(a), y: 124 + 122 * Math.sin(a), 'text-anchor': 'middle', 'font-size': '10', fill: 'currentColor', 'fill-opacity': '.6' }, t)); });
  svg.append(S('text', { x: 120, y: 118, 'text-anchor': 'middle', 'font-size': '28', 'font-weight': '800', fill: 'currentColor' }, `${Math.round(r.parked * 100)}%`),
    S('text', { x: 120, y: 136, 'text-anchor': 'middle', 'font-size': '11', fill: 'currentColor', 'fill-opacity': '.7' }, 'of the time parked'));
  return svg;
}

export const ringKey = () => h('div', { class: 'ckey', style: 'justify-content:center' }, h('span', {}, h('i', { style: 'background:#f59e0b' }), 'Driving'), h('span', {}, h('i', { style: 'background:#22c55e' }), 'Charging'));

const COLORS = ['#22c55e', '#f59e0b', '#a78bfa', '#3b82f6', '#f97316', '#14b8a6', '#eab308', '#38bdf8'];

// One badge: its emoji in a ring that fills with progress, full and in color once earned.
export function badge(b, i, cur) {
  const p = b.progress ?? (b.earned ? 1 : 0), col = COLORS[i % COLORS.length];
  const ring = S('svg', { viewBox: '0 0 78 78', width: 78, height: 78, 'aria-hidden': 'true' });
  const c = 2 * Math.PI * 33;
  ring.append(S('circle', { cx: 39, cy: 39, r: 33, fill: 'var(--card2)', stroke: 'var(--line)', 'stroke-width': 6 }));
  if (p > 0) ring.append(S('circle', { cx: 39, cy: 39, r: 33, fill: 'none', stroke: col, 'stroke-width': 6, 'stroke-linecap': 'round', 'stroke-dasharray': `${Math.max(0.5, c * p)} ${c}`, transform: 'rotate(-90 39 39)' }));
  const sub = b.earned ? new Date(b.earned * 1000).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' })
    : b.kind === 'tempC' ? b.about : `${valueText(b.kind, b.value, cur)} of ${valueText(b.kind, b.goal, cur)}`;
  return h('div', { class: `badge${b.earned ? '' : ' locked'}`, title: b.about }, h('div', { class: 'bring' }, ring, h('span', {}, b.emoji)), h('b', {}, b.name), h('span', { class: 'muted' }, sub));
}

// Earned badges first (newest first), then the ones closest to done.
export function badgeGrid(list, cur, limit = 0) {
  const sorted = [...list].sort((a, b) => (b.earned > 0) - (a.earned > 0) || (b.earned - a.earned) || (b.progress - a.progress));
  return h('div', { class: 'badges' }, ...(limit ? sorted.slice(0, limit) : sorted).map((b, i) => badge(b, list.indexOf(b), cur)));
}
