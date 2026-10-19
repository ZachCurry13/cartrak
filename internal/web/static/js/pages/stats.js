import { h, emptyState, dist, num, isMetric } from '../ui.js';
import { api } from '../api.js';
import { tapBars } from '../chart.js';
import { placeDonut } from '../placedonut.js';

const PERIODS = [['day', 'Day'], ['month', 'Month'], ['year', 'Year']];
const KM = 1.609344;
const money = (cur, v) => `${cur}${v.toFixed(v >= 100 ? 0 : 2)}`;
const hm = (sec) => (sec < 3600 ? `${Math.round(sec / 60)} min` : `${(sec / 3600).toFixed(1)} h`);

// What each tab measures and how its numbers read, in the device's units.
function metrics(cur) {
  const metric = isMetric();
  return [
    { key: 'km', label: isMetric() ? 'Distance' : 'Dist.', value: (b) => (metric ? b.km : b.km / KM), text: (b) => dist(b.km, 0) },
    { key: 'cost', label: 'Cost', value: (b) => b.cost, text: (b) => money(cur, b.cost) },
    { key: 'kwh', label: 'kWh', value: (b) => b.kwh, text: (b) => num(b.kwh, 0, ' kWh') },
    { key: 'time', label: 'Time', value: (b) => b.drivingSec, text: (b) => `${hm(b.drivingSec)} driving` }];
}

// Distance, cost, energy and driving time as bars by day, month or year, and where the charging happened.
export default async function stats(root) {
  let period = 'month', at = '', metric = 'km';
  const box = h('div'), segP = h('div', { class: 'seg' }), segM = h('div', { class: 'seg' });
  root.append(h('p', {}, h('a', { href: '#/charging' }, '← Charging')), h('div', { class: 'page-title' }, h('h2', {}, 'Statistics')), segM, segP, box);

  async function load() {
    const d = await api(`/api/stats?period=${period}${at && period !== 'year' ? `&at=${at}` : ''}`);
    const s = d.stats;
    const ms = metrics(d.currency);
    const cur = ms.find((m) => m.key === metric);
    const start = new Date(s.from * 1000);
    const title = period === 'day' ? start.toLocaleDateString([], { month: 'long', year: 'numeric' }) : period === 'month' ? String(start.getFullYear()) : 'All years';
    const shift = (n) => {
      const t = new Date(start.getFullYear(), start.getMonth() + (period === 'day' ? n : 12 * n), 1);
      at = period === 'day' ? `${t.getFullYear()}-${String(t.getMonth() + 1).padStart(2, '0')}` : String(t.getFullYear());
      load();
    };
    segM.replaceChildren(...ms.map((m) => h('button', { class: metric === m.key ? 'on' : '', onclick: () => { metric = m.key; load(); } }, m.label)));
    segP.replaceChildren(...PERIODS.map(([k, t]) => h('button', { class: period === k ? 'on' : '', onclick: () => { period = k; at = ''; load(); } }, t)));
    const total = s.buckets.reduce((a, b) => ({ km: a.km + b.km, cost: a.cost + b.cost, kwh: a.kwh + b.kwh, drivingSec: a.drivingSec + b.drivingSec }), { km: 0, cost: 0, kwh: 0, drivingSec: 0 });
    const active = s.buckets.filter((b) => b.km > 0 || b.kwh > 0);
    const items = s.buckets.map((b) => ({ label: b.label, value: cur.value(b), text: cur.text(b) }));
    const empty = !active.length;
    box.replaceChildren(...[
      period !== 'year' ? h('div', { class: 'hmonth' }, h('button', { class: 'sec', 'aria-label': 'Earlier', onclick: () => shift(-1) }, '‹'), h('h3', {}, title),
        h('button', { class: 'sec', 'aria-label': 'Later', onclick: () => shift(1) }, '›')) : null,
      h('section', { class: 'card' },
        h('div', { class: 'hbig' }, h('b', {}, cur.text(total)), h('span', { class: 'muted' }, ` total${active.length ? ` · about ${cur.text(Object.fromEntries(Object.entries(total).map(([k, v]) => [k, v / active.length])))} per ${period === 'day' ? 'active day' : period === 'month' ? 'active month' : 'year'}` : ''}`)),
        empty ? emptyState('Nothing recorded', 'No trips or charging in this period.') : tapBars(items, { height: 150, summary: `Tap a bar to read it.` })),
      placeDonut(s.kinds, d.currency)].filter(Boolean));
  }
  await load();
}
