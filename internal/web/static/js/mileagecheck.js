import { h, chip, dist } from './ui.js';
import { api } from './api.js';

// "Does it add up?" for the mileage log. The car's odometer is exact, so the miles it moved are compared
// with the miles CarTrak recorded as trips, and any stretch that no trip covers can be added in one tap.
const when = (ts) => new Date(ts * 1000).toLocaleString([], { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
const clock = (ts) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });

function places(g) {
  if (g.startCity && g.endCity) return g.startCity === g.endCity ? `Around ${g.startCity}` : `${g.startCity} to ${g.endCity}`;
  return g.startCity || g.endCity ? `Near ${g.startCity || g.endCity}` : '';
}

function gapRow(g, refresh, say) {
  const miles = dist(g.km, 1);
  const post = async (url, body) => {
    say('Saving…');
    try { await api(url, { method: 'POST', body }); say(''); refresh(); } catch (e) { say(e.message); }
  };
  const add = (category) => post('/api/trips/missed', { startTs: g.fromTs, endTs: g.toTs, startOdo: g.startOdo, endOdo: g.endOdo,
    startLat: g.startLat, startLon: g.startLon, endLat: g.endLat, endLon: g.endLon, category });
  const extend = () => post(`/api/trips/${g.tripId}/extend`, g.edge === 'start' ? { edge: 'start', odo: g.startOdo } : { edge: 'end', odo: g.endOdo });
  const text = g.edge
    ? `${miles} driven just ${g.edge === 'start' ? 'before' : 'after'} a recorded trip. CarTrak ${g.edge === 'start' ? 'noticed that trip late' : 'closed that trip late'}, so these miles are not in it.`
    : `${miles} driven with no trip recorded.`;
  return h('div', { class: 'gap' },
    h('div', {}, h('b', {}, `${when(g.fromTs)} to ${clock(g.toTs)}`), ' ', h('span', { class: 'muted' }, places(g))),
    h('p', { class: 'muted', style: 'margin:2px 0 8px' }, text),
    h('p', { class: 'seg', style: 'margin:0' },
      g.edge ? h('button', { onclick: extend }, 'Add to that trip') : null,
      g.edge ? null : h('button', { onclick: () => add('work') }, 'Add as work trip'),
      g.edge ? null : h('button', { class: 'sec', onclick: () => add('personal') }, 'Add as personal trip')));
}

export function checkCard(params, refresh) {
  const box = h('section', { class: 'card' }, h('h2', {}, 'Does it add up?'), h('p', { class: 'muted' }, 'Checking the car’s odometer…'));
  const say = (t) => { msg.textContent = t; };
  const msg = h('p', { class: 'muted', role: 'status' });
  const q = new URLSearchParams();
  for (const k of ['weeks', 'year', 'from', 'to']) if (params.get(k)) q.set(k, params.get(k));
  api(`/api/mileage/check?${q}`).then(({ check: c }) => {
    const summary = c.readings < 2
      ? h('p', { class: 'muted' }, 'CarTrak has no odometer readings for this period, so it cannot check it.')
      : h('p', {}, c.ok ? chip('ok', 'Matches') : chip('work', `${dist(c.missingKm, 1)} missing`), ' ',
        `The car’s odometer moved ${dist(c.drivenKm, 1)}. Trips recorded by CarTrak cover ${dist(c.coveredKm, 1)}.`);
    box.replaceChildren(h('h2', {}, 'Does it add up?'), summary,
      ...c.gaps.map((g) => gapRow(g, refresh, say)), msg,
      h('p', { class: 'muted' }, 'This compares every kind of trip, work and personal, not only the ones in the log above. The odometer total is exact; Ford reports it only when the car wakes, so the times inside a gap are approximate.'));
  }).catch((e) => box.replaceChildren(h('h2', {}, 'Does it add up?'), h('p', { class: 'muted' }, e.message)));
  return box;
}
