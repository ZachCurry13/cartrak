import { h } from './ui.js';
import { api } from './api.js';

// A small slippy map with no library: 256 px tiles from CarTrak's own tile address, an SVG layer
// for routes and markers, drag / pinch / wheel / buttons to move and zoom.
const NS = 'http://www.w3.org/2000/svg';
const TILE = 256;
const MIN_ZOOM = 2;
const MAX_ZOOM = 18;

let cfgPromise = null;
// Whether maps are on, and the credit line. One request per page load.
export const mapConfig = () => (cfgPromise ||= api('/api/map').catch(() => ({ enabled: false })));

const project = (lat, lon, z) => {
  const s = TILE * 2 ** z;
  const r = (Math.max(-85.0511, Math.min(85.0511, lat)) * Math.PI) / 180;
  return [((lon + 180) / 360) * s, ((1 - Math.log(Math.tan(r) + 1 / Math.cos(r)) / Math.PI) / 2) * s];
};
const unproject = (x, y, z) => {
  const s = TILE * 2 ** z;
  return [(180 / Math.PI) * Math.atan(Math.sinh(Math.PI - (2 * Math.PI * y) / s)), (x / s) * 360 - 180];
};
const svgEl = (tag, attrs) => {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v);
  return e;
};

// glow: a dark map with every route drawn faint and blended, so streets driven often light up (the Insights map).
export function createMap(cfg, { height = 260, interactive = true, glow = false } = {}) {
  const tiles = h('div', { class: 'map-tiles' });
  const svg = svgEl('svg', { class: 'map-svg' });
  const credit = h('div', { class: 'map-credit' }, /openstreetmap/i.test(cfg.attribution || '') ? h('a', { href: 'https://www.openstreetmap.org/copyright', target: '_blank', rel: 'noopener' }, cfg.attribution) : cfg.attribution || '');
  const el = h('div', { class: `map${interactive ? ' live' : ''}${glow ? ' glow' : ''}`, style: `height:${height}px` }, tiles, svg, credit);
  let center = [0, 0];
  let zoom = MIN_ZOOM;
  let routes = [];
  let markers = [];
  let pendingFit = null;
  const imgs = new Map();

  function render() {
    const w = el.clientWidth;
    const ht = el.clientHeight;
    if (!w || !ht) return;
    if (pendingFit) { const p = pendingFit; pendingFit = null; fitNow(p); }
    const [cx, cy] = project(center[0], center[1], zoom);
    const left = cx - w / 2;
    const top = cy - ht / 2;
    const n = 2 ** zoom;
    const want = new Set();
    for (let y = Math.floor(top / TILE); y <= Math.floor((top + ht) / TILE); y++) {
      if (y < 0 || y >= n) continue;
      for (let x = Math.floor(left / TILE); x <= Math.floor((left + w) / TILE); x++) {
        const key = `${zoom}/${x}/${y}`;
        want.add(key);
        let img = imgs.get(key);
        if (!img) {
          const wrapped = ((x % n) + n) % n;
          img = h('img', { alt: '', draggable: 'false', decoding: 'async', src: cfg.tiles.replace('{z}', zoom).replace('{x}', wrapped).replace('{y}', y) });
          imgs.set(key, img);
          tiles.append(img);
        }
        img.style.transform = `translate(${Math.round(x * TILE - left)}px,${Math.round(y * TILE - top)}px)`;
      }
    }
    for (const [key, img] of imgs) if (!want.has(key)) { img.remove(); imgs.delete(key); }
    drawOverlay(left, top, w, ht);
  }

  function drawOverlay(left, top, w, ht) {
    svg.setAttribute('viewBox', `0 0 ${w} ${ht}`);
    svg.replaceChildren();
    const xy = ([lat, lon]) => { const [x, y] = project(lat, lon, zoom); return `${(x - left).toFixed(1)},${(y - top).toFixed(1)}`; };
    // in glow mode the routes share one blurred, blended layer: one filter pass however many trips there are
    let layer = svg;
    if (glow) {
      const f = svgEl('filter', { id: 'mapglow', x: '-10%', y: '-10%', width: '120%', height: '120%' });
      const merge = svgEl('feMerge', {});
      merge.append(svgEl('feMergeNode', { in: 'b' }), svgEl('feMergeNode', { in: 'SourceGraphic' }));
      f.append(svgEl('feGaussianBlur', { stdDeviation: '2', result: 'b' }), merge);
      const defs = svgEl('defs', {});
      defs.append(f);
      layer = svgEl('g', { filter: 'url(#mapglow)', class: 'map-glow' });
      svg.append(defs, layer);
    }
    for (const r of routes) {
      const pts = r.points.map(xy).join(' ');
      layer.append(svgEl('polyline', { points: pts, fill: 'none', stroke: r.color || '#1b5fc4', 'stroke-width': r.width || 4, 'stroke-linejoin': 'round', 'stroke-linecap': r.cap || (r.dash ? 'butt' : 'round'), 'stroke-dasharray': r.dash || null, opacity: r.opacity || 1 }));
      if (r.onClick) {
        const hit = svgEl('polyline', { points: pts, fill: 'none', stroke: 'transparent', 'stroke-width': 16, class: 'map-hit' });
        hit.style.cursor = 'pointer';
        hit.addEventListener('click', r.onClick);
        svg.append(hit);
      }
    }
    for (const m of markers) {
      const [x, y] = xy([m.lat, m.lon]).split(',');
      const big = m.label != null ? 11 : 7;
      svg.append(svgEl('circle', { cx: x, cy: y, r: big, fill: m.color || '#dc2626', stroke: '#fff', 'stroke-width': 2.5 }));
      if (m.label != null) { const t = svgEl('text', { x, y: Number(y) + 4, 'text-anchor': 'middle', 'font-size': 11, 'font-weight': 700, fill: '#fff' }); t.textContent = String(m.label); svg.append(t); }
    }
  }

  function fitNow(points) {
    const w = el.clientWidth - 48;
    const ht = el.clientHeight - 48;
    const lats = points.map((p) => p[0]);
    const lons = points.map((p) => p[1]);
    const box = [Math.min(...lats), Math.min(...lons), Math.max(...lats), Math.max(...lons)];
    let z = MAX_ZOOM - 1;
    for (; z > MIN_ZOOM; z--) {
      const [x0, y1] = project(box[0], box[1], z);
      const [x1, y0] = project(box[2], box[3], z);
      if (x1 - x0 <= w && y1 - y0 <= ht) break;
    }
    zoom = z;
    center = [(box[0] + box[2]) / 2, (box[1] + box[3]) / 2];
  }

  const setZoom = (z) => { zoom = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, z)); render(); };
  const panBy = (dx, dy) => {
    const [cx, cy] = project(center[0], center[1], zoom);
    center = unproject(cx - dx, cy - dy, zoom);
    render();
  };

  if (interactive) {
    const down = new Map();
    let pinch = 0;
    el.addEventListener('pointerdown', (e) => { if (e.target.closest('.map-ctl, .map-credit')) return; down.set(e.pointerId, [e.clientX, e.clientY]); el.setPointerCapture(e.pointerId); });
    el.addEventListener('pointermove', (e) => {
      const prev = down.get(e.pointerId);
      if (!prev) return;
      if (down.size === 1) panBy(e.clientX - prev[0], e.clientY - prev[1]);
      down.set(e.pointerId, [e.clientX, e.clientY]);
      if (down.size === 2) {
        const [a, b] = [...down.values()];
        const d = Math.hypot(a[0] - b[0], a[1] - b[1]);
        if (pinch && Math.abs(Math.log2(d / pinch)) > 0.6) { setZoom(zoom + (d > pinch ? 1 : -1)); pinch = d; }
        if (!pinch) pinch = d;
      }
    });
    const up = (e) => { down.delete(e.pointerId); if (down.size < 2) pinch = 0; };
    el.addEventListener('pointerup', up);
    el.addEventListener('pointercancel', up);
    el.addEventListener('wheel', (e) => { e.preventDefault(); setZoom(zoom + (e.deltaY < 0 ? 1 : -1)); }, { passive: false });
    el.addEventListener('dblclick', () => setZoom(zoom + 1));
    const btn = (label, title, f) => h('button', { type: 'button', class: 'map-btn', title, 'aria-label': title, onclick: f }, label);
    el.append(h('div', { class: 'map-ctl' }, btn('+', 'Zoom in', () => setZoom(zoom + 1)), btn('−', 'Zoom out', () => setZoom(zoom - 1))));
  }

  new ResizeObserver(() => { try { render(); } catch (e) { console.error("map render failed", e); } }).observe(el);

  return {
    el,
    // Sets what is drawn and zooms to fit it. Safe to call before the map is on the page.
    show({ routes: r = [], markers: m = [], fit }) {
      routes = r;
      markers = m;
      const pts = fit || [...r.flatMap((x) => x.points), ...m.map((x) => [x.lat, x.lon])];
      if (pts.length) pendingFit = pts;
      render();
    },
    update(m) { markers = m; render(); },
  };
}
