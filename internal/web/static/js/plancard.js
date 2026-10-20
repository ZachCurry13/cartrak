import { h, dist, num } from './ui.js';
import { api } from './api.js';

const addDays = (iso, n) => { const d = new Date(`${iso}T12:00:00`); d.setDate(d.getDate() + n); return d; };
const weekday = (d) => d.toLocaleDateString([], { weekday: 'long' });

// What the next week looks like if you keep driving as you have: how far, how much battery, and when to plug in.
export async function planCard() {
  const r = await api('/api/plan-ahead').catch(() => null);
  if (!r || !r.ready) return null;
  const f = r.forecast, days = f.days, max = Math.max(...days.map((d) => d.km), 1);
  const first = days.find((d) => d.endPct < f.reservePct);
  let verdict;
  if (!first) verdict = `Enough for the week without plugging in: about ${num(f.endPct, 0, '%')} left after ${weekday(addDays(days[days.length - 1].date, 0))}.`;
  else if (first === days[0]) verdict = `You would be below ${num(f.reservePct, 0, '%')} by the end of tomorrow, so plug in tonight.`;
  else verdict = `You would drop below ${num(f.reservePct, 0, '%')} on ${weekday(addDays(first.date, 0))}, so plug in by ${weekday(addDays(first.date, -1))}.`;
  const bars = h('div', { class: 'planbars', role: 'img', 'aria-label': 'Expected distance for each of the next seven days' },
    ...days.map((d) => h('div', { class: 'planday', title: `${weekday(addDays(d.date, 0))}: about ${dist(d.km, 0)}` },
      h('i', { style: `height:${Math.max(4, Math.round((d.km / max) * 46))}px` }), h('span', {}, weekday(addDays(d.date, 0)).slice(0, 1)))));
  return h('section', { class: 'card' }, h('h2', {}, 'Plan ahead'),
    h('p', {}, h('b', {}, `About ${dist(f.totalKm, 0)} and ${num(f.totalKwh, 0, ' kWh')}`), ' over the next 7 days.'),
    bars, h('p', {}, verdict),
    h('p', { class: 'muted' }, `An estimate from your last ${Math.round(f.historyDays / 7)} weeks of driving, with no charging. It cannot know about a trip you have not taken yet.`));
}
