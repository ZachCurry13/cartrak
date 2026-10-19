import { h, num } from './ui.js';
import { api } from './api.js';
import { lineChart } from './chart.js';

// Power against battery level for one session: touch and drag along it to read the power at any level.
function draw(c) {
  const pts = c.points.filter((p) => p.kw > 0.2).map((p) => ({ x: p.soc, y: p.kw })).sort((a, b) => a.x - b.x);
  if (pts.length < 2) return h('p', { class: 'muted' }, 'Not enough power readings for a curve in this session.');
  const lo = Math.floor(pts[0].x / 20) * 20;
  const ticks = [];
  for (let t = lo; t <= 100; t += 20) if (t >= pts[0].x - 20 && t <= pts[pts.length - 1].x + 20) ticks.push(t);
  return lineChart({
    points: pts,
    xFmt: (x) => `${Math.round(x)}%`,
    yFmt: (v, axis) => (axis || v >= 100 ? `${Math.round(v)}` : `${v.toFixed(1)}`) + (axis ? '' : ' kW'),
    xTicks: ticks.length >= 2 ? ticks : undefined,
    xTitle: 'Battery level',
    yTitle: 'Power (kW)',
    summary: `Peak ${Math.round(c.peakKw)} kW${c.at80Kw != null ? ` · ${Math.round(c.at80Kw)} kW at 80%` : ''}`,
  });
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

// Fastest and average charging speed for slow (Level 2) and fast (Level 3) charging, from the finished sessions.
// A session that peaked at 30 kW or more counts as fast; Level 2 tops out around 19 kW.
export function recordsCard(sessions) {
  const classes = { l2: { name: 'Level 2 (home or public, slow)', n: 0, best: null, sum: 0, k: 0 }, l3: { name: 'Level 3 (fast chargers)', n: 0, best: null, sum: 0, k: 0 } };
  for (const c of sessions) {
    if (c.open || !c.endTs || !(c.peakKw > 0.5)) continue;
    const g = c.peakKw >= 30 ? classes.l3 : classes.l2;
    g.n++;
    if (!g.best || c.peakKw > g.best.peakKw) g.best = c;
    const hours = (c.endTs - c.startTs) / 3600;
    if (c.addedKwh != null && hours >= 10 / 60) { g.sum += c.addedKwh / hours; g.k++; }
  }
  const rows = Object.values(classes).filter((g) => g.n > 0);
  if (!rows.length) return null;
  const day = (ts) => new Date(ts * 1000).toLocaleDateString([], { month: 'short', day: 'numeric' });
  return h('section', { class: 'card' }, h('h2', {}, 'Charging speed'),
    ...rows.map((g) => h('div', { class: 'row' },
      h('div', { class: 'grow' }, h('b', {}, g.name), h('div', { class: 'muted' }, `${g.n} session${g.n === 1 ? '' : 's'} · fastest ${day(g.best.startTs)}${g.best.place ? ` at ${g.best.place}` : ''}`)),
      h('div', { style: 'text-align:right' }, h('b', {}, num(g.best.peakKw, g.best.peakKw >= 100 ? 0 : 1, ' kW')), h('div', { class: 'muted' }, g.k ? `avg ${num(g.sum / g.k, g.sum / g.k >= 100 ? 0 : 1, ' kW')}` : 'avg —')))),
    h('p', { class: 'muted' }, 'Fastest is the highest power seen. Average is the energy added over the time plugged in, so it includes the slow start and the taper.'));
}
