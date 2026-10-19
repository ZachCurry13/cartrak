import { h, speed, temp, dist, isMetric } from './ui.js';
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
    default:
      return f.pct > 0
        ? 'Something else cost energy: wind, hills, heating or cooling the cabin, hard braking and quick starts. CarTrak cannot tell these apart.'
        : 'Something else helped: a tailwind, downhill stretches or gentle driving.';
  }
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
    d.tags && d.tags.length ? h('p', { class: 'tags' }, ...tagChips(d.tags)) : null,
    ...(d.factors.length ? d.factors.map((x) => h('div', { class: 'row' }, h('span', { class: 'grow' }, say(x, f)), h('b', { class: x.pct > 0 ? 'bad' : '', style: 'white-space:nowrap' }, sign(x.pct))))
      : [h('p', { class: 'muted' }, 'Nothing stands out: speed and weather were typical for you.')]),
    h('p', { class: 'muted' }, [f.regenKm != null ? `braking gave back ${dist(f.regenKm, 1)} of range` : null, f.stops ? `${f.stops} stop${f.stops === 1 ? '' : 's'}${f.stoppedPct >= 1 ? `, ${Math.round(f.stoppedPct)}% of the time stopped` : ''}` : null, `top speed ${speed(f.maxKph)}`].filter(Boolean).join(' · ') + '. Percentages are against your own driving over the last few months.'));
}

// Short reason chips for a trip ("Cold weather", "Hard braking", …): red for what cost energy, green for what helped.
export function tagChips(tags) {
  return (tags || []).map((t) => h('span', { class: `chip ${({ bad: 'bad', good: 'ok' })[t.level] || 'info'}` }, t.label));
}

// Fills a container with a trip's reason chips once they arrive; leaves it empty if there is nothing to say.
export function fillTags(box, id) {
  api(`/api/trips/${id}/why`).then((d) => { if (d && d.tags && d.tags.length) box.replaceChildren(...tagChips(d.tags)); }).catch(() => {});
}
