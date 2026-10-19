import { h, isMetric, num, dist } from './ui.js';
import { speedTable } from './bestspeed.js';

const KEY = 'cartrak.stopmin';
const savedStop = () => { try { const n = Number(localStorage.getItem(KEY)); return n >= 0 && n <= 30 ? n : 5; } catch (e) { return 5; } };
const PKEY = 'cartrak.cmpspeeds';
const savedPair = () => { try { const p = JSON.parse(localStorage.getItem(PKEY)); return Array.isArray(p) ? p : null; } catch (e) { return null; } };
const hm = (m) => (m < 60 ? `${Math.round(m)} min` : `${Math.floor(m / 60)} h ${Math.round(m % 60)} min`);
const MI = 0.621371;

// "Best speed for this trip": total time at each cruising speed, counting the charging the extra energy forces.
export function bestSpeedCard(args, rerun) {
  const stop = h('input', { type: 'number', min: 0, max: 30, inputmode: 'numeric', value: savedStop(), 'aria-label': 'Minutes lost per charging stop', style: 'max-width:5em' });
  stop.addEventListener('change', () => { try { localStorage.setItem(KEY, String(Number(stop.value) || 0)); } catch (e) { /* storage blocked */ } rerun(); });
  const metric = isMetric();
  const speeds = metric ? [90, 100, 110, 120, 130, 140] : [55, 60, 65, 70, 75, 80, 85].map((m) => m / MI);
  const t = speedTable({ ...args, stopMinutes: Number(stop.value) || 0, speeds });
  if (!t) return null;
  const label = (r) => `${Math.round(metric ? r.kph : r.kph * MI)} ${metric ? 'km/h' : 'mph'}`;
  const best = t.best;
  // Two speeds side by side, such as the 70 you drive now and the 75 you are tempted by. Any speed can be typed.
  const pair = savedPair();
  const dflt = metric ? [110, 120] : [70, 75];
  const pick = (i) => {
    const v = pair && Number(pair[i]) >= 20 && Number(pair[i]) <= 200 ? Number(pair[i]) : dflt[i];
    const s = h('input', { type: 'number', min: 20, max: 200, step: 1, inputmode: 'numeric', value: v, 'aria-label': i ? 'Compare with (speed)' : 'Compare (speed)', style: 'max-width:5em' });
    s.addEventListener('change', () => { try { localStorage.setItem(PKEY, JSON.stringify([a.value, b.value])); } catch (e) { /* storage blocked */ } rerun(); });
    return s;
  };
  const a = pick(0), b = pick(1);
  const toKph = (v) => (metric ? Number(v) : Number(v) / MI);
  const two = speedTable({ ...args, stopMinutes: Number(stop.value) || 0, speeds: [toKph(a.value), toKph(b.value)] });
  const ra = two && two.rows.find((r) => Math.abs(r.kph - toKph(a.value)) < 1e-6), rb = two && two.rows.find((r) => Math.abs(r.kph - toKph(b.value)) < 1e-6);
  // One hour of driving at each speed: how far it goes, the energy it uses and the fast charging that gives the energy back.
  const hourOf = (r) => { const kwh = r.kph * (r.kwh / args.distanceKm); return { km: r.kph, kwh, mins: args.chargeKw > 0 ? (kwh / args.chargeKw) * 60 : null }; };
  const hourLine = (r) => { const x = hourOf(r); return h('div', { class: 'row' }, h('b', { class: 'grow' }, `1 hour at ${label(r)}`), h('span', {}, `${dist(x.km, 0)} · ${num(x.kwh, 1, ' kWh')}${x.mins != null ? ` · ${hm(x.mins)} to fast charge it back` : ''}`)); };
  const hour = ra && rb && ra !== rb ? (() => {
    const [slow, fast] = ra.kph < rb.kph ? [ra, rb] : [rb, ra];
    const s1 = hourOf(slow), f1 = hourOf(fast);
    return h('div', {}, hourLine(slow), hourLine(fast), f1.mins != null ? h('p', {}, `In that hour ${label(fast)} takes you ${dist(f1.km - s1.km, 0)} further and costs ${hm(f1.mins - s1.mins)} more fast charging.`) : null);
  })() : null;
  const cmp = ra && rb && ra !== rb ? (() => {
    const [slow, fast] = ra.kph < rb.kph ? [ra, rb] : [rb, ra];
    const drive = slow.drive - fast.drive, charge = fast.charge - slow.charge, stops = fast.stops - slow.stops, net = fast.total - slow.total;
    return h('p', {}, h('b', {}, `${label(fast)} against ${label(slow)}: `), `${hm(drive)} less driving, ${hm(Math.abs(charge))} ${charge < 0 ? 'less' : 'more'} charging${stops ? ` (${stops > 0 ? stops + ' more stop' + (stops === 1 ? '' : 's') : -stops + ' fewer stop' + (stops === -1 ? '' : 's')})` : ''} and ${hm(Math.abs(net))} ${net < 0 ? 'quicker' : 'slower'} overall. It uses ${num(fast.kwh - slow.kwh, 0, ' kWh')} more energy (${Math.round((fast.kwh / slow.kwh - 1) * 100)}% more per ${metric ? 'km' : 'mile'}).`);
  })() : null;
  const both = h('div', {}, hour, cmp);
  const faster = t.rows.find((r) => r.kph > best.kph);
  const slower = [...t.rows].reverse().find((r) => r.kph < best.kph);
  let line = null;
  if (faster) {
    const saves = best.drive - faster.drive, costs = faster.charge + faster.stopTime - best.charge - best.stopTime;
    line = `Going from ${label(best)} to ${label(faster)} saves ${hm(saves)} of driving but adds ${hm(Math.max(0, costs))} of charging and stops, so it is ${hm(Math.abs(saves - costs))} ${saves - costs < 0 ? 'slower' : 'faster'} overall and uses ${num(faster.kwh - best.kwh, 0, ' kWh')} more energy.`;
  }
  if (!faster && slower) {
    line = `Even at the top speed shown, going faster still gets you there sooner on this trip: the charging it adds takes less time than the driving it saves. Slowing to ${label(slower)} would add ${hm(slower.total - best.total)} to the trip and use ${num(best.kwh - slower.kwh, 0, ' kWh')} less energy.`;
  }
  const rows = t.rows.map((r) => h('div', { class: `row${r === best ? ' on' : ''}`, style: r === best ? 'font-weight:700' : '' },
    h('span', { class: 'grow' }, label(r), r === best ? ' · fastest overall' : '', r.estimated ? h('span', { class: 'muted' }, ' · estimate') : null,
      h('div', { class: 'muted', style: 'font-weight:400' }, `drive ${hm(r.drive)} · charge ${hm(r.charge)}${r.stops ? ` · ${r.stops} stop${r.stops === 1 ? '' : 's'}` : ' · no stop'}`)),
    h('b', {}, hm(r.total))));
  if (args.compareOnly) return h('div', {}, h('div', { style: 'display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin:8px 0' }, 'Compare', a, h('span', { class: 'muted' }, metric ? 'km/h with' : 'mph with'), b, h('span', { class: 'muted' }, metric ? 'km/h' : 'mph')), both, h('p', { class: 'muted' }, 'Driving, charging and stops for a trip this long, starting at 90% and charging between your reserve and target. Estimates from your own driving and charging.'));
  return h('section', { class: 'card' }, h('h2', {}, 'Best speed for this trip'),
    h('p', {}, h('b', {}, `${label(best)} gets you there soonest`), `: about ${hm(best.total)} in total.`),
    line ? h('p', {}, line) : null,
    h('div', { style: 'display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin:8px 0' }, 'Compare', a, h('span', { class: 'muted' }, metric ? 'km/h with' : 'mph with'), b, h('span', { class: 'muted' }, metric ? 'km/h' : 'mph')), both, ...rows,
    h('div', { style: 'display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-top:8px' }, h('label', { for: 'sm', style: 'margin:0' }, 'Time lost at each stop (plug in, parking)'), (stop.id = 'sm', stop), h('span', { class: 'muted' }, 'min')),
    h('p', { class: 'muted' }, 'Worked out from how much energy your own driving used at each speed, your charging speed above, and the weather you chose. Charging is taken at one average speed, so a stop that fills past 80% is a little slower in real life. Speeds outside what you have driven are estimates.'));
}
