import { dist, num, temp, speed, isMetric } from './ui.js';
import { efficiency } from './energy.js';

// The words for Wrapped and the badges: a distance compared with something everyone knows, energy in everyday terms, and each
// kind of number written the way people say it.

const KM_PER_MI = 1.609344;
// [km, how to say one of them]; the comparisons are general on purpose, nothing about where anyone lives
const DISTANCES = [
  [384400, 'the way to the Moon'],
  [40075, 'the way around the Earth'],
  [4500, 'a drive across the USA, coast to coast'],
  [3940, 'Route 66, end to end'],
  [1400, 'the length of Florida and back'],
  [42.195, 'a marathon'],
];

// "about half of Route 66, end to end", "2.3 times around the Earth"
export function compareDistance(km) {
  for (const [d, what] of DISTANCES) {
    const r = km / d;
    if (r < 0.4) continue;
    if (r < 0.9) return `${r < 0.6 ? 'about half of' : `${Math.round(r * 10) * 10}% of`} ${what}`;
    if (r < 1.15) return `about ${what}`;
    if (d === 42.195) return `${Math.round(r)} marathons`;
    return `${r < 10 ? r.toFixed(1) : Math.round(r)} times ${what.replace(/^the way /, '').replace(/^a /, '')}`;
  }
  return `${Math.max(1, Math.round(km / 42.195))} marathon${km >= 63 ? 's' : ''}`;
}

// energy in everyday terms: an average home uses about 29 kWh a day, a phone charge is about 0.015 kWh
export function compareEnergy(kwh) {
  const days = kwh / 29;
  return days >= 1.5 ? `enough to run a home for ${Math.round(days)} days` : `${Math.round(kwh / 0.015).toLocaleString()} phone charges`;
}

export const milesOrKm = (km) => Math.round(isMetric() ? km : km / KM_PER_MI).toLocaleString();
export const distUnit = () => (isMetric() ? 'kilometres' : 'miles');

// a value of a badge or a record, by its kind
export function valueText(kind, v, cur = '$') {
  switch (kind) {
    case 'km': return dist(v, v < 100 ? 1 : 0);
    case 'kmPerKwh': return efficiency(v * 100, 100) || '—';
    case 'kph': return speed(v);
    case 'tempC': return temp(v);
    case 'kw': return num(v, 0, ' kW');
    case 'kwh': return v >= 1000 ? `${(v / 1000).toFixed(1)} MWh` : num(v, 0, ' kWh');
    case 'price': return `${cur}${v.toFixed(2)}/kWh`;
    case 'days': return `${Math.round(v)} days`;
    case 'towns': return `${Math.round(v)} towns`;
    default: return Math.round(v).toLocaleString();
  }
}

// "{pct} of your charging started after 9 pm" with the number filled in
export const personalityLine = (p) => p.why.replace('{pct}', `${Math.round(Math.min(1, p.share) * 100)}%`).replace('{eff}', efficiency(p.share * 7.2 * 100, 100) || '');

// "07:15" in the phone's own clock style (7:15 AM, or 07:15)
export const clockText = (hhmm) => { const [hh, mm] = hhmm.split(':').map(Number); return new Date(2000, 0, 1, hh, mm).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }); };

export const monthName = (period) => {
  const [y, m] = period.split('-').map(Number);
  return m ? new Date(y, m - 1, 1).toLocaleDateString([], { month: 'long' }) : String(y);
};
export const periodTitle = (period) => (period.includes('-') ? `${monthName(period)} ${period.slice(0, 4)}` : period);
