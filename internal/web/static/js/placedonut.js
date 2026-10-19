import { h, num } from './ui.js';

const NAMES = { home: 'Home', work: 'Work', public: 'Public' };
const COLORS = { home: 'var(--accent)', work: '#f5a524', public: '#b46cf0' };
const NS = 'http://www.w3.org/2000/svg';
const svg = (tag, attrs) => { const e = document.createElementNS(NS, tag); for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v); return e; };
const money = (cur, v) => `${cur}${v.toFixed(2)}`;

// Charging split into Home, Work and Public: a ring you can read at a glance, and a tile for each with its energy and cost.
export function placeDonut(kinds, cur) {
  const total = kinds.reduce((a, k) => a + k.kwh, 0);
  if (!(total > 0)) return null;
  const R = 52, C = 2 * Math.PI * R;
  const ring = svg('svg', { viewBox: '0 0 140 140', class: 'donut', role: 'img', 'aria-label': kinds.map((k) => `${NAMES[k.kind]} ${Math.round((k.kwh / total) * 100)}%`).join(', ') });
  ring.append(svg('circle', { cx: 70, cy: 70, r: R, fill: 'none', 'stroke-width': 16, class: 'dtrk' }));
  let used = 0;
  for (const k of kinds) {
    const len = (k.kwh / total) * C;
    ring.append(svg('circle', { cx: 70, cy: 70, r: R, fill: 'none', 'stroke-width': 16, stroke: COLORS[k.kind], 'stroke-dasharray': `${Math.max(0, len - 2)} ${C}`, 'stroke-dashoffset': -used, transform: 'rotate(-90 70 70)' }));
    used += len;
  }
  const mid = h('div', { class: 'dmid' }, h('b', {}, num(total, 0, ' kWh')), h('span', { class: 'muted' }, 'charged'));
  const priced = kinds.some((k) => k.cost > 0);
  return h('section', { class: 'card' }, h('h2', {}, 'Where you charge'),
    h('div', { class: 'dwrap' }, h('div', { class: 'dring' }, ring, mid)),
    h('div', { class: 'dtiles' }, ...kinds.map((k) => h('div', { class: 'dtile', style: `border-bottom-color:${COLORS[k.kind]}` },
      h('span', { class: 'muted' }, NAMES[k.kind]), h('b', {}, priced ? money(cur, k.cost) : num(k.kwh, 0, ' kWh')),
      h('span', { class: 'muted' }, `${priced ? `${num(k.kwh, 0, ' kWh')} · ` : ''}${Math.round((k.kwh / total) * 100)}% · ${k.sessions} session${k.sessions === 1 ? '' : 's'}`)))),
    h('p', { class: 'muted' }, 'A place named Work or Office (More, Places) counts as work. Everything else away from home is public.'));
}
