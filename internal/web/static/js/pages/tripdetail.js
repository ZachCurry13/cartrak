import { h, stat, catChip, dist, speed, when, duration, pct, num, isMetric } from '../ui.js';
import { api } from '../api.js';
import { isSaver } from '../mode.js';
import { createMap, mapConfig } from '../map.js';
import { effSpan, kwhText } from '../energy.js';
import { shareCard } from '../sharecard.js';
import { categoryList } from '../categories.js';
import { tripDistanceCard } from '../tripdistance.js';
import { placeCard } from '../placecard.js';
import { mapButtons } from '../maplinks.js';
import { tripFixCard } from '../tripfix.js';
import { whyCard } from '../tripwhy.js';

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
  const est = phone.length < 2 && d.estimate && d.estimate.points && d.estimate.points.length >= 2 ? d.estimate : null;
  if (est && mc.enabled) {
    const line = est.points;
    const m = createMap(mc, { height: 300 });
    m.show({ routes: [{ points: line, dash: '0.1 9', cap: 'round', width: 5, opacity: 0.85 }], markers: [{ lat: line[0][0], lon: line[0][1], color: '#059669' }, { lat: line[line.length - 1][0], lon: line[line.length - 1][1] }] });
    m.el.append(h('span', { class: 'maptag' }, 'Estimated route'));
    return [m.el, h('p', { class: 'muted' }, 'Estimated by a routing service from the positions the car recorded, not where it really went. Phone GPS shows the real path.')];
  }
  if (pts.length < 2) return [h('p', { class: 'muted' }, 'No GPS route recorded for this trip.')];
  if (!mc.enabled) return [route(pts), note];
  const line = pts.map((p) => [p.lat, p.lon]);
  const m = createMap(mc, { height: 300 });
  m.show({ routes: [{ points: line }], markers: [{ lat: line[0][0], lon: line[0][1], color: '#059669' }, { lat: line[line.length - 1][0], lon: line[line.length - 1][1] }] });
  return [m.el, note];
}

// How this trip's efficiency compares with the last year's average, as one plain sentence (or null).
async function versusAverage(t) {
  if (t.open || !(t.energyKwh >= 0.3) || !(t.distanceKm >= 1.6)) return null;
  const d = await api('/api/insights').catch(() => null);
  if (!d || !d.months || d.trips < 5) return null;
  const km = d.months.reduce((a, m) => a + m.km, 0), kwh = d.months.reduce((a, m) => a + m.kwh, 0);
  if (!(kwh > 0)) return null;
  const avg = km / kwh, mine = t.distanceKm / t.energyKwh;
  const diff = Math.round(((mine - avg) / avg) * 100);
  if (Math.abs(diff) < 3) return h('p', { class: 'muted' }, 'About the same efficiency as your average over the last year.');
  return h('p', { class: 'muted' }, `${Math.abs(diff)}% ${diff > 0 ? 'more' : 'less'} efficient than your average over the last year.`);
}

