const NS = 'http://www.w3.org/2000/svg';
const el = (tag, attrs, text) => {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v);
  if (text != null) e.textContent = text;
  return e;
};

// The battery as a ring with the percentage inside. It turns amber at 20% or less and red at 10% or less.
// Use ring.el in the page and call ring.set(percent, "194 mi range") whenever a new reading arrives.
export function batteryRing() {
  const C = 2 * Math.PI * 62;
  const trk = el('circle', { class: 'trk', cx: 75, cy: 75, r: 62, fill: 'none', 'stroke-width': 12 });
  const arc = el('circle', { class: 'arc', cx: 75, cy: 75, r: 62, fill: 'none', 'stroke-width': 12, 'stroke-linecap': 'round', 'stroke-dasharray': `0 ${C}`, transform: 'rotate(-90 75 75)' });
  const big = el('text', { x: 75, y: 82, 'text-anchor': 'middle', 'font-size': 34, 'font-weight': 600 }, '—');
  const sub = el('text', { class: 'sub', x: 75, y: 102, 'text-anchor': 'middle', 'font-size': 12 }, '');
  const svg = el('svg', { viewBox: '0 0 150 150', class: 'ring', role: 'img', 'aria-label': 'Battery' }, null);
  svg.append(trk, arc, big, sub);
  return {
    el: svg,
    set(pct, subText) {
      const p = pct == null ? 0 : Math.max(0, Math.min(100, pct));
      arc.setAttribute('stroke-dasharray', `${Math.max(0.01, (C * p) / 100)} ${C}`);
      big.textContent = pct == null ? '—' : `${Math.round(p)}%`;
      sub.textContent = subText || '';
      svg.classList.toggle('low', pct != null && p <= 20 && p > 10);
      svg.classList.toggle('crit', pct != null && p <= 10);
      svg.setAttribute('aria-label', pct == null ? 'Battery unknown' : `Battery ${Math.round(p)} percent${subText ? `, ${subText}` : ''}`);
    },
  };
}
