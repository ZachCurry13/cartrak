import { h, speed, temp, dist, isMetric, stat, num } from './ui.js';
import { api } from './api.js';

const sign = (p) => `${p > 0 ? '+' : '−'}${Math.abs(Math.round(p))}%`;

// What each reason means for this trip, in plain words and the reader's units.
function say(f, facts) {
  const mph = isMetric() ? 'km/h' : 'mph';
  const top100 = isMetric() ? 100 : 62;
  switch (f.key) {
    case 'speed':
      return f.pct > 0
        ? `Faster than your usual driving: ${speed(facts.avgKph)} on average while moving, and ${Math.round(facts.share100 * 100)}% of the distance above ${top100} ${mph}.`
        : `Slower, gentler speeds than your usual: ${speed(facts.avgKph)} on average while moving.`;
    case 'temperature':
      return f.pct > 0
        ? `Colder or hotter than your best weather${facts.tempC != null ? ` (${temp(facts.tempC)})` : ''}. Heating, cooling and a cold battery all use energy.`
        : `Mild weather${facts.tempC != null ? ` (${temp(facts.tempC)})` : ''} helps the battery and needs little heating or cooling.`;
    case 'wind': {
      const w = facts.wind;
      const gust = w && w.gustKph >= 40 ? `, gusts to ${speed(w.gustKph)}` : '';
      const cross = w && w.crossKph >= 15 ? ` and ${speed(w.crossKph)} of crosswind` : '';
      return f.pct > 0
        ? `A headwind of about ${speed(Math.abs(w.headKph))} on average${cross}${gust}. Pushing against the wind costs more the faster you go.`
        : `A tailwind of about ${speed(Math.abs(w.headKph))} on average${cross}${gust}. It helped push the car along.`;
    }
    case 'rain':
    case 'snow': {
      const w = facts.weather;
      const what = w ? w.label.toLowerCase() : f.key;
      return `${f.key === 'snow' ? 'Snow' : 'Rain'} on the road (${what}, for ${Math.round((w ? w.wetShare : 1) * 100)}% of the time moving). A wet or snowy road adds rolling resistance and spray drag, and the heater and demister work harder.`;
    }
    default:
      return f.pct > 0
        ? 'Something else cost energy: wind, hills, heating or cooling the cabin, hard braking and quick starts. CarTrak cannot tell these apart.'
        : 'Something else helped: a tailwind, downhill stretches or gentle driving.';
  }
}

// The weather in one line, and what it cost in all: "Light rain · 41°F · 12 mph headwind".
function weatherBlock(d) {
  const f = d.facts, w = f.weather;
  if (!w && f.tempC == null) return null;
  const bits = [w ? w.label : null, f.tempC != null ? temp(f.tempC) : null,
    f.wind && Math.abs(f.wind.headKph) >= 5 ? `${speed(Math.abs(f.wind.headKph))} ${f.wind.headKph >= 0 ? 'head' : 'tail'}wind` : null].filter(Boolean);
  const total = w && Math.abs(w.totalPct) >= 3 ? `Together the weather ${w.totalPct > 0 ? 'cost about' : 'saved about'} ${Math.abs(Math.round(w.totalPct))}% ${w.totalPct > 0 ? 'more' : 'less'} energy than a trip of your usual weather.` : (w ? 'The weather made little difference to the energy used.' : null);
  const how = w && !w.dry ? (w.learned ? `The ${w.snowy ? 'snow' : 'rain'} figure is measured from your own ${w.wetTrips} wet-weather trips.` : 'The rain figure is an estimate until CarTrak has measured enough of your own wet and dry trips.') : null;
  return h('div', { style: 'margin:0 0 8px' }, h('p', { style: 'margin:0' }, h('b', {}, 'Weather: '), bits.join(' · ')),
    total ? h('p', { class: 'muted', style: 'margin:2px 0 0' }, [total, how].filter(Boolean).join(' ')) : null);
}

// "Why this efficiency?" on a trip: speed, temperature and what is left over, against your own history.
export async function whyCard(id) {
  const d = await api(`/api/trips/${id}/why`).catch(() => null);
  if (!d || !d.ready) return null;
  const vs = d.vsUsualPct;
  const head = Math.abs(vs) < 3 ? 'About as efficient as your usual.' : `${Math.abs(Math.round(vs))}% ${vs > 0 ? 'more' : 'less'} energy per ${isMetric() ? 'km' : 'mile'} than your usual.`;
  const f = d.facts;
  return h('section', { class: 'card' }, h('h2', {}, 'Why this efficiency?'),
    h('p', {}, h('b', {}, head)),
    f.regenKm != null && f.regenKm > 0 ? h('div', { class: 'grid' }, stat('Range from regen', dist(f.regenKm, 1)), stat('Energy from regen (estimate)', f.regenKwh != null ? `≈ ${num(f.regenKwh, 1, ' kWh')}` : '—')) : null,
    weatherBlock(d),
    d.tags && d.tags.length ? h('p', { class: 'tags' }, ...tagChips(d.tags)) : null,
    ...(d.factors.length ? d.factors.map((x) => h('div', { class: 'row' }, h('span', { class: 'grow' }, say(x, f)), h('b', { class: x.pct > 0 ? 'bad' : '', style: 'white-space:nowrap' }, sign(x.pct))))
      : [h('p', { class: 'muted' }, 'Nothing stands out: speed and weather were typical for you.')]),
    d.smooth ? h('p', {}, h('b', {}, { firm: 'Firmer than your usual. ', smooth: 'Smoother than your usual. ', usual: 'About as firm as your usual. ' }[d.smooth.verdict] || ''), h('span', { class: 'muted' }, `Acceleration was firm in ${Math.round(d.smooth.accelShare)}% of the readings and braking in ${Math.round(d.smooth.brakeShare)}% (about 10% each is your usual, from the car's torque readings).`)) : null,
    f.wind == null ? h('p', { class: 'muted' }, 'Wind, rain and snow are not included. You can switch on the optional weather lookup in More, Settings.') : (!d.factors.some((x) => x.key === 'wind') && (f.wind.crossKph >= 15 || f.wind.gustKph >= 40) ? h('p', { class: 'muted' }, `Wind: ${speed(Math.abs(f.wind.headKph))} ${f.wind.headKph >= 0 ? 'head' : 'tail'}wind on average, ${speed(f.wind.crossKph)} of crosswind, gusts to ${speed(f.wind.gustKph)}. It made little difference to the energy used.`) : null),
    h('p', { class: 'muted' }, [f.stops ? `${f.stops} stop${f.stops === 1 ? '' : 's'}${f.stoppedPct >= 1 ? `, ${Math.round(f.stoppedPct)}% of the time stopped` : ''}` : null, `top speed ${speed(f.maxKph)}`].filter(Boolean).join(' · ') + '. Percentages are against your own driving over the last few months.'));
}

// Short reason chips for a trip ("Cold weather", "Hard braking", …): red for what cost energy, green for what helped.
export function tagChips(tags) {
  return (tags || []).map((t) => h('span', { class: `chip ${({ bad: 'bad', good: 'ok' })[t.level] || 'info'}` }, t.label));
}

// Fills a container with a trip's reason chips once they arrive; leaves it empty if there is nothing to say.
export function fillTags(box, id) {
  api(`/api/trips/${id}/why`).then((d) => { if (d && d.tags && d.tags.length) box.replaceChildren(...tagChips(d.tags)); }).catch(() => {});
}
