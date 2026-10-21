import { h, stat, dist, num, speed, pct, emptyState, isMetric } from '../ui.js';
import { api } from '../api.js';
import { createMap, mapConfig } from '../map.js';
import { effSpan } from '../energy.js';
import { eventRow } from '../eventrow.js';
import { roadDay } from '../roadday.js';
import { expandable } from '../daydetail.js';
import { shareCard } from '../sharecard.js';

const COLORS = ['#1b5fc4', '#d97706', '#7c3aed', '#0d9488', '#be185d', '#475569'];
const STOP_COLOR = { charging: '#16a34a', parked: '#64748b' };
const money = (cur, v) => `${cur}${v.toFixed(2)}`;
const hm = (sec) => (sec < 3600 ? `${Math.round(sec / 60)} min` : `${Math.floor(sec / 3600)} h ${Math.round((sec % 3600) / 60)} min`);
const shift = (date, n) => { const d = new Date(`${date}T12:00:00`); d.setDate(d.getDate() + n); return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`; };

const plural = (n, w, ws) => `${n} ${n === 1 ? w : ws || w + 's'}`;
const clock = (ts, tz) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', timeZone: tz });

// The day in words: how long was driven and stood still, how many stops, what was charged and what it cost.
function summaryCard(x, road) {
  const atHome = x.events.filter((v) => v.kind === 'charging' && v.where === 'home' && !road.hidden.has(v)).length;
  const s = x.summary, t = x.totals, e = x.energy, cur = x.currency;
  if (!s.trips && !s.sessions) return null;
  const lines = [];
  if (s.trips) lines.push(`You drove ${hm(t.drivingSec)} over ${plural(s.trips, 'trip')}, ${dist(t.km, 0)}, at an average of ${speed(s.avgMovingKph)} (top speed ${speed(s.topKph)}).`);
  const remote = x.events.filter((v) => v.kind === 'remote');
  if (remote.length) { const kwh = remote.reduce((n, v) => n + (v.energyKwh || 0), 0); lines.push(`The car was remote started ${remote.length === 1 ? 'once' : plural(remote.length, 'time')}${kwh >= 0.1 ? `, using ${kwh.toFixed(1)} kWh` : ''}.`); }
  if (s.stops) lines.push(`You stopped ${plural(s.stops, 'time')} for ${hm(s.stoppedSec)} in all${s.longestStopSec >= 1800 ? `; the longest was ${hm(s.longestStopSec)}${s.longestStopAt ? ` at ${s.longestStopAt}` : ''}` : ''}.`);
  if (s.sessions) {
    const kind = s.fastSessions ? `${s.fastSessions} fast${s.sessions - s.fastSessions ? ` and ${s.sessions - s.fastSessions} slow` : ''}` : 'slow';
    const price = e.cost > 0 ? `, which cost ${s.estimated ? 'about ' : ''}${money(cur, e.cost)}${s.receipts && !s.estimated ? ' (from your receipts)' : s.receipts ? ` (${s.receipts} from receipts, ${s.estimated} worked out)` : ''}` : '';
    lines.push(`You charged ${plural(s.sessions, 'time')} (${kind}${atHome ? `; ${atHome === 1 ? 'one' : atHome} at home` : ''}) for ${hm(s.chargeSec)}, adding ${num(e.addedKwh, 0, ' kWh')} at an average of ${num(s.avgKw, 0, ' kW')} (peak ${num(s.peakKw, 0, ' kW')})${price}.`);
    if (road.roadCharges && s.firstStart && s.lastEnd > s.firstStart) { // a road trip: how much of the day went on charging, and the longest stretch between charges
      const out = s.lastEnd - s.firstStart;
      lines.push(`On the road for ${hm(out)}, ${hm(road.roadChargeSec)} of it charging (${Math.round((road.roadChargeSec / out) * 100)}%). The longest stretch between charges was ${dist(road.longestKm, 0)}.`);
    }
    if (s.unpriced) lines.push(`${plural(s.unpriced, 'session')} could not be priced. Set your price per kWh in More, Settings, or enter what you paid on the Charging page.`);
  }
  const bits = [
    s.firstStart ? `Out from ${clock(s.firstStart, x.tz)} to ${clock(s.lastEnd, x.tz)}` : null,
    s.startSoc != null && s.endSoc != null ? `Battery ${pct(s.startSoc)} to ${pct(s.endSoc)}` : null,
    s.costPerKm > 0 ? `about ${money(cur, isMetric() ? s.costPerKm : s.costPerKm * 1.609344)} per ${isMetric() ? 'km' : 'mile'} driven today` : null,
    e.usedKwh > 0 && e.usedKm > 0 ? `${num(e.usedKwh, 0, ' kWh')} used driving` : null].filter(Boolean);
  const trip = x.trip && x.trip.km > 0 ? h('p', {}, h('b', {}, `${x.trip.label || 'This trip'} so far: `), `${dist(x.trip.km, 0)} driven since ${new Date(`${x.trip.from}T12:00:00`).toLocaleDateString([], { month: 'short', day: 'numeric' })}`,
    x.trip.costPerKm != null ? `, about ${money(cur, isMetric() ? x.trip.costPerKm : x.trip.costPerKm * 1.609344)} per ${isMetric() ? 'km' : 'mile'}` : '', x.trip.base ? `. Base: ${x.trip.base}.` : '.') : null;
  return h('section', { class: 'card' }, h('h2', {}, 'Summary'), ...lines.map((l) => h('p', {}, l)), trip, bits.length ? h('p', { class: 'muted' }, bits.join(' · ')) : null);
}

// A whole day on one map: every trip's route, the stops between them numbered to match the list underneath, and the day's totals.
export default async function day(root, [date, action], session) {
  const admin = session && session.me.role === 'admin';
  const mc = await mapConfig();
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/trips' }, '← Trips')), h('div', { class: 'page-title' }, h('h2', {}, 'Day view')), box);

  async function load(d) {
    const x = await api(`/api/day${d ? `?date=${encodeURIComponent(d)}` : ''}`);
    const t = x.totals, e = x.energy;
    const label = new Date(`${x.date}T12:00:00`).toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' });
    const nav = h('div', { class: 'hmonth' }, h('button', { class: 'sec', 'aria-label': 'Previous day', onclick: () => { location.hash = `#/day/${shift(x.date, -1)}`; } }, '‹'),
      h('h3', {}, label), h('button', { class: 'sec', 'aria-label': 'Next day', onclick: () => { location.hash = `#/day/${shift(x.date, 1)}`; } }, '›'));
    const picker = h('input', { type: 'date', value: x.date, 'aria-label': 'Pick a day', style: 'max-width:12em' });
    picker.addEventListener('change', () => { if (picker.value) location.hash = `#/day/${picker.value}`; });
    const stops = x.events.filter((v) => v.kind !== 'driving' && v.kind !== 'remote' && !v.base && !v.cont); // time at the base of a trip away is not a stop; a remote start is not one either
    const roadCharges = stops.filter((v) => v.kind === 'charging' && v.where !== 'home').length; // charging stops away from home; a charge at home is not a stop
    const road = roadDay(x.events, x.tz);
    const stopNo = new Map();
    let n = 0;
    for (const v of x.events) if (v.kind !== 'driving' && v.kind !== 'remote' && !v.base && !v.cont && !road.hidden.has(v) && v.lat != null && v.lon != null) stopNo.set(v, ++n);
    const tiles = h('div', { class: 'dtiles day' },
      stat('🛣️ Distance', dist(t.km, 0)), stat('⏱️ Driving', hm(t.drivingSec)),
      e.addedKwh > 0 ? stat('⚡ Charged', `${num(e.addedKwh, 0, ' kWh')} · ${hm(t.chargingSec)}`) : null,
      e.cost > 0 ? stat('💲 Charging cost', money(x.currency, e.cost)) : null,
      e.usedKwh > 0 ? stat('🔋 Efficiency', effSpan(e.usedKm, e.usedKwh)) : null,
      stat('🛑 Stops', roadCharges ? `${roadCharges} charging` : String(stops.filter((v) => v.kind !== 'charging').length || stops.length)));
    let map = null;
    if (mc.enabled && x.routes.length) {
      const m = createMap(mc, { height: 340 });
      const routes = x.routes.map((r, i) => ({ points: r.points, color: COLORS[i % COLORS.length], width: 5,
        ...(r.kind === 'estimate' ? { dash: '0.1 9', cap: 'round', opacity: 0.85 } : {}) }));
      const first = x.routes[0].points[0], lastR = x.routes[x.routes.length - 1].points;
      const markers = [{ lat: first[0], lon: first[1], color: '#059669' }, { lat: lastR[lastR.length - 1][0], lon: lastR[lastR.length - 1][1] },
        ...[...stopNo].map(([v, no]) => ({ lat: v.lat, lon: v.lon, color: STOP_COLOR[v.kind], label: no }))];
      m.show({ routes, markers });
      map = h('div', {}, m.el, x.routes.some((r) => r.kind === 'estimate') ? h('div', { class: 'legend' }, h('span', {}, h('i'), 'recorded route'), h('span', {}, h('i', { class: 'dot' }), 'estimated road')) : null);
    } else if (x.routes.length) map = h('p', { class: 'muted' }, 'Maps are switched off, so only the list is shown.');
    const list = x.events.length ? x.events.filter((v) => !road.hidden.has(v)).flatMap((v) => [road.legBefore.get(v), ...expandable(eventRow(v, x.tz, x.currency, stopNo.get(v), road.extra.get(v)), v, x.currency, admin)].filter(Boolean)).concat(road.lastLeg ? [road.lastLeg] : []) : [emptyState('Nothing on this day', 'No trips or charging were recorded.')];
    const finished = x.events.some((v) => v.kind === 'driving') && !x.events.some((v) => v.kind === 'driving' && v.end == null);
    const share = admin && finished ? shareCard('day', x.date.replace(/-/g, ''), label) : null;
    const goShare = () => { const t = share && share.querySelector('#sht'); if (t) { share.scrollIntoView({ block: 'start' }); t.focus(); } };
    box.replaceChildren(...[nav, h('p', { class: 'seg' }, picker, share ? h('button', { class: 'sec', onclick: goShare }, 'Share this day') : null), summaryCard(x, road), tiles, map, h('h3', { style: 'margin:16px 0 6px' }, 'The day, in order'), ...list, share].filter(Boolean));
    if (action === 'share') setTimeout(goShare, 80);
  }
  await load(date);
}
