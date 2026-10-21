import { h, emptyState, dist, num } from '../ui.js';
import { eventRow, money, span } from '../eventrow.js';
import { api } from '../api.js';

const KINDS = [['all', 'All'], ['charging', 'Charging'], ['driving', 'Driving'], ['parked', 'Parked'], ['remote', 'Remote starts']];
const dayKey = (ts, tz) => new Date(ts * 1000).toLocaleDateString('en-CA', { timeZone: tz });
const dayHead = (ts, tz) => new Date(ts * 1000).toLocaleDateString([], { weekday: 'short', day: 'numeric', timeZone: tz });

// Everything that happened in a month, newest first: charging, driving and parked time in one feed.
export default async function history(root) {
  let month = '', kind = 'all';
  const box = h('div'), seg = h('div', { class: 'seg' });
  root.append(h('p', {}, h('a', { href: '#/trips' }, '← Trips')), h('div', { class: 'page-title' }, h('h2', {}, 'History')), seg, box);

  async function load() {
    const d = await api(`/api/history${month ? `?month=${month}` : ''}`);
    month = d.month;
    const [y, m] = month.split('-').map(Number);
    const title = new Date(y, m - 1, 1).toLocaleDateString([], { month: 'long', year: 'numeric' });
    const shift = (n) => { const t = new Date(y, m - 1 + n, 1); month = `${t.getFullYear()}-${String(t.getMonth() + 1).padStart(2, '0')}`; load(); };
    const nd = new Date();
    const nowMonth = `${nd.getFullYear()}-${String(nd.getMonth() + 1).padStart(2, '0')}`; // this device's month, not UTC's
    seg.replaceChildren(...KINDS.map(([k, t]) => h('button', { class: kind === k ? 'on' : '', onclick: () => { kind = k; load(); } }, t)));
    const t = d.totals;
    const evs = d.events.filter((e) => kind === 'all' || e.kind === kind);
    const days = [];
    for (const e of evs) {
      const k = dayKey(e.start, d.tz);
      if (!days.length || days[days.length - 1].k !== k) days.push({ k, ts: e.start, list: [] });
      days[days.length - 1].list.push(e);
    }
    box.replaceChildren(...[
      h('div', { class: 'hmonth' }, h('button', { class: 'sec', 'aria-label': 'Earlier month', onclick: () => shift(-1) }, '‹'), h('h3', {}, title),
        h('button', { class: 'sec', 'aria-label': 'Later month', disabled: month >= nowMonth, onclick: () => shift(1) }, '›')),
      h('div', { class: 'htotals' }, h('span', {}, '⚡ ', h('b', {}, num(t.kwh, 0, ' kWh'))), h('span', {}, '↗ ', h('b', {}, dist(t.km, 0))), h('span', {}, 'P ', h('b', {}, span(t.parkedSec))),
        t.cost > 0 ? h('span', {}, h('b', {}, money(d.currency, t.cost))) : null),
      ...(days.length ? days.flatMap((g) => [h('div', { class: 'hday' }, dayHead(g.ts, d.tz), h('a', { class: 'hdaylink', href: `#/day/${g.k}` }, 'Map of this day')), ...g.list.map((e) => eventRow(e, d.tz, d.currency))])
        : [emptyState('Nothing here', 'No trips, charging or parked time were recorded for this selection.')])].filter(Boolean));
  }
  await load();
}
