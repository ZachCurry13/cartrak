import { isMetric } from './ui.js';

// Efficiency needs a real distance and a real amount of energy before it means anything.
const MIN_KM = 1.6;
const MIN_KWH = 0.3;

// "3.42 mi/kWh" or "18.2 kWh/100 km" following this device's unit choice, or null when not meaningful.
export function efficiency(distanceKm, kwh) {
  if (kwh == null || kwh < MIN_KWH || !(distanceKm >= MIN_KM)) return null;
  return isMetric() ? `${((kwh / distanceKm) * 100).toFixed(1)} kWh/100 km` : `${((distanceKm * 0.621371) / kwh).toFixed(2)} mi/kWh`;
}

export const kwhText = (v, estimated) => (v == null ? '—' : `${estimated ? '≈ ' : ''}${v.toFixed(1)} kWh`);
