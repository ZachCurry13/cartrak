import { h, stat, catChip, dist, speed, when, duration, pct, num } from '../ui.js';
import { api } from '../api.js';
import { isSaver } from '../mode.js';

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

export default async function tripDetail(root, [id], session) {
  const saver = isSaver();
  const { trip: t, samples } = await api(`/api/trips/${encodeURIComponent(id)}${saver ? '?samples=0' : ''}`);
  const routeBox = h('section', { class: 'card' }, h('h2', {}, 'Route'));
  if (saver) {
    routeBox.append(h('p', { class: 'muted' }, 'Data saver is on, so the route is not downloaded automatically.'),
      h('button', { class: 'sec', onclick: async (e) => {
        e.target.disabled = true;
        const full = await api(`/api/trips/${encodeURIComponent(id)}`);
        routeBox.replaceChildren(h('h2', {}, 'Route'), route(full.samples));
      } }, 'Load route'));
  } else {
    routeBox.append(route(samples));
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
      h('div', { class: 'grid' }, stat('Distance', dist(t.distanceKm)), stat('Time', duration(t.startTs, t.endTs)), stat('Top speed', speed(t.maxSpeedKph)),
        stat('Battery used', t.startSoc != null && t.endSoc != null ? num(t.startSoc - t.endSoc, 0, '%') : '—'), stat('Start', pct(t.startSoc)), stat('End', pct(t.endSoc))),
      kwh),
    routeBox,
    admin ? h('section', { class: 'card' }, h('h2', {}, 'Classify this trip'),
      h('label', { for: 'nt' }, 'Notes'), (notes.id = 'nt', notes),
      h('p', { class: 'seg' }, h('button', { onclick: set('work') }, 'Work'), h('button', { onclick: set('personal') }, 'Personal')),
      h('p', { class: 'muted' }, t.categorySource === 'manual' ? 'Set by you; rules will not change it.' : t.categorySource === 'rule' ? 'Set by a rule. Choosing here overrides it.' : 'Default. Choose to override.'), msg) : null);
}
