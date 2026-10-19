import { h, isMetric, num } from './ui.js';
import { speedTable } from './bestspeed.js';

const KEY = 'cartrak.stopmin';
const savedStop = () => { try { const n = Number(localStorage.getItem(KEY)); return n >= 0 && n <= 30 ? n : 5; } catch (e) { return 5; } };
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
  return h('section', { class: 'card' }, h('h2', {}, 'Best speed for this trip'),
    h('p', {}, h('b', {}, `${label(best)} gets you there soonest`), `: about ${hm(best.total)} in total.`),
    line ? h('p', {}, line) : null, ...rows,
    h('div', { style: 'display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-top:8px' }, h('label', { for: 'sm', style: 'margin:0' }, 'Time lost at each stop (plug in, parking)'), (stop.id = 'sm', stop), h('span', { class: 'muted' }, 'min')),
    h('p', { class: 'muted' }, 'Worked out from how much energy your own driving used at each speed, your charging speed above, and the weather you chose. Charging is taken at one average speed, so a stop that fills past 80% is a little slower in real life. Speeds outside what you have driven are estimates.'));
}
