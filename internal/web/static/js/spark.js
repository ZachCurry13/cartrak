import { h } from './ui.js';

const NS = 'http://www.w3.org/2000/svg';
const day = (ts) => new Date(ts * 1000).toLocaleDateString([], { month: 'short', day: 'numeric' });

// A small trend line of one value per day, with its range underneath. No chart library.
export function spark(series, format) {
  const days = series.days;
  const lo = Math.min(...days.map((d) => d.min));
  const hi = Math.max(...days.map((d) => d.max));
  const span = hi - lo || 1;
  const w = 300, ht = 56, pad = 4;
  const x = (i) => pad + (i * (w - pad * 2)) / Math.max(days.length - 1, 1);
  const y = (v) => ht - pad - ((v - lo) / span) * (ht - pad * 2);
  const svg = document.createElementNS(NS, 'svg');
  svg.setAttribute('viewBox', `0 0 ${w} ${ht}`);
  svg.setAttribute('class', 'spark');
  svg.setAttribute('role', 'img');
  svg.setAttribute('aria-label', series.label);
  const line = document.createElementNS(NS, 'polyline');
  line.setAttribute('points', days.map((d, i) => `${x(i).toFixed(1)},${y(d.avg).toFixed(1)}`).join(' '));
  line.setAttribute('fill', 'none');
  line.setAttribute('stroke', 'currentColor');
  line.setAttribute('stroke-width', '2.5');
  line.setAttribute('stroke-linejoin', 'round');
  svg.append(line);
  return h('div', { class: 'trend' }, h('div', { class: 'muted' }, series.label), svg,
    h('div', { class: 'muted' }, `${day(days[0].ts)} to ${day(days[days.length - 1].ts)} · low ${format(series.kind, lo)} · high ${format(series.kind, hi)}`));
}
