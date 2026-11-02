// How the same kinds of figure are written, everywhere. A page that needs money, a length of time or a day uses these and does not
// make its own, so a charge costs "$4.20", a drive takes "1 h 5 min" and a date reads the same on every screen.

import { locale } from './region.js';

const missing = (v) => v == null || Number.isNaN(+v);

// Money: the currency symbol, thousands separators and two decimals ("$1,000.00", "$4.20"); more decimals for what is priced very
// finely ("$0.125" a mile). The separators follow the country chosen in Display and data (United States by default).
export const money = (cur, v, decimals = 2) => (missing(v) ? '—' : `${cur}${(+v).toLocaleString(locale(), { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}`);

// A length of time given in seconds: "45 min", "1 h 5 min". Under a minute still says "1 min", never "0 min".
export function hm(seconds) {
  if (missing(seconds)) return '—';
  const total = Math.max(1, Math.round(+seconds / 60));
  if (total < 60) return `${total} min`;
  const h = Math.floor(total / 60), m = total % 60;
  return m ? `${h} h ${m} min` : `${h} h`;
}

// The same for a length of time given in minutes.
export const hmMin = (minutes) => (missing(minutes) ? '—' : hm(+minutes * 60));

// Days and times of day: one style each.
export const dayLong = (d) => d.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' });
export const dayShort = (d) => d.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
export const dayTime = (d) => d.toLocaleString([], { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
export const timeOfDay = (d) => d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
