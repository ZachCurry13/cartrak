import { h, dist, num } from './ui.js';

// Day breaks for the Trips list and the Charging sessions (kind "charge"): a heading each time the day changes (newest first), with the day's trips and miles. The day is the
// one the trip happened in, in the time zone it was recorded in.
const dayOf = (t) => {
  try { return new Date(t.startTs * 1000).toLocaleDateString('en-CA', { timeZone: t.tz || undefined }); } catch (e) { return new Date(t.startTs * 1000).toLocaleDateString('en-CA'); }
};
const label = (t) => {
  const o = { weekday: 'long', month: 'short', day: 'numeric' };
  try { return new Date(t.startTs * 1000).toLocaleDateString([], { ...o, timeZone: t.tz || undefined }); } catch (e) { return new Date(t.startTs * 1000).toLocaleDateString([], o); }
};

export function withDayBreaks(trips, row, kind = 'trip') {
  const days = new Map();
  for (const t of trips) {
    const k = dayOf(t), d = days.get(k) || { n: 0, km: 0 };
    d.n += 1; d.km += kind === 'charge' ? t.addedKwh || 0 : t.distanceKm || 0;
    days.set(k, d);
  }
  const out = [];
  let prev = '';
  for (const t of trips) {
    const k = dayOf(t);
    if (k !== prev) {
      const d = days.get(k);
      out.push(h('h3', { class: 'daybreak', style: 'margin:18px 0 6px;display:flex;gap:8px;align-items:baseline;flex-wrap:wrap' }, label(t),
        h('span', { class: 'muted', style: 'font-weight:400;font-size:.85em' }, `${d.n} ${kind === 'charge' ? 'charge' : 'trip'}${d.n === 1 ? '' : 's'} · ${kind === 'charge' ? num(d.km, 1, ' kWh') : dist(d.km, 1)}`)));
      prev = k;
    }
    out.push(row(t));
  }
  return out;
}
