import { h } from './ui.js';

const NS = 'http://www.w3.org/2000/svg';
const mk = (tag, attrs, text) => {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs || {})) e.setAttribute(k, v);
  if (text != null) e.textContent = text;
  return e;
};

// A line chart you can read on a phone: big labels, axis titles and units, the highest point marked, and a readout
// that follows your finger. Touch and drag along the chart (or move the mouse over it) to see the exact value there.
//   points   [{ x, y }] sorted by x
//   xFmt/yFmt  how to write an x or y value (for the axes and the readout)
//   xTicks   optional [x, ...] where to label the x axis
//   summary  the line shown before anything is touched
//   xTitle/yTitle  words under and beside the axes
export function lineChart({ points, xFmt, yFmt, xTicks, summary, xTitle, yTitle, yMin = 0, height = 190, color = 'var(--accent)' }) {
  if (!points || points.length < 2) return h('p', { class: 'muted' }, 'Not enough readings for a chart yet.');
  const W = 320, H = height, L = 40, R = 10, T = 10, B = 34;
  const xs = points.map((p) => p.x), ys = points.map((p) => p.y);
  const x0 = Math.min(...xs), x1 = Math.max(...xs);
  const yMax = Math.max(...ys), yLo = Math.min(yMin, ...ys);
  const span = yMax - yLo || 1;
  const top = yLo + Math.ceil((span * 1.08) / niceStep(span)) * niceStep(span);
  const sx = (x) => L + ((x - x0) / (x1 - x0 || 1)) * (W - L - R);
  const sy = (y) => T + (1 - (y - yLo) / (top - yLo || 1)) * (H - T - B);

  const readout = h('div', { class: 'chartread', role: 'status' }, summary || '');
  const svg = mk('svg', { viewBox: `0 0 ${W} ${H}`, class: 'chart', role: 'img', 'aria-label': yTitle ? `${yTitle} chart` : 'Chart' });
  svg.style.touchAction = 'pan-y';

  const step = niceStep(top - yLo);
  for (let v = yLo; v <= top + 1e-9; v += step) {
    svg.append(mk('line', { x1: L, x2: W - R, y1: sy(v), y2: sy(v), stroke: 'currentColor', 'stroke-opacity': '.14' }));
    svg.append(mk('text', { x: L - 5, y: sy(v) + 4, 'text-anchor': 'end', 'font-size': '11', fill: 'currentColor', 'fill-opacity': '.75' }, yFmt(v, true)));
  }
  const ticks = xTicks || [x0, x0 + (x1 - x0) / 2, x1];
  for (const t of ticks) {
    svg.append(mk('text', { x: sx(t), y: H - 16, 'text-anchor': t <= x0 ? 'start' : t >= x1 ? 'end' : 'middle', 'font-size': '11', fill: 'currentColor', 'fill-opacity': '.75' }, xFmt(t, true)));
  }
  if (xTitle) svg.append(mk('text', { x: (L + W - R) / 2, y: H - 2, 'text-anchor': 'middle', 'font-size': '10', fill: 'currentColor', 'fill-opacity': '.6' }, xTitle));
  if (yTitle) svg.append(mk('text', { x: 4, y: 9, 'font-size': '10', fill: 'currentColor', 'fill-opacity': '.6' }, yTitle));

  const line = points.map((p) => `${sx(p.x).toFixed(1)},${sy(p.y).toFixed(1)}`).join(' ');
  svg.append(mk('polygon', { points: `${sx(x0)},${sy(yLo)} ${line} ${sx(x1)},${sy(yLo)}`, fill: color, 'fill-opacity': '.12' }));
  svg.append(mk('polyline', { points: line, fill: 'none', stroke: color, 'stroke-width': '2.5', 'stroke-linejoin': 'round', 'stroke-linecap': 'round' }));

  const peak = points.reduce((a, p) => (p.y > a.y ? p : a), points[0]);
  svg.append(mk('circle', { cx: sx(peak.x), cy: sy(peak.y), r: 4.5, fill: color }));
  const pl = mk('text', { x: Math.min(Math.max(sx(peak.x), 60), W - 60), y: Math.max(sy(peak.y) - 8, 12), 'text-anchor': 'middle', 'font-size': '11', 'font-weight': '700', fill: 'currentColor' }, `high ${yFmt(peak.y)}`);
  svg.append(pl);

  const cross = mk('line', { y1: T, y2: H - B, stroke: 'currentColor', 'stroke-opacity': '.5', 'stroke-dasharray': '3 3', visibility: 'hidden' });
  const dot = mk('circle', { r: 6, fill: color, stroke: 'var(--card)', 'stroke-width': '2', visibility: 'hidden' });
  svg.append(cross, dot);

  const at = (clientX) => {
    const r = svg.getBoundingClientRect();
    const x = x0 + (((clientX - r.left) / r.width) * W - L) / (W - L - R) * (x1 - x0);
    let best = points[0];
    for (const p of points) if (Math.abs(p.x - x) < Math.abs(best.x - x)) best = p;
    return best;
  };
  const show = (clientX) => {
    const p = at(clientX);
    cross.setAttribute('x1', sx(p.x)); cross.setAttribute('x2', sx(p.x)); cross.setAttribute('visibility', 'visible');
    dot.setAttribute('cx', sx(p.x)); dot.setAttribute('cy', sy(p.y)); dot.setAttribute('visibility', 'visible');
    readout.textContent = `${xFmt(p.x)} · ${yFmt(p.y)}`;
    pl.setAttribute('visibility', 'hidden');
  };
  const reset = () => { cross.setAttribute('visibility', 'hidden'); dot.setAttribute('visibility', 'hidden'); readout.textContent = summary || ''; pl.setAttribute('visibility', 'visible'); };
  svg.addEventListener('pointerdown', (e) => { svg.setPointerCapture && svg.setPointerCapture(e.pointerId); show(e.clientX); });
  svg.addEventListener('pointermove', (e) => { if (e.buttons || e.pointerType === 'mouse' || e.pointerType === 'touch') show(e.clientX); });
  svg.addEventListener('pointerleave', reset);
  svg.addEventListener('pointerup', (e) => { if (e.pointerType !== 'mouse') setTimeout(reset, 2500); });
  svg.addEventListener('keydown', () => {});
  return h('div', { class: 'chartbox' }, readout, svg, h('div', { class: 'muted chartnote' }, 'Touch and drag along the chart to read a value.'));
}

