import { h, chip, dist, isMetric } from './ui.js';
import { api } from './api.js';
import { matchDetail } from './placematch.js';
import { hmMin as hm } from './format.js';

const toKm = (v) => (isMetric() ? v : v / 0.621371);
const PLUGS = [['both', 'Either: NACS plugs or a built-in CCS dock'], ['nacs', 'NACS plugs only (with the NACS adapter)'], ['dock', 'Built-in CCS dock (Magic Dock) only']];

// A long trip: give a destination and your limits and CarTrak picks the Superchargers your car may use, no leg over your time or
// distance, and cuts the trip into days. `get()` gives the planner's charge settings, `start()` where the trip begins and
// `use(stops, day)` fills one day into the route planner below, which adds the real roads, weather and battery.
export function tripChainCard(get, start, use) {
  let dest = null;
  const unit = isMetric() ? 'km' : 'miles';
  const q = h('input', { placeholder: 'Final address or business', maxlength: 200, 'aria-label': 'Destination' });
  const found = h('div'), out = h('div'), msg = h('p', { class: 'muted', role: 'status' });
  const legH = h('input', { id: 'tc-h', type: 'number', inputmode: 'decimal', min: 0.5, step: 0.25, value: 2.5 });
  const legD = h('input', { id: 'tc-d', type: 'number', inputmode: 'decimal', min: 0, step: 10, value: isMetric() ? 320 : 200 });
  const dayH = h('input', { id: 'tc-day', type: 'number', inputmode: 'decimal', min: 2, step: 0.5, value: 12 });
  const plugs = h('select', { id: 'tc-plugs' }, ...PLUGS.map(([v, t]) => h('option', { value: v }, t)));
  const field = (id, label, input) => [h('label', { for: id }, label), input];

  const showDest = () => {
    found.replaceChildren(dest ? h('p', {}, h('b', {}, dest.name), ' ', h('button', { class: 'text sm', onclick: () => { dest = null; showDest(); } }, 'Change')) : '');
  };
  async function search() {
    found.replaceChildren(h('p', { class: 'muted' }, 'Searching...'));
    try {
      const r = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`);
      found.replaceChildren(...(r.length ? r.map((m) => h('button', { class: 'sec', style: 'display:block;width:100%;text-align:left;margin:4px 0;white-space:normal',
        onclick: () => { dest = { name: m.name || m.label, lat: m.lat, lon: m.lon }; showDest(); } }, h('b', {}, m.name), h('div', { class: 'muted' }, matchDetail(m))))
        : [h('p', { class: 'muted' }, 'Not found. Add the town, or type the address.')]));
    } catch (e) { found.replaceChildren(h('p', { class: 'muted' }, e.message)); }
  }
  q.addEventListener('keydown', (e) => { if (e.key === 'Enter') search(); });

  const day = (d, i, n) => {
    const stops = d.stops;
    return h('div', { class: 'card', style: 'margin:8px 0;padding:10px' },
      h('p', { style: 'margin:0 0 4px' }, h('b', {}, n > 1 ? `Day ${i + 1}` : 'The trip'), ' ', h('span', { class: 'muted' }, `${dist(d.km, 0)} · ${hm(d.driveMin)} driving${d.chargeMin ? ` + about ${hm(d.chargeMin)} charging` : ''} · ${d.chargeStops} charge stop${d.chargeStops === 1 ? '' : 's'}`)),
      ...stops.map((s, j) => h('div', { class: 'row', style: 'gap:8px' }, h('span', { style: 'flex:0 0 1.4em;text-align:center' }, j === 0 ? 'A' : String(j)),
        h('div', { class: 'grow', style: 'min-width:0;overflow-wrap:anywhere' }, s.name, s.kw ? h('span', { class: 'muted' }, ` · ${s.kw} kW`) : null),
        s.kind === 'overnight' && j > 0 ? chip('work', 'Overnight') : null, s.kind === 'charge' ? chip('ok', 'Charge') : null)),
      i > 0 ? h('p', { class: 'muted', style: 'margin:4px 0' }, 'Starts about the charge target after charging overnight.') : null,
      h('p', { style: 'margin:6px 0 0' }, h('button', { onclick: () => use(stops.map((s) => ({ name: s.name, label: s.name, lat: s.lat, lon: s.lon })), i) }, 'Plan this day below')));
  };

  async function go() {
    const s = start();
    if (!dest) { msg.textContent = 'Find the destination first.'; return; }
    if (!s || s.lat == null) { msg.textContent = 'The route planner below needs its starting point first.'; return; }
    const g = get();
    msg.textContent = 'Finding chargers...'; out.replaceChildren();
    try {
      const r = await api('/api/trip-chain', { method: 'POST', body: { start: { name: s.name || 'Start', lat: s.lat, lon: s.lon }, dest, startPct: g.startPct, reservePct: g.reservePct, targetPct: g.targetPct,
        chargeKw: g.chargeKw, maxLegMin: (Number(legH.value) || 0) * 60, maxLegKm: legD.value ? toKm(Number(legD.value)) : 0, dayMin: (Number(dayH.value) || 12) * 60, plugs: plugs.value } });
      msg.textContent = `${r.sites} Superchargers open to other cars were considered.`;
      out.replaceChildren(...r.days.map((d, i) => day(d, i, r.days.length)),
        h('p', { class: 'muted' }, 'Stops are picked with estimated distances; planning a day below uses real roads, the forecast and your car\'s usual efficiency. A site that lets other cars charge can still be full or down: check it in the Tesla app or Google Maps before you go.'));
    } catch (e) { msg.textContent = e.message; }
  }
  return h('details', { class: 'card', style: 'margin:8px 0;padding:10px' }, h('summary', {}, 'Long trip: pick the Superchargers'),
    h('p', { class: 'muted' }, 'Set how long a leg may be and how long a day on the road, and CarTrak chooses chargers your car can use between the start and the destination. The list of sites is downloaded from supercharge.info about once a week; nothing about you or the trip is sent.'),
    ...field('tc-q', 'Destination', h('div', {}, q, h('p', {}, h('button', { class: 'sec', onclick: search }, 'Find')), found)),
    ...field('tc-h', 'Longest leg (hours)', legH), ...field('tc-d', `Or no more than (${unit})`, legD), ...field('tc-day', 'Longest day on the road (hours)', dayH), ...field('tc-plugs', 'Chargers', plugs),
    h('p', {}, h('button', { onclick: go }, 'Find the chargers')), msg, out);
}
