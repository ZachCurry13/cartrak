import { h, ago } from './ui.js';
import { timelineChart } from './timeline.js';

const stampDay = (ts) => new Date(ts * 1000).toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' });
const stampTime = (ts) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
const minuteOf = (ts) => { const d = new Date(ts * 1000); return d.getHours() * 60 + d.getMinutes(); };
const clockOf = (min) => { const d = new Date(); d.setHours(Math.floor(min / 60), min % 60, 0, 0); return d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }); };

// The top of Car data: pick a day with the arrows, drag the slider along the day (or tap the chart) to see the car at that moment.
// d is the reading shown, line the day's timeline, from and to the day's ends; load(ts) shows the car at a time (0 for the latest).
export function carDataHead(d, line, { from, to, fuel, load }) {
  const nowTs = Math.floor(Date.now() / 1000);
  const slider = h('input', { type: 'range', min: 0, max: 1439, step: 5, value: minuteOf(d.ts), 'aria-label': 'Time of day', style: 'width:100%' });
  const shown = h('b', { style: 'font-size:var(--t-title-l,1.5rem)' }, stampTime(d.ts));
  slider.addEventListener('input', () => { shown.textContent = clockOf(Number(slider.value)); }); // while dragging, only the readout moves
  slider.addEventListener('change', () => load(from + Number(slider.value) * 60)); // on release the car at that time is shown
  const dayBtn = (label, text, ts, off) => h('button', { class: 'sec', type: 'button', 'aria-label': label, disabled: off, onclick: () => load(ts), style: 'min-width:44px' }, text);
  return h('section', { class: 'card' },
    h('div', { class: 'row', style: 'gap:8px;align-items:center;flex-wrap:nowrap' },
      dayBtn('Previous day', '‹', from - 1, d.first && from <= d.first), h('b', { class: 'grow', style: 'text-align:center' }, stampDay(d.ts)), dayBtn('Next day', '›', Math.min(to + 1, nowTs), to >= nowTs)),
    h('div', { style: 'text-align:center;margin:var(--sp-2) 0' }, shown, ' ', d.latest ? h('span', { class: 'chip ok' }, 'latest reading') : h('span', { class: 'chip work' }, ago(d.ts))),
    line && line.points.length > 1 ? timelineChart(line.points, from, to, d.ts, (t) => load(t), fuel) : h('p', { class: 'muted' }, 'No readings on this day.'),
    h('div', { style: 'margin-top:var(--sp-2)' }, slider, h('div', { class: 'row muted', style: 'justify-content:space-between' }, h('span', {}, '12 AM'), h('span', {}, 'Noon'), h('span', {}, '11 PM'))),
    h('p', { class: 'seg' }, h('button', { class: 'sec', type: 'button', disabled: !d.prev, onclick: () => load(d.prev) }, '◀ Earlier reading'), h('button', { class: 'sec', type: 'button', disabled: !d.next, onclick: () => load(d.next) }, 'Later reading ▶'),
      d.latest ? null : h('button', { type: 'button', onclick: () => load(0) }, 'Latest')),
    h('p', { class: 'muted' }, 'Drag the slider or tap the chart to see the car at another time. It shows the last reading CarTrak had by then.'));
}
