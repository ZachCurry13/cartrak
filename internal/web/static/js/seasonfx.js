import { isSaver } from './mode.js';

// The seasonal effect on Home: a few seconds of snow, bats, hearts and so on drifting over the page, then it stops. On the holiday
// itself it is bigger (fireworks on the Fourth, confetti on New Year's Day). Never with "reduce motion" or Data saver, and never
// in the way: the layer does not take taps.

const GLYPHS = { bats: ['🦇'], hearts: ['💕', '💗', '❤️'], shamrocks: ['☘️', '🍀'], eggs: ['🥚', '🐣', '🌷'], petals: ['🌸', '🌼', '🌷'], leaves: ['🍂', '🍁'] };
const CONFETTI = ['#f43f5e', '#f59e0b', '#22c55e', '#3b82f6', '#a855f7', '#facc15'];

const still = () => isSaver() || matchMedia('(prefers-reduced-motion: reduce)').matches;

export function playSeason(effect, big = false) {
  if (still() || document.getElementById('seasonfx')) return;
  if (effect === 'lights') { twinkle(); effect = 'snow'; }
  if (effect === 'sun') return glow();
  const c = document.createElement('canvas');
  c.id = 'seasonfx';
  c.setAttribute('aria-hidden', 'true');
  const W = (c.width = innerWidth * devicePixelRatio), H = (c.height = innerHeight * devicePixelRatio);
  document.body.append(c);
  const g = c.getContext('2d');
  const k = devicePixelRatio, n = big ? 70 : 28, life = big ? 9000 : 6000;
  const rnd = (a, b) => a + Math.random() * (b - a);
  const glyphs = GLYPHS[effect];
  const parts = [];
  const add = () => {
    if (effect === 'fireworks') {
      const x = rnd(0.15, 0.85) * W, y = rnd(0.12, 0.45) * H, col = CONFETTI[Math.floor(Math.random() * CONFETTI.length)];
      for (let i = 0; i < 26; i++) { const a = (i / 26) * Math.PI * 2, v = rnd(1.2, 2.6) * k; parts.push({ x, y, vx: Math.cos(a) * v, vy: Math.sin(a) * v, col, r: 2.2 * k, fade: 1, burst: true }); }
      return;
    }
    parts.push({ x: rnd(0, W), y: rnd(-0.2, 0) * H, vx: rnd(-0.4, 0.4) * k, vy: rnd(0.6, 1.6) * k * (effect === 'bats' ? 0.7 : 1), rot: rnd(0, 6), vr: rnd(-0.03, 0.03),
      size: rnd(14, 26) * k, glyph: glyphs ? glyphs[Math.floor(Math.random() * glyphs.length)] : null, col: CONFETTI[Math.floor(Math.random() * CONFETTI.length)], fade: 1, swing: rnd(0, 6) });
  };
  for (let i = 0; i < (effect === 'fireworks' ? 2 : n); i++) add();
  const t0 = performance.now();
  let last = t0, spawned = 0;
  const step = (now) => {
    const age = now - t0, dt = Math.min(50, now - last) / 16.7;
    last = now;
    if (effect === 'fireworks' && age < life - 2000 && age / 700 > spawned) { spawned++; add(); }
    g.clearRect(0, 0, W, H);
    const ending = age > life - 1500 ? Math.max(0, (life - age) / 1500) : 1;
    for (const p of parts) {
      p.x += (p.vx + (p.burst ? 0 : Math.sin(age / 600 + p.swing) * 0.3 * k)) * dt;
      p.y += p.vy * dt;
      if (p.burst) { p.vy += 0.03 * k * dt; p.fade -= 0.012 * dt; }
      if (p.rot != null) p.rot += p.vr * dt;
      g.globalAlpha = Math.max(0, p.fade) * ending;
      if (p.burst) { g.fillStyle = p.col; g.beginPath(); g.arc(p.x, p.y, p.r, 0, 6.3); g.fill(); continue; }
      if (effect === 'snow') { g.fillStyle = '#fff'; g.beginPath(); g.arc(p.x, p.y, p.size / 8, 0, 6.3); g.fill(); } else if (effect === 'confetti') {
        g.save(); g.translate(p.x, p.y); g.rotate(p.rot); g.fillStyle = p.col; g.fillRect(-p.size / 4, -p.size / 8, p.size / 2, p.size / 4); g.restore();
      } else { g.save(); g.translate(p.x, p.y); g.rotate(p.rot); g.font = `${p.size}px serif`; g.textAlign = 'center'; g.fillText(p.glyph, 0, 0); g.restore(); }
      if (!p.burst && p.y > H + 40) { p.y = -30; p.x = rnd(0, W); }
    }
    if (age < life && c.isConnected) requestAnimationFrame(step); else c.remove();
  };
  requestAnimationFrame(step);
  addEventListener('hashchange', () => c.remove(), { once: true });
}

// Christmas: a string of lights along the top of the page, twinkling for a while.
function twinkle() {
  const head = document.getElementById('top');
  if (!head || head.querySelector('.xlights')) return;
  const row = document.createElement('div');
  row.className = 'xlights';
  row.setAttribute('aria-hidden', 'true');
  for (let i = 0; i < 18; i++) { const b = document.createElement('i'); b.style.animationDelay = `${(i % 6) * 0.25}s`; row.append(b); }
  head.append(row);
}

// Summer: a warm glow that rises and fades.
function glow() {
  const d = document.createElement('div');
  d.id = 'seasonfx';
  d.className = 'sunglow';
  d.setAttribute('aria-hidden', 'true');
  document.body.append(d);
  setTimeout(() => d.remove(), 5000);
}
