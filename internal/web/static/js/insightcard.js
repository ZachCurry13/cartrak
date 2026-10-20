import { h, tempBandLabel } from './ui.js';
import { api } from './api.js';
import { efficiency } from './energy.js';

const monthName = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'long', year: 'numeric' }); };
const REASON = { speed: 'Driving fast', temperature: 'Weather', wind: 'Wind', other: 'Everything else' };

// One thing worth knowing about how the car is driven, taken from the efficiency insights. It changes each day, and tapping it
// opens all of them.
export async function insightCard() {
  const [d, why] = await Promise.all([api('/api/insights').catch(() => null), api('/api/efficiency-reasons?days=60').catch(() => null)]);
  if (!d || d.trips < 3) return null;
  const facts = [];
  const bands = (d.bands || []).filter((b) => b.trips >= 2 && b.kmPerKwh > 0);
  if (bands.length >= 2) {
    const best = bands.reduce((a, b) => (b.kmPerKwh > a.kmPerKwh ? b : a));
    const worst = bands.reduce((a, b) => (b.kmPerKwh < a.kmPerKwh ? b : a));
    const pct = Math.round((1 - worst.kmPerKwh / best.kmPerKwh) * 100);
    if (pct >= 5) facts.push(`${tempBandLabel(worst.name)} weather costs you about ${pct}% more energy than ${tempBandLabel(best.name)}: ${efficiency(worst.kmPerKwh, 1)} against ${efficiency(best.kmPerKwh, 1)}.`);
  }
  const months = (d.months || []).filter((m) => m.trips >= 3 && m.kmPerKwh > 0);
  if (months.length >= 2) {
    const best = months.reduce((a, b) => (b.kmPerKwh > a.kmPerKwh ? b : a));
    facts.push(`Your most efficient month was ${monthName(best.month)}, at ${efficiency(best.kmPerKwh, 1)} over ${best.trips} trips.`);
  }
  if (d.best && d.best[0]) facts.push(`Your most efficient trip so far got ${efficiency(d.best[0].kmPerKwh, 1)}.`);
  if (why && why.items) {
    const top = why.items.filter((i) => i.kwh >= 1).sort((a, b) => b.kwh - a.kwh)[0];
    if (top) facts.push(`${REASON[top.key] || top.key} used about ${Math.round(top.kwh)} kWh more than your usual over the last ${why.days} days (${Math.round(top.pctOfUse)}% of what you used).`);
  }
  if (!facts.length) return null;
  const pick = facts[Math.floor(Date.now() / 86400000) % facts.length];
  return h('a', { class: 'card', href: '#/insights', style: 'display:block;text-decoration:none;color:inherit' }, h('h2', {}, 'Did you know'), h('p', {}, pick), h('p', { class: 'muted' }, 'See all efficiency insights'));
}
