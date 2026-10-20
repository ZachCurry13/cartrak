import { h, emptyState, dist, num } from '../ui.js';
import { api } from '../api.js';

const KINDS = [['all', 'All'], ['charging', 'Charging'], ['driving', 'Driving'], ['parked', 'Parked']];
const GLYPH = { charging: '⚡', driving: '↗', parked: 'P' };
const money = (cur, v) => `${cur}${v.toFixed(2)}`;
const WHERE = { home: 'Home', public: 'Public' };

function span(sec) {
  const m = Math.round(sec / 60);
  if (m < 60) return `${m} min`;
  if (m < 2880) return `${Math.floor(m / 60)} h ${m % 60} min`;
  return `${Math.floor(m / 1440)} d ${Math.floor((m % 1440) / 60)} h`;
}

const clock = (ts, tz) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', timeZone: tz });
const dayKey = (ts, tz) => new Date(ts * 1000).toLocaleDateString('en-CA', { timeZone: tz });
const dayHead = (ts, tz) => new Date(ts * 1000).toLocaleDateString([], { weekday: 'short', day: 'numeric', timeZone: tz });

// A thin bar of the battery level: what was left in a quiet colour and the part that changed in the event's own colour.
function levelBar(e) {
  const a = e.startSoc, b = e.endSoc;
  if (a == null && b == null) return null;
  const lo = Math.min(a == null ? b : a, b == null ? a : b), hi = Math.max(a == null ? b : a, b == null ? a : b);
  return h('div', { class: 'hbar', role: 'img', 'aria-label': `Battery ${Math.round(a == null ? b : a)}% to ${Math.round(b == null ? a : b)}%` },
    h('i', { class: 'base', style: `width:${lo}%` }), h('i', { class: `chg ${e.kind}`, style: `width:${Math.max(1, hi - lo)}%` }));
}

// One row of the feed. stopNumber, when given, is the stop's number on the day map and replaces the little icon.
export function eventRow(e, tz, cur, stopNumber) {
  const now = Date.now() / 1000;
  const end = e.end == null ? now : e.end;
  const delta = e.startSoc != null && e.endSoc != null ? Math.round(e.endSoc - e.startSoc) : null;
  const place = e.kind === 'driving' ? [e.fromPlace, e.place].filter(Boolean).join(' → ') : e.place || (e.kind === 'charging' ? WHERE[e.where] || 'Charging' : '');
  const right = e.kind === 'charging' ? (e.cost != null ? money(cur, e.cost) : e.addedKwh != null ? num(e.addedKwh, 1, ' kWh') : '') : e.kind === 'driving' ? dist(e.distanceKm, 1) : '';
  const sub = e.kind === 'charging' && e.cost != null && e.addedKwh != null ? num(e.addedKwh, 1, ' kWh') : '';
  const body = [
    h('div', { class: 'hrow1' }, h('span', { class: `hico ${e.kind}${stopNumber != null ? ' num' : ''}`, 'aria-hidden': 'true' }, stopNumber != null ? stopNumber : GLYPH[e.kind]),
      h('b', {}, `${clock(e.start, tz)} – ${e.end == null ? 'Now' : clock(e.end, tz)}`), h('span', { class: 'muted grow' }, span(end - e.start)),
      delta != null ? h('span', { class: 'hdelta' }, `${delta > 0 ? '+' : ''}${delta}%`) : null),
    levelBar(e),
    h('div', { class: 'hrow2' }, h('span', { class: 'hplace' }, place || (e.kind === 'parked' ? 'Parked' : 'Drive')), h('span', { class: 'hright' }, right, sub ? h('div', { class: 'muted' }, sub) : null))];
  const href = e.kind === 'driving' ? `#/trip/${e.id}` : e.kind === 'charging' ? '#/charging' : null;
  return h(href ? 'a' : 'div', { class: `hcard ${e.kind}`, ...(href ? { href } : {}) }, ...body.filter(Boolean));
}

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
