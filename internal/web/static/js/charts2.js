import { h } from './ui.js';

// More charts for the Insights pages, in the same manner as chart.js: a readout above that says in words what was touched,
// theme colours only, and sizes that work on a phone.

const NS = 'http://www.w3.org/2000/svg';
const mk = (tag, attrs, text) => {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs || {})) e.setAttribute(k, v);
  if (text != null) e.textContent = text;
  return e;
};
const box = (readout, body, note) => h('div', { class: 'chartbox' }, readout, body, note ? h('div', { class: 'muted chartnote' }, note) : null);

// Bars made of parts stacked on each other (home and public energy in a month). items [{ label, parts: [value, ...], text }],
// classes name each part's colour (in order), legend names them.
export function stackBars(items, { classes, legend, summary = '', height = 130 }) {
  if (!items.length) return null;
  const top = Math.max(...items.map((i) => i.parts.reduce((a, b) => a + b, 0)), 1e-9);
  const readout = h('div', { class: 'chartread', role: 'status' }, summary);
  const cols = items.map((it) => {
    const b = h('button', { class: 'ybar tapbar', type: 'button', 'aria-label': `${it.label}: ${it.text}`, onclick: () => { cols.forEach((c) => c.classList.remove('on')); b.classList.add('on'); readout.textContent = `${it.label}: ${it.text}`; } },
      h('div', { class: 'ystack' }, ...it.parts.map((v, i) => h('div', { class: `yfill ${classes[i]}`, style: `height:${(v / top) * 100}%` }))), h('span', { class: 'muted' }, it.label));
    return b;
  });
  const key = h('div', { class: 'ckey' }, ...legend.map((t, i) => h('span', {}, h('i', { class: classes[i] }), t)));
  return box(readout, h('div', {}, h('div', { class: 'ybars', style: `height:${height}px` }, ...cols), key), 'Tap a bar to read its value.');
}

// A grid of cells shaded by value: hour of the week, for example. rows and cols are labels (cols may be sparse: '' for no
// label), values[r][c] numbers, text(r, c, v) what the readout says for a cell.
export function heatGrid(rows, cols, values, { text, summary = '', colour = 'var(--ok)' }) {
  const top = Math.max(...values.flat(), 1e-9);
  const readout = h('div', { class: 'chartread', role: 'status' }, summary);
  const grid = h('div', { class: 'hgrid', style: `grid-template-columns:2.4em repeat(${cols.length},minmax(0,1fr))` });
  rows.forEach((rl, r) => {
    grid.append(h('span', { class: 'muted hgl' }, rl));
    cols.forEach((_c, c) => {
      const v = values[r][c];
      const cell = h('button', { type: 'button', class: 'hcell', 'aria-label': text(r, c, v), style: `background:${v > 0 ? `color-mix(in srgb,${colour} ${Math.round(18 + (v / top) * 82)}%,var(--card2))` : 'var(--card2)'}`,
        onclick: () => { grid.querySelectorAll('.hcell.on').forEach((x) => x.classList.remove('on')); cell.classList.add('on'); readout.textContent = text(r, c, v); } });
      grid.append(cell);
    });
  });
  grid.append(h('span'), ...cols.map((c) => h('span', { class: 'muted hgx' }, c)));
  return box(readout, grid, 'Darker means more. Tap a square to read it.');
}

// A calendar of days, one column per week (Sunday at the top), shaded by value. days [{ date: 'YYYY-MM-DD', value, text }].
// It shows the last `weeks` weeks and scrolls sideways inside itself when that is wider than the screen.
export function calendarHeat(days, { weeks = 26, summary = '', colour = 'var(--ok)', today = new Date() }) {
  const byDate = new Map(days.map((d) => [d.date, d]));
  const top = Math.max(...days.map((d) => d.value), 1e-9);
  const readout = h('div', { class: 'chartread', role: 'status' }, summary);
  const end = new Date(today.getFullYear(), today.getMonth(), today.getDate());
  const start = new Date(end); start.setDate(end.getDate() - end.getDay() - (weeks - 1) * 7);
  const grid = h('div', { class: 'cgrid', style: `grid-template-columns:repeat(${weeks},12px)` });
  const iso = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  for (let w = 0; w < weeks; w++) {
    for (let r = 0; r < 7; r++) {
      const d = new Date(start); d.setDate(start.getDate() + w * 7 + r);
      if (d > end) { grid.append(h('span')); continue; }
      const x = byDate.get(iso(d));
      const v = x ? x.value : 0;
      const label = x ? `${d.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })}: ${x.text}` : `${d.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })}: no driving`;
      const cell = h('button', { type: 'button', class: 'ccell', title: label, 'aria-label': label, style: `grid-row:${r + 1};grid-column:${w + 1};background:${v > 0 ? `color-mix(in srgb,${colour} ${Math.round(25 + (v / top) * 75)}%,var(--card2))` : 'var(--card2)'}`,
        onclick: () => { grid.querySelectorAll('.ccell.on').forEach((c) => c.classList.remove('on')); cell.classList.add('on'); readout.textContent = label; } });
      grid.append(cell);
    }
  }
  const wrap = h('div', { class: 'cwrap' }, grid);
  requestAnimationFrame(() => { wrap.scrollLeft = wrap.scrollWidth; });
  return box(readout, wrap, 'Each square is a day; darker means farther. Tap one to read it.');
}

