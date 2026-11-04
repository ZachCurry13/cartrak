import { h, isMetric } from './ui.js';
import { api } from './api.js';
import { routeUrls } from './maplinks.js';
import { matchDetail } from './placematch.js';
import { hmMin as hm } from './format.js';
import { tripDay } from './longtripdays.js';

const toKm = (v) => (isMetric() ? v : v / 0.621371);
const PLUGS = [['both', 'Superchargers: NACS plugs or a built-in CCS dock'], ['nacs', 'Superchargers: NACS plugs only (with the adapter)'], ['dock', 'Superchargers: built-in CCS dock only'], ['none', 'No Superchargers']];
const AVOID = 'cartrak.avoid';
const key = (s) => `${s.lat.toFixed(4)},${s.lon.toFixed(4)}`;
const readAvoid = () => { try { return JSON.parse(localStorage.getItem(AVOID)) || {}; } catch (e) { return {}; } };
const writeAvoid = (a) => { try { localStorage.setItem(AVOID, JSON.stringify(a)); } catch (e) { /* storage blocked */ } };
const review = (s) => (s.ps ? `https://www.plugshare.com/location/${s.ps}` : `https://www.plugshare.com/?latitude=${s.lat}&longitude=${s.lon}&zoom=17`);

// A place you search for and choose: set(place) fills it, get() reads it.
function picker(label) {
  let place = null;
  const q = h('input', { placeholder: label, maxlength: 200, 'aria-label': label });
  const found = h('div'), box = h('div');
  const draw = () => {
    box.replaceChildren(place ? h('p', { style: 'margin:4px 0' }, h('b', {}, place.name), ' ', h('button', { class: 'text sm', type: 'button', onclick: () => { place = null; draw(); } }, 'Change'))
      : h('div', {}, q, h('p', { style: 'margin:4px 0' }, h('button', { class: 'sec', type: 'button', onclick: search }, 'Find')), found));
  };
  async function search() {
    found.replaceChildren(h('p', { class: 'muted' }, 'Searching...'));
    try {
      const r = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`);
      found.replaceChildren(...(r.length ? r.map((m) => h('button', { class: 'sec', type: 'button', style: 'display:block;width:100%;text-align:left;margin:4px 0;white-space:normal',
        onclick: () => { place = { name: m.name || m.label, lat: m.lat, lon: m.lon }; draw(); } }, h('b', {}, m.name), h('div', { class: 'muted' }, matchDetail(m))))
        : [h('p', { class: 'muted' }, 'Not found. Add the town, or type the address.')]));
    } catch (e) { found.replaceChildren(h('p', { class: 'muted' }, e.message)); }
  }
  q.addEventListener('keydown', (e) => { if (e.key === 'Enter') search(); });
  draw();
  return { el: box, get: () => place, set: (p) => { place = p; draw(); } };
}

// A long trip: a destination, places you insist on (a town on the road you want, a hotel already booked), the networks you accept
// and your limits. CarTrak picks the chargers, splits the trip into days and hands a day to the route planner. `get()` gives the
// planner's charge settings, `start()` where the trip begins and `use(stops, day)` fills one day into the planner.
export function tripChainCard(get, start, use) {
  const unit = isMetric() ? 'km' : 'miles';
  const dest = picker('Final address or business');
  const vias = []; // { pick, charge, night }
  const viaBox = h('div'), out = h('div'), msg = h('p', { class: 'muted', role: 'status' }), saved = h('div'), avoidBox = h('div');
  const input = (id, val, min, step) => h('input', { id, type: 'number', inputmode: 'decimal', min, step, value: val });
  const legH = input('tc-h', 2.5, 0.5, 0.25), legD = input('tc-d', isMetric() ? 320 : 200, 0, 10), dayH = input('tc-day', 12, 2, 0.5);
  const plugs = h('select', { id: 'tc-plugs' }, ...PLUGS.map(([v, t]) => h('option', { value: v }, t)));
  const ea = h('input', { type: 'checkbox', id: 'tc-ea' });
  const field = (id, label, el) => [h('label', { for: id }, label), el];
  let last = null; // what was last planned: { body, result }

  const newVia = (v = {}) => { const pick = picker('A town, or the hotel you booked'); if (v.lat != null) pick.set({ name: v.name, lat: v.lat, lon: v.lon }); return { pick, charge: v.charge !== false, night: !!v.night }; };
  const drawVias = () => viaBox.replaceChildren(...vias.map((v, i) => h('div', { class: 'card', style: 'margin:6px 0;padding:8px' },
    h('div', { class: 'muted' }, `Place ${i + 1}, in the order you drive them`), v.pick.el,
    h('select', { 'aria-label': 'What happens here', onchange: (e) => { v.night = e.target.value === 'night'; } },
      h('option', { value: 'pass', selected: !v.night }, 'Drive through it'), h('option', { value: 'night', selected: v.night }, 'Stop for the night here (a hotel)')),
    h('label', { style: 'display:flex;gap:8px;align-items:center;min-height:44px' }, h('input', { type: 'checkbox', checked: v.charge, onchange: (e) => { v.charge = e.target.checked; } }), 'Plug in here, or at a charger beside it'),
    h('button', { class: 'text sm', type: 'button', onclick: () => { vias.splice(i, 1); drawVias(); } }, 'Remove')))
    .concat(vias.length < 6 ? [h('button', { class: 'sec', type: 'button', onclick: () => { vias.push(newVia()); drawVias(); } }, 'Add a place')] : []));

  const drawAvoid = () => {
    const a = readAvoid(), keys = Object.keys(a);
    avoidBox.replaceChildren(keys.length ? h('details', {}, h('summary', {}, `Chargers you avoid (${keys.length})`), ...keys.map((k) => h('div', { class: 'row' }, h('div', { class: 'grow' }, a[k]),
      h('button', { class: 'text sm', type: 'button', onclick: () => { const b = readAvoid(); delete b[k]; writeAvoid(b); drawAvoid(); } }, 'Allow again')))) : '');
  };

  const links = (points, what) => routeUrls(points).map((href, i, all) => h('a', { class: 'btn', href, target: '_blank', rel: 'noopener' }, all.length > 1 ? `${what}: part ${i + 1} of ${all.length}` : `${what} in Google Maps`));
  const avoidSite = (x) => { const a = readAvoid(); a[key(x)] = x.name; writeAvoid(a); go(); };
  const dayCard = (d, i, n) => tripDay(d, i, n, { currency: last.result.currency || '$', review, avoid: avoidSite,
    use: (day, k) => use(day.stops.map((x) => ({ name: x.name, label: x.name, lat: x.lat, lon: x.lon })), k, day.startPct),
    links: (day, k, count) => links(day.stops.map((x) => [x.lat, x.lon]), count > 1 ? `Open day ${k + 1}` : 'Open the trip') });

  function show(r, body) {
    last = { body, result: r };
    const all = r.days.flatMap((d, i) => (i ? d.stops.slice(1) : d.stops)).map((s) => [s.lat, s.lon]);
    out.replaceChildren(
      r.alt ? h('p', { class: 'banner' }, `Adding Electrify America would save about ${hm(r.alt.minutesSaved)} (${r.alt.chargeStops} charge stop${r.alt.chargeStops === 1 ? '' : 's'} instead of ${r.alt.was}). `,
        h('button', { class: 'sec', type: 'button', onclick: () => { ea.checked = true; go(); } }, 'Plan it with Electrify America')) : null,
      ...r.days.map((d, i) => dayCard(d, i, r.days.length)),
      r.days.length > 1 ? h('p', { class: 'seg' }, ...links(all, 'Open the whole trip')) : null,
      h('p', { class: 'seg' }, h('button', { class: 'sec', type: 'button', onclick: saveTrip }, 'Save this trip')),
      h('p', { class: 'muted' }, 'Stops are picked with estimated distances; planning a day in the planner uses real roads, the forecast and your car\'s usual efficiency. Superchargers come first, and Electrify America is used when it saves enough to be worth it. A site can be full or down, so check its reviews before you go.'));
  }
  const body = () => {
    const s = start(), d = dest.get(), g = get();
    if (!d) { msg.textContent = 'Find the destination first.'; return null; }
    if (!s || s.lat == null) { msg.textContent = 'The Stops tab needs its starting point first.'; return null; }
    if (vias.some((v) => !v.pick.get())) { msg.textContent = 'Finish or remove the places you added.'; return null; }
    return { start: { name: s.name || 'Start', lat: s.lat, lon: s.lon }, dest: d, via: vias.map((v) => ({ ...v.pick.get(), charge: v.charge, night: v.night })), startPct: g.startPct, reservePct: g.reservePct, targetPct: g.targetPct,
      chargeKw: g.chargeKw, departTs: g.departTs || 0, maxLegMin: (Number(legH.value) || 0) * 60, maxLegKm: legD.value ? toKm(Number(legD.value)) : 0, dayMin: (Number(dayH.value) || 12) * 60, plugs: plugs.value, ea: ea.checked, avoid: Object.keys(readAvoid()) };
  };
  async function go() {
    const b = body();
    if (!b) return;
    msg.textContent = 'Finding chargers...'; out.replaceChildren(); drawAvoid();
    try {
      const r = await api('/api/trip-chain', { method: 'POST', body: b });
      msg.textContent = `${r.sites} chargers were considered.`;
      show(r, b);
    } catch (e) { msg.textContent = e.message; }
  }

  async function loadSaved() {
    try {
      const l = await api('/api/saved-trips');
      saved.replaceChildren(l.length ? h('details', {}, h('summary', {}, `Saved trips (${l.length})`), ...l.map((t) => h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, t.name)),
        h('button', { class: 'text sm', type: 'button', onclick: () => open(t) }, 'Open'),
        h('button', { class: 'text sm', type: 'button', onclick: async () => { if (confirm(`Delete ${t.name}?`)) { await api(`/api/saved-trips/${t.id}`, { method: 'DELETE' }); loadSaved(); } } }, 'Delete')))) : '');
    } catch (e) { saved.replaceChildren(); }
  }
  function open(t) {
    try {
      const x = JSON.parse(t.body), b = x.body;
      dest.set(b.dest);
      vias.splice(0, vias.length, ...b.via.map(newVia));
      legH.value = b.maxLegMin / 60; legD.value = isMetric() ? Math.round(b.maxLegKm) : Math.round(b.maxLegKm * 0.621371); dayH.value = b.dayMin / 60;
      plugs.value = b.plugs; ea.checked = !!b.ea; drawVias(); msg.textContent = `Opened ${t.name}. Find the chargers again for today's battery and charger list.`; show(x.result, b);
    } catch (e) { msg.textContent = 'That saved trip could not be opened.'; }
  }
  async function saveTrip() {
    const name = prompt('Name this trip', `${last.body.dest.name} trip`);
    if (!name) return;
    try { await api('/api/saved-trips', { method: 'POST', body: { name, body: { body: last.body, result: last.result } } }); msg.textContent = `Saved as ${name}.`; loadSaved(); } catch (e) { msg.textContent = e.message; }
  }
  loadSaved(); drawAvoid(); drawVias();
  return h('div', {},
    h('p', { class: 'muted' }, 'Set how long a leg may be and how long a day on the road, and CarTrak chooses chargers your car can use between the start and the destination. Lists of chargers are downloaded about once a week; nothing about you or the trip is sent.'),
    ...field('tc-q', 'Destination', dest.el), h('h3', {}, 'Places you insist on'),
    h('p', { class: 'muted' }, 'Add a town to take one road and not another, or a hotel you already booked to end a day there. They are passed in the order you add them.'), viaBox,
    ...field('tc-h', 'Longest leg (hours)', legH), ...field('tc-d', `Or no more than (${unit})`, legD), ...field('tc-day', 'Longest day on the road (hours)', dayH), ...field('tc-plugs', 'Chargers', plugs),
    h('label', { for: 'tc-ea', style: 'display:flex;gap:8px;align-items:center;min-height:44px' }, ea, 'Also use Electrify America (needs the charger list key in Settings)'),
    h('p', {}, h('button', { onclick: go }, 'Find the chargers')), msg, out, avoidBox, saved);
}
