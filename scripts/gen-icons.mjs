// Draws the app icons (a car seen from above with two trails of tire tracks behind it) and writes them to internal/web/static/icons.
// Run from the repository root: node scripts/gen-icons.mjs
// All art is defined once, in a 512 x 512 space, and drawn by a small anti-aliased rasterizer, so changing the design means
// changing the numbers in ART and running this again. Android crops an icon to a circle or squircle and only the middle ~60% is
// guaranteed to show, so everything that matters sits within 156 px of the centre.
import fs from 'fs';
import zlib from 'zlib';

const OUT = 'internal/web/static/icons/';
const C = { bgTop: [10, 75, 176], bgBot: [0, 40, 100], car: [244, 248, 255], wheel: [150, 176, 222], track: [138, 178, 240], roof: [188, 210, 245] };

// ---- the art (512 space) ----
const rr = (cx, cy, hw, hh, r, x, y) => { const dx = Math.abs(x - cx) - (hw - r), dy = Math.abs(y - cy) - (hh - r); return Math.hypot(Math.max(dx, 0), Math.max(dy, 0)) + Math.min(Math.max(dx, dy), 0) <= r; };
const ell = (cx, cy, rx, ry, x, y) => ((x - cx) / rx) ** 2 + ((y - cy) / ry) ** 2 <= 1;
function poly(pts, x, y) { let inside = false; for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) { const [xi, yi] = pts[i], [xj, yj] = pts[j]; if ((yi > y) !== (yj > y) && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) inside = !inside; } return inside; }
const bez = (p0, p1, p2, p3, n) => Array.from({ length: n + 1 }, (_, i) => { const t = i / n, u = 1 - t; return [u ** 3 * p0[0] + 3 * u * u * t * p1[0] + 3 * u * t * t * p2[0] + t ** 3 * p3[0], u ** 3 * p0[1] + 3 * u * u * t * p1[1] + 3 * u * t * t * p2[1] + t ** 3 * p3[1]]; });
const TRACKS = [bez([216, 262], [216, 310], [176, 336], [184, 392], 60), bez([296, 262], [296, 310], [256, 336], [264, 392], 60)].map((pts) => {
  const cum = [0]; for (let i = 1; i < pts.length; i++) cum.push(cum[i - 1] + Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1]));
  return { pts, cum };
});
const WIND = [[231, 140], [281, 140], [275, 171], [237, 171]], REAR = [[237, 214], [275, 214], [281, 240], [231, 240]];
const WHEELS = [[205, 150], [307, 150], [205, 232], [307, 232]];
const DASH = 18, GAP = 13, TRACK_W = 5.5;

function onTrack(x, y) {
  if (y < 250 || y > 400) return false;
  for (const { pts, cum } of TRACKS) for (let i = 1; i < pts.length; i++) {
    const [ax, ay] = pts[i - 1], [bx, by] = pts[i], vx = bx - ax, vy = by - ay, l2 = vx * vx + vy * vy;
    const t = Math.max(0, Math.min(1, ((x - ax) * vx + (y - ay) * vy) / l2));
    if (Math.hypot(x - ax - t * vx, y - ay - t * vy) <= TRACK_W) return (cum[i - 1] + t * Math.sqrt(l2)) % (DASH + GAP) < DASH;
  }
  return false;
}
// what is at a point of the art: 'window' (a cut-out of the car), 'car', 'wheel', 'track' or null
function art(x, y) {
  if (poly(WIND, x, y) || poly(REAR, x, y)) return rr(256, 191, 48, 85, 40, x, y) ? 'window' : null;
  if (WHEELS.some(([cx, cy]) => rr(cx, cy, 8, 22, 6, x, y))) return 'wheel';
  if (rr(257, 193, 29, 19, 9, x, y)) return 'roof'; // the pale roof panel between the windows: it is what makes the shape read as a car
  if (rr(256, 191, 48, 85, 40, x, y) || ell(204, 172, 5, 7, x, y) || ell(308, 172, 5, 7, x, y)) return 'car';
  return onTrack(x, y) ? 'track' : null;
}