// Points against two axes, with an optional straight trend line. points [{ x, y, size, text, href }]; line [a, b] for y = a + b·x.
export function scatter(points, { xFmt, yFmt, xTitle, yTitle, line, summary = '', colour = 'var(--ok)' }) {
  if (points.length < 3) return h('p', { class: 'muted' }, 'Not enough trips for this chart yet.');
  const W = 320, H = 220, L = 42, R = 10, T = 22, B = 34;
  const xs = points.map((p) => p.x), ys = points.map((p) => p.y);
  const x0 = Math.min(...xs), x1 = Math.max(...xs), y0 = Math.min(...ys) * 0.95, y1 = Math.max(...ys) * 1.05;
  const sx = (x) => L + ((x - x0) / (x1 - x0 || 1)) * (W - L - R), sy = (y) => T + (1 - (y - y0) / (y1 - y0 || 1)) * (H - T - B);
  const svg = mk('svg', { viewBox: `0 0 ${W} ${H}`, class: 'chart', role: 'img', 'aria-label': `${yTitle} by ${xTitle}` });
  const readout = h('div', { class: 'chartread', role: 'status' }, summary);
  for (let i = 0; i <= 3; i++) {
    const v = y0 + ((y1 - y0) * i) / 3;
    svg.append(mk('line', { x1: L, x2: W - R, y1: sy(v), y2: sy(v), stroke: 'currentColor', 'stroke-opacity': '.12' }), mk('text', { x: L - 5, y: sy(v) + 4, 'text-anchor': 'end', 'font-size': '10', fill: 'currentColor', 'fill-opacity': '.75' }, yFmt(v, true)));
  }
  [x0, (x0 + x1) / 2, x1].forEach((t, i) => svg.append(mk('text', { x: sx(t), y: H - 16, 'text-anchor': ['start', 'middle', 'end'][i], 'font-size': '10', fill: 'currentColor', 'fill-opacity': '.75' }, xFmt(t, true))));
  svg.append(mk('text', { x: (L + W - R) / 2, y: H - 2, 'text-anchor': 'middle', 'font-size': '10', fill: 'currentColor', 'fill-opacity': '.6' }, xTitle), mk('text', { x: 4, y: 9, 'font-size': '10', fill: 'currentColor', 'fill-opacity': '.6' }, yTitle));
  const maxSize = Math.max(...points.map((p) => p.size || 1));
  const dots = points.map((p) => {
    const c = mk('circle', { cx: sx(p.x), cy: sy(p.y), r: 2.5 + 3.5 * Math.sqrt((p.size || 1) / maxSize), fill: colour, 'fill-opacity': '.55' });
    svg.append(c);
    return c;
  });
  if (line) svg.append(mk('line', { x1: sx(x0), y1: sy(line[0] + line[1] * x0), x2: sx(x1), y2: sy(line[0] + line[1] * x1), stroke: 'var(--warn)', 'stroke-width': '2.5', 'stroke-dasharray': '6 4' }));
  let picked = null;
  svg.style.touchAction = 'pan-y';
  svg.addEventListener('pointerdown', (e) => {
    const r = svg.getBoundingClientRect(), px = ((e.clientX - r.left) / r.width) * W, py = ((e.clientY - r.top) / r.height) * H;
    let best = 0, bd = Infinity;
    points.forEach((p, i) => { const d = (sx(p.x) - px) ** 2 + (sy(p.y) - py) ** 2; if (d < bd) { bd = d; best = i; } });
    if (picked != null) dots[picked].setAttribute('fill-opacity', '.55');
    picked = best;
    dots[best].setAttribute('fill-opacity', '1');
    const p = points[best];
    readout.replaceChildren(p.text, p.href ? h('span', {}, ' · ', h('a', { href: p.href }, 'open')) : '');
  });
  return box(readout, svg, `Each dot is a trip (bigger is longer)${line ? '; the dashed line is the trend' : ''}. Tap a dot to read it.`);
}