// A "nice" step for gridlines: 1, 2, 5 times a power of ten, giving about four lines.
function niceStep(span) {
  const raw = span / 4;
  const p = 10 ** Math.floor(Math.log10(raw || 1));
  const f = raw / p;
  return (f < 1.5 ? 1 : f < 3.5 ? 2 : f < 7.5 ? 5 : 10) * p;
}

// Month or day bars where tapping a bar tells you its value in words, under the chart.
//   items [{ label, value, text }]
export function tapBars(items, { height = 120, summary = '' } = {}) {
  if (!items.length) return null;
  const top = Math.max(...items.map((i) => i.value), 1e-9);
  const readout = h('div', { class: 'chartread', role: 'status' }, summary);
  const cols = items.map((it) => {
    const b = h('button', { class: 'ybar tapbar', 'aria-label': `${it.label}: ${it.text}`, type: 'button', onclick: () => {
      for (const c of cols) c.classList.remove('on');
      b.classList.add('on');
      readout.textContent = `${it.label}: ${it.text}`;
    } }, h('div', { class: 'yfill', style: `height:${Math.max(3, Math.round((it.value / top) * 100))}%` }), h('span', { class: 'muted' }, it.label));
    return b;
  });
  return h('div', { class: 'chartbox' }, readout, h('div', { class: 'ybars', style: `height:${height}px` }, ...cols), h('div', { class: 'muted chartnote' }, 'Tap a bar to read its value.'));
}
