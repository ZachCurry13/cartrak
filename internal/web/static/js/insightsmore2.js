import { h, num } from './ui.js';
import { api } from './api.js';
import { lineChart } from './chart.js';
import { valueText, periodTitle } from './funwords.js';
import { windowPicker } from './insightsmore.js';

// The detailed Insights pages for records and the monthly recap, and for the battery (#/analytics/records, #/analytics/battery).

const card = (title, ...kids) => h('section', { class: 'card' }, h('h2', {}, title), ...kids);
const ICON = { longest: '🛣️', efficient: '🍃', topSpeed: '🏁', coldest: '🥶', hottest: '🥵', peakKw: '⚡', bigCharge: '🔋', cheapest: '💲', bigDay: '📅', streak: '🔥', bestMpg: '🍃', longestTank: '⛽', biggestFill: '🛢️', cheapestGas: '💲' };
const day = (ts, date) => (date ? new Date(`${date}T12:00`) : new Date(ts * 1000)).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });

function recapCard(period, onMove) {
  const box = h('section', { class: 'card recap' });
  api(`/api/analytics/recap?month=${period}`).then((r) => {
    const cur = r.currency || '$';
    const up = r.prevKm > 0 ? Math.round(((r.km - r.prevKm) / r.prevKm) * 100) : null;
    box.replaceChildren(h('div', { class: 'row', style: 'border:0;padding:0' }, h('h2', { class: 'grow', style: 'margin:0' }, periodTitle(r.period)),
      h('button', { class: 'sec sm', 'aria-label': 'Month before', onclick: () => onMove(-1) }, '‹'), h('button', { class: 'sec sm', 'aria-label': 'Month after', onclick: () => onMove(1) }, '›')),
    r.trips ? h('div', { class: 'bigfig', style: 'margin-top:12px' },
      h('div', {}, h('b', {}, valueText('km', r.km)), h('span', {}, `${r.trips} trips${up != null ? ` · ${up >= 0 ? '▲' : '▼'} ${Math.abs(up)}%` : ''}`)),
      r.fuel && !(r.kwhCharged > 0)
        ? h('div', {}, h('b', {}, r.fuel.kmPerGal ? valueText('kmPerGal', r.fuel.kmPerGal) : '—'), h('span', {}, r.fuel.gal ? `${valueText('gal', r.fuel.gal)} burned` : 'fuel economy'))
        : h('div', {}, h('b', {}, r.kmPerKwh ? valueText('kmPerKwh', r.kmPerKwh) : '—'), h('span', {}, `${num(r.kwhUsed, 0, ' kWh')} used`)),
      r.fuel && !(r.kwhCharged > 0)
        ? h('div', {}, h('b', {}, `${cur}${r.fuel.spent.toFixed(2)}`), h('span', {}, `${r.fuel.fills} fill-up${r.fuel.fills === 1 ? '' : 's'}${r.fuel.avgPrice > 0 ? ` · ${valueText('pricePerGal', r.fuel.avgPrice, cur)}` : ''}`))
        : h('div', {}, h('b', {}, `${cur}${r.cost.toFixed(2)}`), h('span', {}, `${r.sessions} charges · ${Math.round(r.homeShare * 100)}% at home`)),
      r.saved != null ? h('div', {}, h('b', {}, `${cur}${Math.round(r.saved)} saved`), h('span', {}, `against a gas car${r.kgCo2Saved > 0 ? ` · ${Math.round(r.kgCo2Saved)} kg CO₂` : ''}`)) : h('div', {}, h('b', {}, `${r.daysDriven} days`), h('span', {}, 'with driving'))) : h('p', { class: 'muted' }, 'No trips this month.'),
    r.trips ? h('p', { class: 'seg', style: 'margin-top:12px' }, h('a', { class: 'btn', href: `#/wrapped/${r.period}` }, 'Play it as Wrapped ▶')) : null);
  });
  return box;
}

