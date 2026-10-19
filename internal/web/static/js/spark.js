import { h } from './ui.js';
import { lineChart } from './chart.js';

const day = (ts) => new Date(ts * 1000).toLocaleDateString([], { month: 'short', day: 'numeric' });

// A daily trend you can read: touch and drag along it to see the value on any day, with the range for the period.
export function spark(series, format) {
  const days = series.days;
  if (days.length < 2) return h('div', { class: 'trend' }, h('div', { class: 'muted' }, series.label), h('p', { class: 'muted' }, 'Not enough days of readings yet.'));
  const lo = Math.min(...days.map((d) => d.min)), hi = Math.max(...days.map((d) => d.max));
  const chart = lineChart({
    points: days.map((d) => ({ x: d.ts, y: d.avg })),
    xFmt: (x) => day(x),
    yFmt: (v) => format(series.kind, v),
    yMin: Math.min(...days.map((d) => d.avg)) - (hi - lo || 1) * 0.3,
    xTicks: [days[0].ts, days[Math.floor(days.length / 2)].ts, days[days.length - 1].ts],
    summary: `${day(days[0].ts)} to ${day(days[days.length - 1].ts)} · low ${format(series.kind, lo)} · high ${format(series.kind, hi)}`,
    height: 170,
  });
  return h('div', { class: 'trend' }, h('div', { class: 'muted' }, series.label), chart);
}
