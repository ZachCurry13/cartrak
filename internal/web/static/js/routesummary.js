import { h, chip, temp, speed, num } from './ui.js';
import { hmMin as hm } from './format.js';

const RAIN = ['Drizzle', 'Light rain', 'Rain', 'Heavy rain', 'Thunderstorm'];

// Where the legs are: "on the way to Ames, Bell and 2 more".
const wayTo = (legs) => `on the way to ${legs.length > 3 ? `${legs.slice(0, 2).map((l) => l.to).join(', ')} and ${legs.length - 2} more` : legs.map((l) => l.to).join(legs.length > 2 ? ', ' : ' and ')}`;

// The answer first: what you will have left if you leave now, and what the weather on the way will be like.
export function routeSummary(r, startPct) {
  const legs = r.legs;
  const charges = legs.filter((l) => l.charge);
  const chargeMin = charges.reduce((m, l) => m + l.charge.minutes, 0);
  const low = legs.some((l) => l.short) || r.arrivePct < r.reservePct;
  const where = r.roundTrip ? `be back at ${r.stops[0].name || 'the start'}` : `arrive at ${legs[legs.length - 1].to}`;
  const head = `Leave at ${Math.round(startPct)}% and you will ${where} with about ${Math.round(r.arrivePct)}%${charges.length ? `, after ${charges.length} charging stop${charges.length === 1 ? '' : 's'} (about ${hm(chargeMin)})` : ''}.`;

  const known = legs.filter((l) => l.weather && l.weather.known);
  const alerts = [];
  const add = (text, subset) => { if (subset.length) alerts.push(`${text} ${wayTo(subset)}.`); };
  add('Rain is expected', known.filter((l) => RAIN.includes(l.weather.label)));
  add('Snow is expected', known.filter((l) => /snow/i.test(l.weather.label)));
  const cold = known.filter((l) => l.weather.tempC != null && l.weather.tempC <= 0);
  if (cold.length) alerts.push(`Freezing weather is expected ${wayTo(cold)}: down to ${temp(Math.min(...cold.map((l) => l.weather.tempC)))}. Cold uses more battery, and roads can ice up.`);
  const icy = known.filter((l) => RAIN.includes(l.weather.label) && l.weather.tempC != null && l.weather.tempC > 0 && l.weather.tempC <= 2);
  add('Rain close to freezing is expected, so watch for ice', icy);
  const windy = known.filter((l) => l.weather.gustKph >= 55 || Math.abs(l.weather.headKph) >= 30);
  if (windy.length) alerts.push(`Strong wind is expected ${wayTo(windy)}: gusts to ${speed(Math.max(...windy.map((l) => l.weather.gustKph)))}.`);
  const hot = known.filter((l) => l.weather.tempC != null && l.weather.tempC >= 32);
  if (hot.length) alerts.push(`Hot weather is expected ${wayTo(hot)} (${temp(Math.max(...hot.map((l) => l.weather.tempC)))}), so the air conditioning will use more.`);

  let extra = 0, total = 0;
  for (const l of known) { const f = (1 + l.weather.windPct / 100) * (1 + l.weather.rainPct / 100); extra += l.kwh * (1 - 1 / f); }
  for (const l of legs) total += l.kwh;
  const pct = total - extra > 0 ? (extra / (total - extra)) * 100 : 0;

  return h('div', { style: 'margin:0 0 8px' },
    h('p', { style: 'margin:0 0 6px' }, chip(low ? 'bad' : charges.length ? 'work' : 'ok', low ? 'Low on battery' : charges.length ? 'Needs a charge' : 'Enough battery'), ' ', h('b', {}, head)),
    ...alerts.map((a) => h('p', { class: 'banner', style: 'margin:4px 0' }, a)),
    pct >= 2 ? h('p', { class: 'muted', style: 'margin:4px 0' }, `The wind and rain add about ${Math.round(pct)}% (${num(extra, 1, ' kWh')}) to the energy for this route.`) : null);
}
