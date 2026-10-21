import { h, dist, speed, pct } from './ui.js';
import { api } from './api.js';
import { createMap, mapConfig } from './map.js';

// Trip replay: the trip played back on its map. A dot drives the route while the clock, speed, battery and distance follow it;
// drag the slider to jump, choose how fast it plays. The route already driven is drawn bright, the rest faint.

const SPEEDS = [[10, '10×'], [30, '30×'], [60, '60×'], [120, '120×']];

// where the car was at time ts, between two points, and the values there
function at(pts, ts) {
  let i = 0;
  while (i + 1 < pts.length && pts[i + 1].ts <= ts) i++;
  const a = pts[i], b = pts[Math.min(i + 1, pts.length - 1)];
  const k = b.ts > a.ts ? Math.max(0, Math.min(1, (ts - a.ts) / (b.ts - a.ts))) : 0;
  const mix = (x, y) => (x == null ? y : y == null ? x : x + (y - x) * k);
  return { i, lat: mix(a.lat, b.lat), lon: mix(a.lon, b.lon), kph: mix(a.kph, b.kph), soc: mix(a.soc, b.soc), km: mix(a.km, b.km) };
}

export function replayCard(t, color) {
  const box = h('div');
  const start = h('button', { type: 'button', onclick: () => open() }, '▶ Replay this trip');
  const card = h('section', { class: 'card' }, h('h2', {}, 'Replay'), h('p', { class: 'muted' }, 'Watch the trip play back on the map, with its speed and battery.'), h('p', {}, start), box);
  async function open() {
    start.disabled = true;
    start.textContent = 'Loading…';
    let d;
    try { d = await api(`/api/trips/${t.id}/replay`); } catch (e) { start.textContent = e.message; return; }
    const cfg = await mapConfig();
    start.remove();
    card.querySelector('p.muted').remove();
    const pts = d.points;
    const t0 = pts[0].ts, t1 = pts[pts.length - 1].ts;
    const map = createMap(cfg.enabled ? cfg : { tiles: '', attribution: '' }, { height: 300 });
    const full = pts.map((p) => [p.lat, p.lon]);
    map.show({ routes: [{ points: full, color: '#94a3b8', width: 4, opacity: 0.5 }], fit: full });
    const clock = h('b'), spd = h('b'), soc = h('b'), km = h('b');
    const fmt = (ts) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', second: '2-digit', timeZone: d.tz || undefined });
    const stat = (label, el) => h('div', {}, el, h('span', {}, label));
    const slider = h('input', { type: 'range', min: 0, max: 1000, value: 0, 'aria-label': 'Where in the trip', style: 'width:100%' });
    let now = t0, rate = 30, playing = false, last = 0;
    const play = h('button', { type: 'button', class: 'on', style: 'min-width:6.5em' }, '▶ Play');
    const draw = () => {
      const p = at(pts, now);
      const done = [...full.slice(0, p.i + 1), [p.lat, p.lon]];
      map.redraw({ routes: [{ points: full, color: '#94a3b8', width: 4, opacity: 0.5 }, { points: done, color, width: 5 }], markers: [{ lat: p.lat, lon: p.lon, color }] });
      clock.textContent = fmt(now);
      spd.textContent = p.kph != null ? speed(p.kph) : '—';
      soc.textContent = p.soc != null ? pct(p.soc) : '—';
      km.textContent = dist(p.km || 0, 1);
      slider.value = String(Math.round(((now - t0) / (t1 - t0 || 1)) * 1000));
    };
    // twenty steps a second is plenty, and a plain timer keeps going where animation frames are held back
    const tick = () => {
      if (!playing || !card.isConnected) return;
      const ms = performance.now();
      if (last) now = Math.min(t1, now + ((ms - last) / 1000) * rate);
      last = ms;
      draw();
      if (now >= t1) { stop(); return; }
      setTimeout(tick, 50);
    };
    const stop = () => { playing = false; last = 0; play.textContent = now >= t1 ? '↺ Again' : '▶ Play'; };
    play.onclick = () => {
      if (playing) { stop(); return; }
      if (now >= t1) now = t0;
      playing = true;
      play.textContent = '⏸ Pause';
      tick();
    };
    slider.oninput = () => { now = t0 + (Number(slider.value) / 1000) * (t1 - t0); draw(); };
    const rates = h('div', { class: 'seg', style: 'margin:0' });
    const drawRates = () => rates.replaceChildren(...SPEEDS.map(([v, txt]) => h('button', { type: 'button', class: v === rate ? 'on' : '', 'aria-pressed': String(v === rate), onclick: () => { rate = v; drawRates(); } }, txt)));
    drawRates();
    box.append(map.el, h('div', { class: 'bigfig', style: 'margin:10px 0' }, stat('time', clock), stat('speed', spd), stat('battery', soc), stat('so far', km)),
      slider, h('div', { style: 'display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin-top:8px' }, play, rates),
      h('p', { class: 'muted', style: 'margin-top:8px' }, d.source === 'phone' ? `${pts.length} positions from the phone's GPS.` : d.source === 'estimate' ? "Along the likely road shown on the map, timed by the car's own positions, so it moves at the pace the car went." : `${pts.length} positions from the car, which reports only now and then; between them the dot moves in a straight line.`));
    draw();
  }
  return card;
}
