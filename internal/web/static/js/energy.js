import { h, isMetric } from './ui.js';
import { api } from './api.js';

// Efficiency needs a real distance and a real amount of energy before it means anything.
const MIN_KM = 1.6;
const MIN_KWH = 0.3;
const MI = 0.621371;

// Three ways to say it; tap any efficiency figure to change them all.
//   kwh  distance per kWh (mi/kWh), or kWh per 100 km when metric
//   wh   watt-hours per mile (or per km)
//   pct  distance per percent of battery (needs the battery size, learned from the car)
const UNITS = ['kwh', 'wh', 'pct'];
const KEY = 'cartrak.effunit';
let unit = 'kwh';
try { unit = UNITS.includes(localStorage.getItem(KEY)) ? localStorage.getItem(KEY) : 'kwh'; } catch (e) { /* storage blocked */ }
let capacityKwh = null; // energy in a full battery, for the per-percent view
let loading = false;

function loadCapacity() {
  if (capacityKwh || loading) return;
  loading = true;
  api('/api/latest?summary=1').then((l) => { if (l && l.capacityKwh > 0) { capacityKwh = l.capacityKwh; refresh(); } }).catch(() => { /* not signed in or offline */ });
}

// "3.42 mi/kWh", "292 Wh/mi" or "1.2 mi/%" following the chosen unit, or null when not meaningful.
// The efficiency unit chosen on this device: kwh, wh or pct.
export const getEfficiencyUnit = () => unit;

export function efficiency(distanceKm, kwh) {
  if (kwh == null || kwh < MIN_KWH || !(distanceKm >= MIN_KM)) return null;
  const metric = isMetric();
  if (unit === 'pct' && capacityKwh) {
    const pctUsed = kwh / (capacityKwh / 100);
    return metric ? `${(distanceKm / pctUsed).toFixed(2)} km/%` : `${((distanceKm * MI) / pctUsed).toFixed(2)} mi/%`;
  }
  if (unit === 'wh') {
    return metric ? `${Math.round((kwh * 1000) / distanceKm)} Wh/km` : `${Math.round((kwh * 1000) / (distanceKm * MI))} Wh/mi`;
  }
  return metric ? `${((kwh / distanceKm) * 100).toFixed(1)} kWh/100 km` : `${((distanceKm * MI) / kwh).toFixed(2)} mi/kWh`;
}

function refresh() {
  document.querySelectorAll('.eff').forEach((el) => {
    const t = efficiency(Number(el.dataset.km), Number(el.dataset.kwh));
    if (t) el.textContent = `${el.dataset.pre || ''}${t}`;
  });
}

function cycle() {
  const order = capacityKwh ? UNITS : UNITS.filter((u) => u !== 'pct');
  unit = order[(order.indexOf(unit) + 1) % order.length] || 'kwh';
  try { localStorage.setItem(KEY, unit); } catch (e) { /* ignore */ }
  refresh();
}

// The figure as a tappable element. `pre` goes in front, for example "≈ " when it is an estimate. Returns a
// plain dash when there is nothing meaningful to show.
export function effSpan(distanceKm, kwh, pre = '') {
  const text = efficiency(distanceKm, kwh);
  if (!text) return document.createTextNode('—');
  loadCapacity();
  const el = h('span', { class: 'eff', role: 'button', tabindex: '0', title: 'Tap to change the unit', style: 'cursor:pointer;border-bottom:1px dotted currentColor' }, `${pre}${text}`);
  el.dataset.km = String(distanceKm);
  el.dataset.kwh = String(kwh);
  el.dataset.pre = pre;
  const go = (e) => { e.preventDefault(); e.stopPropagation(); cycle(); };
  el.addEventListener('click', go);
  el.addEventListener('keydown', (e) => { if (e.key === 'Enter' || e.key === ' ') go(e); });
  return el;
}

export const kwhText = (v, estimated) => (v == null ? '—' : `${estimated ? '≈ ' : ''}${v.toFixed(1)} kWh`);