export default async function tripDetail(root, [id, action], session) {
  const saver = isSaver();
  const [{ trip: t, ...data }, mc] = await Promise.all([api(`/api/trips/${encodeURIComponent(id)}${saver ? '?samples=0' : ''}`), mapConfig()]);
  const routeBox = h('section', { class: 'card' }, h('h2', {}, 'Route'));
  if (saver) {
    routeBox.append(h('p', { class: 'muted' }, 'Data saver is on, so the route is not downloaded automatically.'),
      h('button', { class: 'sec', onclick: async (e) => {
        e.target.disabled = true;
        const full = await api(`/api/trips/${encodeURIComponent(id)}`);
        routeBox.replaceChildren(h('h2', {}, 'Route'), ...routeFor(full, mc).filter(Boolean));
      } }, 'Load route'));
  } else {
    routeBox.append(...routeFor(data, mc).filter(Boolean));
  }
  const notes = h('input', { value: t.notes || '', placeholder: 'Notes (client, purpose…)', maxlength: 200 });
  const msg = h('p', { class: 'muted', role: 'status' });
  const admin = session.me.role === 'admin';
  const set = (category) => async () => {
    try { await api(`/api/trips/${t.id}`, { method: 'PATCH', body: { category, notes: notes.value } }); location.reload(); } catch (e) { msg.textContent = e.message; }
  };
  const avg = saver ? null : await versusAverage(t); // one extra request, skipped on data saver
  const why = saver || t.open ? null : await whyCard(t.id);
  const hours = t.endTs && t.startTs ? (t.endTs - t.startTs) / 3600 : 0;
  const avgKph = hours > 0 && t.distanceKm != null ? t.distanceKm / hours : null;
  const hasStart = t.startLat != null && t.startLon != null;
  const hasEnd = t.endLat != null && t.endLon != null && !t.open;
  root.append(...[h('p', {}, h('a', { href: '#/trips' }, '← Trips'), ' · ', h('a', { href: `#/day/${new Date(t.startTs * 1000).toLocaleDateString('en-CA', t.tz ? { timeZone: t.tz } : {})}` }, 'See the whole day')),
    h('section', { class: 'card' }, h('div', { class: 'row', style: 'border:0;padding:0' }, h('h2', { class: 'grow' }, when(t.startTs, t.tz)), catChip(t)),
      t.startPlace || t.endPlace ? h('p', {}, h('b', {}, `${t.startPlace || '…'} → ${t.endPlace || '…'}`)) : null,
      h('div', { class: 'grid' }, stat('Distance', dist(t.distanceKm)), stat('Time', duration(t.startTs, t.endTs)), stat('Average speed', speed(avgKph)), stat('Top speed', speed(t.maxSpeedKph)),
        stat('Battery used', t.startSoc != null && t.endSoc != null ? num(t.startSoc - t.endSoc, 0, '%') : '—'), stat('Start', pct(t.startSoc)), stat('End', pct(t.endSoc)),
        stat('Energy used', kwhText(t.energyKwh, t.energyEstimated)), stat(t.energyEstimated ? 'Efficiency (estimate)' : 'Efficiency', effSpan(t.distanceKm, t.energyKwh, t.energyEstimated ? '≈ ' : '')),
        t.estCost != null ? stat('Energy cost (estimate)', `≈ ${t.currency || '$'}${t.estCost.toFixed(2)}${t.distanceKm >= 1.6 ? ` (${t.currency || '$'}${(t.estCost / t.distanceKm * (isMetric() ? 1 : 1.609344)).toFixed(2)}/${isMetric() ? 'km' : 'mi'})` : ''}`) : null),
      avg),
    why,
    routeBox,
    hasStart || hasEnd ? h('section', { class: 'card' }, h('h2', {}, 'Open in maps'),
      hasStart ? h('div', {}, h('b', {}, t.startPlace || 'Where it started'), mapButtons(t.startLat, t.startLon, 'where it started')) : null,
      hasEnd ? h('div', {}, h('b', {}, t.endPlace || 'Where it ended'), mapButtons(t.endLat, t.endLon, 'where it ended')) : null) : null,
    admin && !t.open ? placeCard(t) : null,
    admin && !t.open ? shareCard('trip', t.id) : null,
    admin ? h('section', { class: 'card' }, h('h2', {}, 'Classify this trip'),
      h('label', { for: 'nt' }, 'Notes'), (notes.id = 'nt', notes),
      h('p', { class: 'seg' }, ...categoryList().map((c) => h('button', { class: c.key === t.category ? '' : 'sec', onclick: set(c.key) }, c.label))),
      h('p', { class: 'muted' }, t.categorySource === 'manual' ? 'Set by you; rules will not change it.' : t.categorySource === 'rule' ? 'Set by a rule. Choosing here overrides it.' : 'Default. Choose to override.'), msg) : null,
    admin ? tripDistanceCard(t) : null,
    admin ? tripFixCard(t) : null].filter(Boolean));
  if (action === 'share') { // opened from the Share button on Home: bring the share box into view
    setTimeout(() => { const el = root.querySelector('#sht'); if (el) { el.closest('section').scrollIntoView({ block: 'start' }); el.focus(); } }, 60);
  }
}
