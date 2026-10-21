import { compareDistance, milesOrKm, distUnit, personalityLine, periodTitle, valueText } from './funwords.js';
import { efficiency } from './energy.js';

// The Wrapped infographic: one tall picture (1080 x 1920, a phone story) of a month or a year, drawn on a canvas so it can be
// shared or saved as an image. w is the Wrapped answer, routes the period's routes for the little glowing map.

const W = 1080, H = 1920, PAD = 72;
const FONT = 'system-ui, -apple-system, "Segoe UI", Roboto, sans-serif';

function text(g, s, x, y, size, { weight = 600, color = '#fff', align = 'left', alpha = 1, max } = {}) {
  g.globalAlpha = alpha;
  g.fillStyle = color;
  g.textAlign = align;
  g.font = `${weight} ${size}px ${FONT}`;
  g.fillText(s, x, y, max);
  g.globalAlpha = 1;
}

function wrap(g, s, x, y, size, width, lineH, opts) {
  g.font = `${opts && opts.weight || 600} ${size}px ${FONT}`;
  const words = s.split(' ');
  let line = '';
  for (const w of words) {
    if (g.measureText(`${line} ${w}`).width > width && line) { text(g, line, x, y, size, opts); y += lineH; line = w; } else line = line ? `${line} ${w}` : w;
  }
  text(g, line, x, y, size, opts);
  return y + lineH;
}

function tile(g, x, y, w, hgt, big, small) {
  g.fillStyle = 'rgba(255,255,255,.08)';
  g.beginPath(); g.roundRect(x, y, w, hgt, 28); g.fill();
  text(g, big, x + 32, y + 78, 56, { weight: 800, max: w - 64 });
  text(g, small, x + 32, y + 126, 30, { alpha: 0.75, max: w - 64 });
}

// the routes, fitted into a box, faint and added on top of each other so streets driven often glow
function glowMap(g, routes, x, y, w, hgt) {
  g.save();
  g.beginPath(); g.roundRect(x, y, w, hgt, 32); g.clip();
  g.fillStyle = '#04070d'; g.fillRect(x, y, w, hgt);
  const pts = routes.flat();
  if (pts.length > 1) {
    // frame where most of the driving is (the middle 90% each way), so one long trip does not shrink the rest to a dot
    const lats = pts.map((p) => p[0]).sort((a, b) => a - b), lons = pts.map((p) => p[1]).sort((a, b) => a - b);
    const q = (arr, f) => arr[Math.floor(f * (arr.length - 1))];
    const few = pts.length < 40;
    const la0 = few ? lats[0] : q(lats, 0.05), la1 = few ? lats[lats.length - 1] : q(lats, 0.95), lo0 = few ? lons[0] : q(lons, 0.05), lo1 = few ? lons[lons.length - 1] : q(lons, 0.95);
    const k = Math.cos(((la0 + la1) / 2) * Math.PI / 180);
    const sc = Math.min((w - 80) / ((lo1 - lo0) * k || 1e-6), (hgt - 80) / ((la1 - la0) || 1e-6));
    const ox = x + (w - (lo1 - lo0) * k * sc) / 2, oy = y + (hgt - (la1 - la0) * sc) / 2;
    g.globalCompositeOperation = 'lighter';
    g.lineJoin = g.lineCap = 'round';
    const boost = Math.min(5, Math.max(1, 60 / routes.length)); // a few trips are drawn brighter, many add up on their own
    for (const [width, alpha] of [[10, 0.035 * boost], [3, Math.min(0.9, 0.16 * boost)]]) {
      g.lineWidth = width;
      g.strokeStyle = `rgba(94,234,212,${alpha})`;
      for (const r of routes) {
        g.beginPath();
        r.forEach((p, i) => { const px = ox + (p[1] - lo0) * k * sc, py = oy + (la1 - p[0]) * sc; i ? g.lineTo(px, py) : g.moveTo(px, py); });
        g.stroke();
      }
    }
  }
  g.restore();
}

// where the energy went, as one bar with a key under it
function energyBar(g, f, x, y, width) {
  if (!f) return;
  const parts = [['City', f.cityKwh, '#f59e0b'], ['Highway', f.highwayKwh, '#f97316'], ['Climate, parked', f.otherKwh, '#a78bfa'], ['Lost', f.lostKwh, '#94a3b8']].filter((p) => p[1] > 0.5);
  const total = parts.reduce((a, p) => a + p[1], 0);
  if (total < 1) return;
  text(g, 'Where the energy went', x, y + 24, 30, { alpha: 0.7, weight: 700 });
  let px = x;
  g.save(); g.beginPath(); g.roundRect(x, y + 48, width, 44, 14); g.clip();
  for (const [, v, col] of parts) { const pw = (v / total) * width; g.fillStyle = col; g.fillRect(px, y + 48, pw, 44); px += pw; }
  g.restore();
  let kx = x, ky = y + 116;
  g.font = `600 26px ${FONT}`;
  for (const [name, v, col] of parts) {
    const label = `${name} ${Math.round((v / total) * 100)}%`;
    const lw = 32 + g.measureText(label).width + 30;
    if (kx > x && kx + lw - 30 > x + width) { kx = x; ky += 44; } // wrap the key rather than run off the edge
    g.fillStyle = col; g.fillRect(kx, ky, 22, 22);
    text(g, label, kx + 32, ky + 20, 26, { alpha: 0.85 });
    g.font = `600 26px ${FONT}`;
    kx += lw;
  }
}

