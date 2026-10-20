import { h } from './ui.js';
import { createMap } from './map.js';

// The page behind a public share link. It shows only what the link's owner chose to share, and it does not
// use the CarTrak app, an account, or any outside service.
const base = location.pathname.replace(/\/$/, '');
const root = document.getElementById('share');

const fmtDist = (km, units, d = 1) => (units === 'km' ? `${km.toFixed(d)} km` : `${(km * 0.621371).toFixed(d)} mi`);
const fmtTime = (sec) => { const m = Math.round(sec / 60); return m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`; };
const tile = (label, value) => h('div', { class: 'stat' }, h('div', { class: 'v' }, value), h('div', { class: 'l' }, label));

const fmtSpeed = (kph, units) => (units === 'km' ? `${Math.round(kph)} km/h` : `${Math.round(kph * 0.621371)} mph`);
const fmtTemp = (c, units) => (units === 'km' ? `${Math.round(c)}°C` : `${Math.round(c * 9 / 5 + 32)}°F`);

// The efficiency detail a link was set up to show: speeds, the weather, and what made the trip use more or less energy.
function insightsCard(d) {
  const i = d.insights;
  const vs = i.vsUsualPct;
  return h('section', { class: 'card' }, h('h2', {}, 'How it went'),
    h('div', { class: 'grid' }, tile('Top speed', fmtSpeed(i.topKph, d.units)), i.avgKph ? tile('Average speed', fmtSpeed(i.avgKph, d.units)) : null,
      i.tempC != null ? tile('Temperature', fmtTemp(i.tempC, d.units)) : null, i.stops != null ? tile('Stops', String(i.stops)) : null),
    vs != null ? h('p', {}, Math.abs(vs) < 3 ? 'About as efficient as the driver\'s usual.' : `${Math.abs(Math.round(vs))}% ${vs > 0 ? 'more' : 'less'} energy per ${d.units === 'km' ? 'km' : 'mile'} than the driver\'s usual.`) : null,
    i.tags && i.tags.length ? h('p', { class: 'tags' }, ...i.tags.map((t) => h('span', { class: `chip ${t.level === 'bad' ? 'bad' : t.level === 'good' ? 'ok' : 'work'}` }, t.label))) : null);
}

function plainRoute(routes) {
  const NS = 'http://www.w3.org/2000/svg';
  const pts = routes.flat();
  const lats = pts.map((p) => p[0]), lons = pts.map((p) => p[1]);
  const minLat = Math.min(...lats), maxLat = Math.max(...lats), minLon = Math.min(...lons), maxLon = Math.max(...lons);
  const k = Math.cos(((minLat + maxLat) / 2) * Math.PI / 180);
  const w = Math.max((maxLon - minLon) * k, 1e-5), ht = Math.max(maxLat - minLat, 1e-5);
  const sc = 280 / Math.max(w, ht), pad = 10;
  const svg = document.createElementNS(NS, 'svg');
  svg.setAttribute('class', 'route');
  svg.setAttribute('viewBox', `0 0 ${w * sc + pad * 2} ${ht * sc + pad * 2}`);
  for (const r of routes) {
    const line = document.createElementNS(NS, 'polyline');
    line.setAttribute('points', r.map((p) => `${pad + (p[1] - minLon) * k * sc},${pad + (maxLat - p[0]) * sc}`).join(' '));
    line.setAttribute('fill', 'none'); line.setAttribute('stroke', 'currentColor'); line.setAttribute('stroke-width', '3'); line.setAttribute('stroke-linejoin', 'round');
    svg.append(line);
  }
  return svg;
}

async function main() {
  const r = await fetch(`${base}/data`, { cache: 'no-store', referrerPolicy: 'no-referrer' });
  if (!r.ok) {
    document.getElementById('title').textContent = 'Link not available';
    return root.replaceChildren(h('p', { class: 'muted' }, 'This link does not exist, has expired, or was turned off by its owner.'));
  }
  const d = await r.json();
  const journey = d.kind === 'journey' || d.kind === 'day';
  const title = d.title || (d.kind === 'day' ? 'A day of driving' : journey ? 'A journey' : 'A trip');
  document.title = title;
  document.getElementById('title').textContent = title;
  const when = d.start ? `${new Date(d.start * 1000).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short', timeZone: d.tz || undefined })} to ${new Date(d.end * 1000).toLocaleTimeString([], { timeStyle: 'short', timeZone: d.tz || undefined })}` : '';
  const eff = d.kmPerKwh ? (d.units === 'km' ? `${(100 / d.kmPerKwh).toFixed(1)} kWh/100 km` : `${(d.kmPerKwh * 0.621371).toFixed(2)} mi/kWh`) : null;
  const mapBox = h('div');
  root.replaceChildren(...[
    h('section', { class: 'card' },
      d.origin || d.dest ? h('p', {}, h('b', {}, [d.origin, d.dest].filter(Boolean).join(' → '))) : null,
      when ? h('p', { class: 'muted' }, when) : null,
      h('div', { class: 'grid' }, tile('Distance', fmtDist(d.distanceKm, d.units)), tile('Driving time', fmtTime(d.durationSec)),
        ...(d.kind === 'day' ? [tile('Trips', String(d.legs))] : []),
        ...(journey ? [tile('Elapsed', d.elapsedSec != null ? fmtTime(d.elapsedSec) : '—'), tile('Stops', String(d.stops))] : []),
        ...(d.energyKwh != null ? [tile('Energy used', `${d.energyEstimated ? '≈ ' : ''}${d.energyKwh.toFixed(1)} kWh`), eff ? tile('Efficiency', eff) : null] : []).filter(Boolean))),
    d.insights ? insightsCard(d) : null,
    d.routeHidden ? h('p', { class: 'muted' }, 'The route is hidden because the trip is too short to show without revealing where it started or ended.') : null,
    mapBox,
    h('p', { class: 'muted foot' }, 'Shared from a CarTrak. This page shows only what its owner chose to share.')].filter(Boolean));
  if (d.routes && d.routes.length) {
    if (d.mapsOn) {
      const m = createMap({ tiles: `${base}/tile/{z}/{x}/{y}`, attribution: '© OpenStreetMap contributors' }, { height: 360 });
      mapBox.replaceChildren(h('section', { class: 'card' }, m.el));
      m.show({ routes: d.routes.map((pts) => ({ points: pts, color: '#2fd4a0', width: 4, opacity: 0.95 })) });
    } else {
      mapBox.replaceChildren(h('section', { class: 'card', style: 'color:var(--accent)' }, plainRoute(d.routes)));
    }
  }
}
main().catch(() => { root.replaceChildren(h('p', { class: 'muted' }, 'Could not load this page.')); });