export async function records(body) {
  let at = new Date();
  const key = () => `${at.getFullYear()}-${String(at.getMonth() + 1).padStart(2, '0')}`;
  const slot = h('div');
  const move = (n) => { const next = new Date(at.getFullYear(), at.getMonth() + n, 1); if (next <= new Date()) { at = next; slot.replaceChildren(recapCard(key(), move)); } };
  slot.append(recapCard(key(), move));
  const d = await api('/api/analytics/records');
  const rows = d.records.map((x) => h(x.tripId ? 'a' : 'div', { class: 'rec', ...(x.tripId ? { href: `#/trip/${x.tripId}` } : x.chargeId ? { href: '#/charging' } : {}) },
    h('span', { class: 'ri', 'aria-hidden': 'true' }, ICON[x.key] || '⭐'), h('span', { class: 'grow' }, h('b', {}, valueText(x.kind, x.value, d.currency)), h('span', { class: 'muted' }, `${x.title} · ${x.key === 'streak' ? `ended ${day(0, x.date)}` : day(x.ts, x.date)}`))));
  body.append(slot, rows.length ? card('Personal bests', ...rows) : h('p', { class: 'muted' }, 'Records appear after a few trips.'));
}

const BANDS = ['bad', 'warn', 'ok', 'ok', 'personal'];

export function battery(body) {
  const out = h('div');
  windowPicker(body, async (days) => {
    const d = await api(`/api/analytics/battery?days=${days}`);
    const ago = (ts) => `${Math.round((Date.now() / 1000 - ts) / 86400)} days ago`;
    const fromDay = (s) => new Date(`${s}T12:00`).getTime() / 1000;
    const level = d.days.map((x) => ({ x: fromDay(x.date), y: x.end }));
    const cap = d.capacity || [], v12 = d.volts12 || [];
    out.replaceChildren(
      level.length >= 2 ? card('Battery level, day by day', lineChart({ points: level, xFmt: ago, yFmt: (y) => `${Math.round(y)}%`, summary: 'The level at the end of each day', yTitle: '%', yCap: 100 })) : null,
      d.bands && d.bands.length ? card('Where the battery spends its time',
        h('div', { class: 'socbar' }, ...d.bands.map((b, i) => h('i', { style: `width:${b.share * 100}%;background:var(--${BANDS[i]})`, title: `${b.from}–${b.to}%` }))),
        ...d.bands.map((b, i) => h('div', { class: 'row' }, h('span', { class: 'grow' }, h('i', { class: 'sw', style: `background:var(--${BANDS[i]})` }), `${b.from}–${b.to}%`), h('b', {}, `${Math.round(b.share * 100)}%`))),
        h('p', { class: 'muted' }, `On average ${Math.round(d.avgPct)}%. ${Math.round(d.highShare * 100)}% of the time at 90% or more and ${Math.round(d.lowShare * 100)}% under 20%. Most of the time in the middle is easiest on the battery.`)) : null,
      cap.length >= 2 ? card('Usable capacity', lineChart({ points: cap.map((x) => ({ x: x.ts, y: x.avg })), xFmt: ago, yFmt: (y, axis) => (axis ? y.toFixed(1) : `${y.toFixed(1)} kWh`), yMin: Math.floor(Math.min(...cap.map((x) => x.avg)) - 1),
        summary: `${cap[0].avg.toFixed(1)} → ${cap[cap.length - 1].avg.toFixed(1)} kWh in this stretch`, yTitle: 'kWh' })) : null,
      d.remote && d.remote.count ? card('Remote starts', h('div', { class: 'bigfig' }, h('div', {}, h('b', {}, String(d.remote.count)), h('span', {}, 'remote starts')),
        h('div', {}, h('b', {}, num(d.remote.kwh, 1, ' kWh')), h('span', {}, 'taken from the battery')), h('div', {}, h('b', {}, `${Math.round(d.remote.minutes)} min`), h('span', {}, 'of the car running'))),
        h('p', { class: 'muted' }, 'Warming or cooling the car while it is plugged in takes the energy from the wall instead of the battery.')) : null,
      v12.length >= 2 ? card('12 V battery', lineChart({ points: v12.map((x) => ({ x: x.ts, y: x.avg })), xFmt: ago, yFmt: (y, axis) => (axis ? y.toFixed(1) : `${y.toFixed(2)} V`), yMin: 11.5, summary: 'Daily average; the car charges it while it runs', yTitle: 'V' })) : null);
    if (!out.children.length) out.append(h('p', { class: 'muted' }, 'Not enough readings yet.'));
  });
  body.append(out);
}
