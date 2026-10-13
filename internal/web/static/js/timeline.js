import { h } from './ui.js';

const NS = 'http://www.w3.org/2000/svg';
const el = (tag, attrs) => {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v);
  return e;
};

// A day at a glance: the battery percentage as a line, with bands for when the car was driving
// (blue) or charging (green). Tap or click anywhere to jump to that moment. No chart library.
export function timelineChart(points, from, to, markTs, onPick) {
  const w = 400;
  const ht = 130;
  const x = (ts) => ((ts - from) / (to - from)) * w;
  const y = (soc) => ht - 14 - (soc / 100) * (ht - 28);
  const svg = el('svg', { viewBox: `0 0 ${w} ${ht}`, class: 'tl', role: 'img', 'aria-label': 'Battery through the day' });
  for (let i = 1; i < points.length; i++) {
    const p = points[i - 1];
    if (!p.driving && !p.charging) continue;
    svg.append(el('rect', { x: x(p.ts), y: 0, width: Math.max(2, x(points[i].ts) - x(p.ts)), height: ht - 14, fill: p.charging ? '#059669' : '#1b5fc4', opacity: 0.22 }));
  }
  const line = points.filter((p) => p.soc != null);
  if (line.length > 1) svg.append(el('polyline', { points: line.map((p) => `${x(p.ts).toFixed(1)},${y(p.soc).toFixed(1)}`).join(' '), fill: 'none', stroke: 'currentColor', 'stroke-width': 2.5, 'stroke-linejoin': 'round' }));
  for (const hr of [0, 6, 12, 18, 24]) {
    const t = from + (hr / 24) * (to - from);
    svg.append(el('line', { x1: x(t), x2: x(t), y1: ht - 14, y2: ht - 10, stroke: 'currentColor', opacity: 0.4 }));
    const label = el('text', { x: Math.min(w - 14, Math.max(8, x(t))), y: ht - 1, 'font-size': 10, 'text-anchor': 'middle', fill: 'currentColor', opacity: 0.7 });
    label.textContent = hr === 24 ? '' : `${hr}:00`;
    svg.append(label);
  }
  if (markTs >= from && markTs <= to) svg.append(el('line', { x1: x(markTs), x2: x(markTs), y1: 0, y2: ht - 14, stroke: '#dc2626', 'stroke-width': 2 }));
  svg.addEventListener('click', (e) => {
    const r = svg.getBoundingClientRect();
    onPick(Math.round(from + ((e.clientX - r.left) / r.width) * (to - from)));
  });
  svg.style.cursor = 'pointer';
  return h('div', { class: 'tlwrap' }, svg, h('div', { class: 'muted tllegend' },
    h('span', { class: 'chip work' }, 'battery %'), ' ', h('span', { class: 'chip ok' }, 'charging'), ' ', h('span', { class: 'chip personal' }, 'driving'), '  Tap the chart to jump to a moment.'));
}
