// How numbers and dates are written on this device. United States by default ($1,000.00, 11/02/2026, 12-hour clock); a few other
// countries are offered in Display and data. The choice is remembered on the device, like the units.
const REGIONS = [['US', 'en-US', 'United States'], ['CA', 'en-CA', 'Canada'], ['GB', 'en-GB', 'United Kingdom'], ['AU', 'en-AU', 'Australia'], ['DE', 'de-DE', 'Germany']];

let code = 'US';
try { code = localStorage.getItem('region') || 'US'; } catch (e) { /* storage blocked: United States */ }

export const locale = () => (REGIONS.find((r) => r[0] === code) || REGIONS[0])[1];

export function setRegion(c) {
  code = REGIONS.some((r) => r[0] === c) ? c : 'US';
  try { localStorage.setItem('region', code); } catch (e) { /* ignore */ }
}

export const getRegion = () => code;

// The choices with a sample date of each: "United States (11/02/2026)".
export function regionChoices() {
  const sample = new Date(2026, 10, 2);
  return REGIONS.map(([c, loc, name]) => [c, `${name} (${sample.toLocaleDateString(loc, { year: 'numeric', month: '2-digit', day: '2-digit' })})`]);
}

// Dates, times and numbers drawn without naming a language come out in the chosen country's style, on every screen, instead of in whatever
// the phone is set to. Only calls that name none are changed; one that names its own is left alone.
const unnamed = (l) => l === undefined || (Array.isArray(l) && l.length === 0);
for (const [proto, method] of [[Date.prototype, 'toLocaleDateString'], [Date.prototype, 'toLocaleTimeString'], [Date.prototype, 'toLocaleString'], [Number.prototype, 'toLocaleString']]) {
  const original = proto[method];
  proto[method] = function (l, o) { return original.call(this, unnamed(l) ? locale() : l, o); };
}