// how tall the energy block is: its title, the bar and one or two lines of key
const ENERGY_H = 200;

// colors: the season's two colours for the glow behind the picture, or null for the usual teal and blue.
export function drawInfographic(w, routes, colors = null) {
  const c = document.createElement('canvas');
  c.width = W; c.height = H;
  const g = c.getContext('2d');
  const r = w.recap;
  const bg = g.createLinearGradient(0, 0, W, H);
  bg.addColorStop(0, '#0b1a2b'); bg.addColorStop(1, '#070b14');
  g.fillStyle = bg; g.fillRect(0, 0, W, H);
  const tint = (hex, a) => `rgba(${parseInt(hex.slice(1, 3), 16)},${parseInt(hex.slice(3, 5), 16)},${parseInt(hex.slice(5, 7), 16)},${a})`;
  const glows = colors ? [tint(colors[0], 0.4), tint(colors[1], 0.3)] : ['rgba(34,211,165,.35)', 'rgba(59,130,246,.25)'];
  for (const [cx, cy, rad, col] of [[W, 0, 900, glows[0]], [0, H * 0.55, 800, glows[1]]]) {
    const rg = g.createRadialGradient(cx, cy, 0, cx, cy, rad);
    rg.addColorStop(0, col); rg.addColorStop(1, 'rgba(0,0,0,0)');
    g.fillStyle = rg; g.fillRect(0, 0, W, H);
  }
  text(g, 'CARTRAK · WRAPPED', PAD, 120, 30, { alpha: 0.7, weight: 700 });
  text(g, periodTitle(r.period), PAD, 200, 72, { weight: 800 });
  text(g, milesOrKm(r.km), PAD, 380, 190, { weight: 800 });
  let y = wrap(g, `${distUnit()} · ${compareDistance(r.km)}`, PAD, 450, 40, W - PAD * 2, 52, { weight: 600 });
  const tw = (W - PAD * 2 - 28) / 2;
  const eff = r.kmPerKwh > 0 ? efficiency(r.kmPerKwh * 100, 100) : '—';
  const cur = w.currency || '$';
  y += 20;
  tile(g, PAD, y, tw, 160, `${r.trips}`, `trips · ${Math.round(r.hours)} h driving`);
  tile(g, PAD + tw + 28, y, tw, 160, eff, 'efficiency');
  tile(g, PAD, y + 188, tw, 160, valueText('kwh', r.kwhCharged), `charged · ${Math.round(r.homeShare * 100)}% at home`);
  tile(g, PAD + tw + 28, y + 188, tw, 160, r.saved != null ? `${cur}${Math.round(r.saved)} saved` : `${cur}${r.cost.toFixed(0)}`, r.saved != null ? 'against a gas car' : 'charging cost');
  y += 188 * 2 + 30;
  // the parts under the map have their own heights; the map takes what is left, so nothing runs into the footer
  const p = (w.personalities || [])[0];
  const badges = (w.badges || []).slice(0, 6);
  const below = (p ? 190 : 0) + (badges.length ? 110 : 0) + (w.flow ? ENERGY_H : 0) + 70;
  const mapH = Math.max(260, H - y - below - 60);
  glowMap(g, routes, PAD, y, W - PAD * 2, mapH);
  text(g, w.towns ? `${w.towns} towns` : 'Everywhere you went', PAD + 32, y + 60, 34, { weight: 700 });
  y += mapH + 80;
  if (p) {
    text(g, p.emoji, PAD, y + 40, 96, {});
    text(g, p.name, PAD + 130, y, 52, { weight: 800 });
    y = wrap(g, personalityLine(p), PAD + 130, y + 56, 32, W - PAD * 2 - 130, 42, { alpha: 0.8 });
    y += 30;
  }
  if (badges.length) {
    text(g, 'Earned', PAD, y + 20, 30, { alpha: 0.7, weight: 700 });
    badges.forEach((b, i) => text(g, b.emoji, PAD + 150 + i * 110, y + 34, 72, {}));
    y += 110;
  }
  energyBar(g, w.flow, PAD, y + 10, W - PAD * 2);
  text(g, 'Made with CarTrak', W / 2, H - 40, 28, { alpha: 0.55, align: 'center' });
  return c;
}

// Share the picture, or save it when sharing a file is not possible here.
export async function shareImage(canvas, name) {
  const blob = await new Promise((res) => canvas.toBlob(res, 'image/png'));
  const file = new File([blob], `${name}.png`, { type: 'image/png' });
  if (navigator.canShare && navigator.canShare({ files: [file] })) {
    try { await navigator.share({ files: [file], title: name }); return 'shared'; } catch (e) { if (e.name === 'AbortError') return 'cancelled'; }
  }
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = `${name}.png`;
  document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 4000);
  return 'saved';
}
