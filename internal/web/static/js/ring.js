const NS = 'http://www.w3.org/2000/svg';
const el = (tag, attrs, text) => {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v);
  if (text != null) e.textContent = text;
  return e;
};

// The battery (or the fuel tank) as a half-circle gauge with the percentage in the bowl. It turns amber at 20% or less and red at
// 10% or less. Use ring.el in the page and call ring.set(percent, "194 mi range") whenever a new reading arrives. A gas or hybrid
// car shows its fuel in the same gauge (what = 'Fuel').
export function batteryRing(what = 'Battery') {
  const d = 'M13 80 A62 62 0 0 1 137 80'; // the arc over the top, from 9 o'clock to 3 o'clock; pathLength 100 makes a percent a length
  const trk = el('path', { class: 'trk', d, fill: 'none', 'stroke-width': 12, 'stroke-linecap': 'round', pathLength: 100 });
  const arc = el('path', { class: 'arc', d, fill: 'none', 'stroke-width': 12, 'stroke-linecap': 'round', pathLength: 100, 'stroke-dasharray': '0.01 100' });
  const big = el('text', { x: 75, y: 66, 'text-anchor': 'middle', 'font-size': 34, 'font-weight': 600 }, '—');
  const sub = el('text', { class: 'sub', x: 75, y: 88, 'text-anchor': 'middle', 'font-size': 12 }, '');
  const svg = el('svg', { viewBox: '0 0 150 96', class: 'ring', role: 'img', 'aria-label': what }, null);
  // in a seasonal theme the arc runs between the season's two colors (css/look.css picks it up only then)
  const defs = el('defs', {}, null), grad = el('linearGradient', { id: 'seasonring', x1: 0, y1: 0, x2: 1, y2: 0 }, null);
  grad.append(el('stop', { offset: 0, style: 'stop-color:var(--season1, var(--accent))' }, null), el('stop', { offset: 1, style: 'stop-color:var(--season2, var(--accent))' }, null));
  defs.append(grad);
  svg.append(defs, trk, arc, big, sub);
  const gauge = {
    el: svg,
    what, // what it shows; a page may change it (Live, when the car turns out to have a tank and no battery level)
    set(pct, subText) {
      const p = pct == null ? 0 : Math.max(0, Math.min(100, pct));
      arc.setAttribute('stroke-dasharray', `${Math.max(0.01, p)} 100`);
      big.textContent = pct == null ? '—' : `${Math.round(p)}%`;
      sub.textContent = subText || '';
      svg.classList.toggle('low', pct != null && p <= 20 && p > 10);
      svg.classList.toggle('crit', pct != null && p <= 10);
      svg.setAttribute('aria-label', pct == null ? `${gauge.what} unknown` : `${gauge.what} ${Math.round(p)} percent${subText ? `, ${subText}` : ''}`);
    },
  };
  return gauge;
}
