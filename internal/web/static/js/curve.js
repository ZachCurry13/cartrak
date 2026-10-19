import { h, num } from './ui.js';
import { api } from './api.js';

const NS = 'http://www.w3.org/2000/svg';
const mk = (tag, attrs) => { const e = document.createElementNS(NS, tag); for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v); return e; };

// Power against battery level for one session, drawn as a plain SVG line (no chart library).
function draw(c) {
  const pts = c.points.filter((p) => p.kw > 0.2);
  if (pts.length < 2) return h('p', { class: 'muted' }, 'Not enough power readings for a curve in this session.');
  const w = 300, ht = 130, l = 28, b = 18, t = 6, r = 6;
  const top = Math.max(10, Math.ceil(c.peakKw / 10) * 10);
  const x = (soc) => l + (soc / 100) * (w - l - r);
  const y = (kw) => ht - b - (kw / top) * (ht - b - t);
  const svg = mk('svg', { viewBox: `0 0 ${w} ${ht}`, class: 'curve', role: 'img', 'aria-label': 'Charging power against battery level' });
  for (const kw of [0, top / 2, top]) {
    svg.append(mk('line', { x1: l, x2: w - r, y1: y(kw), y2: y(kw), stroke: 'currentColor', 'stroke-opacity': '.15' }));
    const tx = mk('text', { x: l - 4, y: y(kw) + 3, 'text-anchor': 'end', 'font-size': '9', fill: 'currentColor' });
    tx.textContent = Math.round(kw);
    svg.append(tx);
  }
  for (const soc of [0, 25, 50, 75, 100]) {
    const tx = mk('text', { x: x(soc), y: ht - 4, 'text-anchor': 'middle', 'font-size': '9', fill: 'currentColor' });
    tx.textContent = `${soc}%`;
    svg.append(tx);
  }
  svg.append(mk('polyline', { points: pts.map((p) => `${x(p.soc).toFixed(1)},${y(p.kw).toFixed(1)}`).join(' '), fill: 'none', stroke: 'var(--accent)', 'stroke-width': '2.5', 'stroke-linejoin': 'round' }));
  return svg;
}

// "Charging curve" under a session: opens to the chart and the numbers worth comparing between chargers.
export function curveBox(id) {
  const body = h('div', { class: 'muted' }, 'Loading…');
  const box = h('details', { class: 'curvebox' }, h('summary', { class: 'muted' }, 'Charging curve'), body);
  let loaded = false;
  box.addEventListener('toggle', async () => {
    if (!box.open || loaded) return;
    loaded = true;
    try {
      const c = await api(`/api/charges/${id}/curve`);
      const bits = [c.peakKw > 0 ? `Peak ${num(c.peakKw, 0, ' kW')}` : null, c.avgKw != null ? `average ${num(c.avgKw, 0, ' kW')}` : null,
        c.minutes10to80 != null ? `10 to 80% in ${Math.round(c.minutes10to80)} min` : null, c.at80Kw != null ? `${num(c.at80Kw, 0, ' kW')} at 80%` : null,
        c.addedKwh != null ? `${num(c.addedKwh, 1, ' kWh')} added` : null].filter(Boolean);
      body.replaceChildren(draw(c), h('p', {}, bits.join(' · ')));
    } catch (e) { body.textContent = e.message; }
  });
  return box;
}

// How the places you charge compare: sessions that reached 40 kW or more, by named place.
export function chargersCompared(sessions) {
  const by = new Map();
  for (const c of sessions) {
    if (c.open || !c.endTs || c.peakKw < 40 || c.addedKwh == null) continue;
    const hours = (c.endTs - c.startTs) / 3600;
    if (hours < 5 / 60) continue;
    const k = c.place || 'Unnamed place';
    const g = by.get(k) || { n: 0, peak: 0, kw: 0 };
    g.n++; g.peak += c.peakKw; g.kw += c.addedKwh / hours;
    by.set(k, g);
  }
  if (!by.size) return null;
  const rows = [...by].sort((a, b) => b[1].kw / b[1].n - a[1].kw / a[1].n);
  return h('section', { class: 'card' }, h('h2', {}, 'Fast chargers compared'),
    h('p', { class: 'muted' }, 'Sessions that reached 40 kW or more. Name a place under More, Places to see it here by name.'),
    ...rows.map(([name, g]) => h('div', { class: 'row' }, h('span', { class: 'grow' }, name, h('span', { class: 'muted' }, ` · ${g.n} session${g.n === 1 ? '' : 's'}`)),
      h('span', {}, h('b', {}, num(g.kw / g.n, 0, ' kW')), h('div', { class: 'muted' }, `avg · peak ${num(g.peak / g.n, 0)}`)))));
}