// ---- rasterizer ----
// mode: 'color' (on the dark background), 'mono' (the art alone in one colour, windows see-through); shape: 'full' bleed or 'rounded' corners
function render(size, { mode = 'color', shape = 'full', scale = 1, tint = [255, 255, 255] } = {}) {
  const px = Buffer.alloc(size * size * 4), S = 4, k = 512 / size;
  for (let j = 0; j < size; j++) for (let i = 0; i < size; i++) {
    let r = 0, g = 0, b = 0, a = 0;
    for (let sy = 0; sy < S; sy++) for (let sx = 0; sx < S; sx++) {
      const X = (i + (sx + 0.5) / S) * k, Y = (j + (sy + 0.5) / S) * k;
      const ax = (X - 256) / scale + 256, ay = (Y - 256) / scale + 256, what = art(ax, ay);
      let c = null;
      if (mode === 'mono') { if (what && what !== 'window') c = tint; }
      else if (shape === 'rounded' && !rr(256, 256, 256, 256, 112, X, Y)) c = null;
      else {
        const t = Y / 512, bg = C.bgTop.map((v, n) => v + (C.bgBot[n] - v) * t);
        c = what === 'roof' ? C.roof : what === 'car' ? C.car : what === 'wheel' ? C.wheel : what === 'track' ? C.track : bg;
      }
      if (c) { r += c[0]; g += c[1]; b += c[2]; a++; }
    }
    const o = (j * size + i) * 4, n = S * S;
    if (a) { px[o] = r / a; px[o + 1] = g / a; px[o + 2] = b / a; px[o + 3] = Math.round((255 * a) / n); }
  }
  return px;
}

// ---- PNG ----
function png(size, px) {
  const chunk = (type, data) => { const len = Buffer.alloc(4); len.writeUInt32BE(data.length); const td = Buffer.concat([Buffer.from(type), data]); const crc = Buffer.alloc(4); crc.writeUInt32BE(zlib.crc32(td) >>> 0); return Buffer.concat([len, td, crc]); };
  const raw = Buffer.alloc((size * 4 + 1) * size);
  for (let y = 0; y < size; y++) { raw[y * (size * 4 + 1)] = 0; px.copy(raw, y * (size * 4 + 1) + 1, y * size * 4, (y + 1) * size * 4); }
  const ihdr = Buffer.alloc(13); ihdr.writeUInt32BE(size, 0); ihdr.writeUInt32BE(size, 4); ihdr[8] = 8; ihdr[9] = 6;
  return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk('IHDR', ihdr), chunk('IDAT', zlib.deflateSync(raw, { level: 9 })), chunk('IEND', Buffer.alloc(0))]);
}

// ---- the SVG (the same shapes as vectors; the tracks dashed along their curves) ----
function svg() {
  const hex = (c) => '#' + c.map((v) => v.toString(16).padStart(2, '0')).join('');
  const path = (pts) => 'M' + pts.map((p) => p.map((v) => v.toFixed(1)).join(' ')).join('L');
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="${hex(C.bgTop)}"/><stop offset="1" stop-color="${hex(C.bgBot)}"/></linearGradient></defs>`
    + `<rect width="512" height="512" rx="112" fill="url(#g)"/><g transform="translate(256 256) scale(1.4) translate(-256 -256)">`
    + `<g fill="none" stroke="${hex(C.track)}" stroke-width="${TRACK_W * 2}" stroke-dasharray="${DASH} ${GAP}">${TRACKS.map((t) => `<path d="${path(t.pts)}"/>`).join('')}</g>`
    + `<g fill="${hex(C.wheel)}">${WHEELS.map(([cx, cy]) => `<rect x="${cx - 8}" y="${cy - 22}" width="16" height="44" rx="6"/>`).join('')}</g>`
    + `<rect x="208" y="106" width="96" height="170" rx="40" fill="${hex(C.car)}"/><ellipse cx="204" cy="172" rx="5" ry="7" fill="${hex(C.car)}"/><ellipse cx="308" cy="172" rx="5" ry="7" fill="${hex(C.car)}"/>`
    + `<rect x="228" y="174" width="58" height="38" rx="9" fill="${hex(C.roof)}"/>`
    + `<path d="${path(WIND)}Z${path(REAR)}Z" fill="${hex(C.bgTop)}"/></g></svg>\n`;
}

const files = {
  'icon-192.png': png(192, render(192, { shape: 'rounded', scale: 1.4 })),
  'icon-512.png': png(512, render(512, { shape: 'rounded', scale: 1.4 })),
  'icon-maskable-512.png': png(512, render(512, { shape: 'full', scale: 1 })),
  'icon-mono-512.png': png(512, render(512, { mode: 'mono', scale: 1, tint: [0, 0, 0] })),
  'apple-touch-icon.png': png(180, render(180, { shape: 'full', scale: 1.3 })),
  'badge-96.png': png(96, render(96, { mode: 'mono', scale: 1.3, tint: [255, 255, 255] })),
  'icon.svg': Buffer.from(svg()),
};
for (const [name, data] of Object.entries(files)) { fs.writeFileSync(OUT + name, data); console.log(name, data.length, 'bytes'); }
