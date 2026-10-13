import { h, stat, catChip, dist, speed, when, duration, pct, num } from '../ui.js';
import { api } from '../api.js';
import { isSaver } from '../mode.js';
import { createMap, mapConfig } from '../map.js';
import { efficiency, kwhText } from '../energy.js';
import { shareCard } from '../sharecard.js';
import { placeCard } from '../placecard.js';

const NS = 'http://www.w3.org/2000/svg';

// Route drawn as a plain SVG line: no map tiles, nothing sent to third parties.
function route(samples) {
  const pts = samples.filter((s) => s.lat != null && s.lon != null);
  if (pts.length < 2) return h('p', { class: 'muted' }, 'No GPS route recorded for this trip.');
  const lats = pts.map((p) => p.lat), lons = pts.map((p) => p.lon);
  const minLat = Math.min(...lats), maxLat = Math.max(...lats), minLon = Math.min(...lons), maxLon = Math.max(...lons);
  const k = Math.cos(((minLat + maxLat) / 2) * Math.PI / 180);
  const w = Math.max((maxLon - minLon) * k, 1e-5), ht = Math.max(maxLat - minLat, 1e-5);
  const sc = 280 / Math.max(w, ht), pad = 10;
  const xy = (p) => [pad + (p.lon - minLon) * k * sc, pad + (maxLat - p.lat) * sc];
  const svg = document.createElementNS(NS, 'svg');
  svg.setAttribute('class', 'route');
  svg.setAttribute('viewBox', `0 0 ${w * sc + pad * 2} ${ht * sc + pad * 2}`);
  const line = document.createElementNS(NS, 'polyline');
  line.setAttribute('points', pts.map((p) => xy(p).join(',')).join(' '));
  line.setAttribute('fill', 'none'); line.setAttribute('stroke', 'currentColor'); line.setAttribute('stroke-width', '3'); line.setAttribute('stroke-linejoin', 'round');
  svg.append(line);
  [[pts[0], '#059669'], [pts[pts.length - 1], '#dc2626']].forEach(([p, c]) => {
    const dot = document.createElementNS(NS, 'circle');
    const [cx, cy] = xy(p);
    dot.setAttribute('cx', cx); dot.setAttribute('cy', cy); dot.setAttribute('r', 6); dot.setAttribute('fill', c);
    svg.append(dot);
  });
  return svg;
}

// The phone's GPS track is finer than Ford's readings, so it is drawn when there is one.
function routeFor(d, mc) {
  const phone = (d.phone || []).filter((p) => p.lat != null && p.lon != null);
  const pts = phone.length >= 2 ? phone : (d.samples || []).filter((s) => s.lat != null && s.lon != null);
  const note = phone.length >= 2 ? h('p', { class: 'muted' }, `Drawn from your phone's GPS (${phone.length} points).`) : null;
  if (pts.length < 2) return [h('p', { class: 'muted' }, 'No GPS route recorded for this trip.')];
  if (!mc.enabled) return [route(pts), note];
  const line = pts.map((p) => [p.lat, p.lon]);
  const m = createMap(mc, { height: 300 });
  m.show({ routes: [{ points: line }], markers: [{ lat: line[0][0], lon: line[0][1], color: '#059669' }, { lat: line[line.length - 1][0], lon: line[line.length - 1][1] }] });
  return [m.el, note];
}

export default async function tripDetail(root, [id], session) {
  const saver = isSaver();
  const [{ trip: t, ...data }, mc] = await Promise.all([api(`/api/trips/${encodeURIComponent(id)}${saver ? '?samples=0' : ''}`), mapConfig()]);
  const routeBox = h('section', { class: 'card' }, h('h2', {}, 'Route'));
  if (saver) {
    routeBox.append(h('p', { class: 'muted' }, 'Data saver is on, so the route is not downloaded automatically.'),
      h('button', { class: 'sec', onclick: async (e) => {
        e.target.disabled = true;
        const full = await api(`/api/trips/${encodeURIComponent(id)}`);
        routeBox.replaceChildren(h('h2', {}, 'Route'), ...routeFor(full, mc));
      } }, 'Load route'));
  } else {
    routeBox.append(...routeFor(data, mc));
  }
  const notes = h('input', { value: t.notes || '', placeholder: 'Notes (client, purpose…)', maxlength: 200 });
  const msg = h('p', { class: 'muted', role: 'status' });
  const admin = session.me.role === 'admin';
  const set = (category) => async () => {
    try { await api(`/api/trips/${t.id}`, { method: 'PATCH', body: { category, notes: notes.value } }); location.reload(); } catch (e) { msg.textContent = e.message; }
  };
  const kwh = t.startSoc != null && t.endSoc != null ? null : null;
  root.append(h('p', {}, h('a', { href: '#/trips' }, '← Trips')),
    h('section', { class: 'card' }, h('div', { class: 'row', style: 'border:0;padding:0' }, h('h2', { class: 'grow' }, when(t.startTs, t.tz)), catChip(t)),
      t.startPlace || t.endPlace ? h('p', {}, h('b', {}, `${t.startPlace || '…'} → ${t.endPlace || '…'}`)) : null,
      h('div', { class: 'grid' }, stat('Distance', dist(t.distanceKm)), stat('Time', duration(t.startTs, t.endTs)), stat('Top speed', speed(t.maxSpeedKph)),
        stat('Battery used', t.startSoc != null && t.endSoc != null ? num(t.startSoc - t.endSoc, 0, '%') : '—'), stat('Start', pct(t.startSoc)), stat('End', pct(t.endSoc)),
        stat('Energy used', kwhText(t.energyKwh, t.energyEstimated)), stat(t.energyEstimated ? 'Efficiency (estimate)' : 'Efficiency', efficiency(t.distanceKm, t.energyKwh) || '—')),
      kwh),
    routeBox,
    admin && !t.open ? placeCard(t) : null,
    admin && !t.open ? shareCard('trip', t.id) : null,
    admin ? h('section', { class: 'card' }, h('h2', {}, 'Classify this trip'),
      h('label', { for: 'nt' }, 'Notes'), (notes.id = 'nt', notes),
      h('p', { class: 'seg' }, h('button', { onclick: set('work') }, 'Work'), h('button', { onclick: set('personal') }, 'Personal')),
      h('p', { class: 'muted' }, t.categorySource === 'manual' ? 'Set by you; rules will not change it.' : t.categorySource === 'rule' ? 'Set by a rule. Choosing here overrides it.' : 'Default. Choose to override.'), msg) : null);
}
