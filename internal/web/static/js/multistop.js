import { h, chip, dist, speed, temp, num } from './ui.js';
import { api } from './api.js';
import { routeUrls } from './maplinks.js';

const MAX = 11; // the start and up to ten destinations
const hm = (m) => (m < 60 ? `${Math.round(m)} min` : `${Math.floor(m / 60)} h ${Math.round(m % 60)} min`);
const clock = (ts) => new Date(ts * 1000).toLocaleString([], { weekday: 'short', hour: 'numeric', minute: '2-digit' });
const localInput = (d) => new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);

// A route with up to ten destinations: where the battery will be at each stop, the forecast on each leg and how your car
// usually drives in that weather, and where a charge is needed. `get()` gives the starting charge, reserve, target and charging power.
export async function multiStopCard(get, latest) {
  const named = (await api('/api/named-places').catch(() => [])) || [];
  const stops = [{ name: '', label: '', lat: null, lon: null }, { name: '', label: '', lat: null, lon: null }];
  if (latest && latest.lat != null && latest.lon != null) Object.assign(stops[0], { name: 'Where the car is', label: 'Where the car is now', lat: latest.lat, lon: latest.lon });
  const when = h('input', { type: 'datetime-local', id: 'ms-when', value: localInput(new Date()), 'aria-label': 'Leaving at' });
  const list = h('div'), out = h('div'), msg = h('p', { class: 'muted', role: 'status' });
  const best = h('input', { type: 'checkbox', id: 'ms-best', checked: true });
  const back = h('input', { type: 'checkbox', id: 'ms-back', checked: true });
  const tick = (box, id, text) => h('label', { for: id, style: 'display:flex;gap:8px;align-items:center;margin:4px 0;min-height:44px' }, box, text);

  const rowFor = (s, i) => {
    const q = h('input', { value: s.label, placeholder: i === 0 ? 'Start: an address' : `Destination ${i}: an address`, maxlength: 200, 'aria-label': i === 0 ? 'Start' : `Destination ${i}` });
    const found = h('div');
    const set = (name, label, lat, lon) => { Object.assign(s, { name, label, lat, lon }); draw(); };
    const find = async () => {
      found.replaceChildren(h('p', { class: 'muted' }, 'Searching...'));
      try {
        const r = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`);
        found.replaceChildren(...(r.length ? r.map((m) => h('button', { class: 'sec', style: 'display:block;text-align:left;margin:4px 0;white-space:normal', onclick: () => set(m.label.split(',').slice(0, 2).join(',').trim(), m.label.split(',').slice(0, 3).join(',').trim(), m.lat, m.lon) }, m.label))
          : [h('p', { class: 'muted' }, 'Nothing found. Try the street and the town.')]));
      } catch (e) { found.replaceChildren(h('p', { class: 'muted' }, e.message)); }
    };
    q.addEventListener('keydown', (e) => { if (e.key === 'Enter') find(); });
    q.addEventListener('input', () => { s.lat = s.lon = null; });
    const pick = named.length ? h('select', { 'aria-label': 'Or one of your places', onchange: (e) => { const p = named.find((x) => String(x.id) === e.target.value); if (p) set(p.name, p.name, p.lat, p.lon); e.target.value = ''; } },
      h('option', { value: '' }, 'or one of your places'), ...named.map((p) => h('option', { value: String(p.id) }, p.name))) : null;
    const move = (d) => () => { const j = i + d; if (j >= 0 && j < stops.length) { [stops[i], stops[j]] = [stops[j], stops[i]]; draw(); } };
    return h('div', { style: 'margin:0 0 12px' }, h('div', { class: 'row', style: 'gap:6px;align-items:center;flex-wrap:wrap' }, h('div', { style: 'flex:1 1 12em;min-width:0' }, q),
      h('button', { class: 'sec', onclick: find }, 'Find'), pick,
      h('button', { class: 'sec', 'aria-label': 'Move up', disabled: i === 0, onclick: move(-1) }, '↑'), h('button', { class: 'sec', 'aria-label': 'Move down', disabled: i === stops.length - 1, onclick: move(1) }, '↓'),
      stops.length > 2 ? h('button', { class: 'sec', 'aria-label': 'Remove this stop', onclick: () => { stops.splice(i, 1); draw(); } }, 'Remove') : null),
      s.lat != null ? h('div', { class: 'muted' }, `✓ ${s.label}`) : null, found);
  };
  function draw() {
    list.replaceChildren(...stops.map(rowFor), h('p', {}, h('button', { class: 'sec', disabled: stops.length >= MAX, onclick: () => { stops.push({ name: '', label: '', lat: null, lon: null }); draw(); } }, 'Add a stop'),
      ' ', h('span', { class: 'muted' }, `${stops.length - 1} of 10 destinations`)));
  }

  const legRow = (l) => {
    const w = l.weather;
    const bits = w.known ? [w.label, w.sun ? w.sun.toLowerCase() : null, w.tempC != null ? temp(w.tempC) : null, Math.abs(w.headKph) >= 5 ? `${speed(Math.abs(w.headKph))} ${w.headKph >= 0 ? 'head' : 'tail'}wind` : null].filter(Boolean).join(' · ') : null;
    const extra = w.known ? Math.round(((1 + w.windPct / 100) * (1 + w.rainPct / 100) - 1) * 100) : 0;
    return h('div', { class: 'card', style: 'margin:8px 0' },
      l.charge ? h('p', { style: 'margin:0 0 6px' }, chip('work', `Charge at ${l.from} from ${Math.round(l.charge.fromPct)}% to ${Math.round(l.charge.toPct)}%`), ' ', h('span', { class: 'muted' }, `about ${hm(l.charge.minutes)}`)) : null,
      h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, `${l.from} → ${l.to}`), h('div', { class: 'muted' }, `${dist(l.km, 0)}${l.estimated ? ' (straight line)' : ''} · ${hm(l.minutes)} · ${clock(l.departTs)} to ${clock(l.arriveTs)}`)),
        h('b', { class: l.short ? 'bad' : '' }, `${Math.round(l.arrivePct)}%`)),
      bits ? h('div', {}, h('span', {}, bits), extra >= 2 ? h('span', { class: 'muted' }, ` · wind and rain add about ${extra}%`) : null) : null,
      h('div', { class: 'muted' }, `${num(l.kwh, 1, ' kWh')} used${w.tempBand ? ` · your usual efficiency in ${w.tempBand.toLowerCase()} weather` : ''}${w.learned ? ' · rain from your own wet trips' : ''}`));
  };
  async function plan() {
    const bad = stops.findIndex((s) => s.lat == null);
    if (bad >= 0) { msg.textContent = `Find ${bad === 0 ? 'the start' : `destination ${bad}`} first: type an address and tap Find, or pick one of your places.`; return; }
    const g = get();
    msg.textContent = 'Planning...'; out.replaceChildren();
    try {
      const r = await api('/api/route-plan', { method: 'POST', body: { stops: stops.map((s) => ({ name: s.name, lat: s.lat, lon: s.lon })), departTs: Math.floor(new Date(when.value).getTime() / 1000) || 0, startPct: g.startPct, reservePct: g.reservePct, targetPct: g.targetPct, chargeKw: g.chargeKw, optimize: best.checked, roundTrip: back.checked } });
      msg.textContent = '';
      if (best.checked && r.savedMinutes > 0) { const was = stops.slice(); stops.splice(0, stops.length, ...r.order.map((i) => was[i])); draw(); } // show the order it chose
      const links = routeUrls(r.stops.map((s) => [s.lat, s.lon]));
      const charges = r.legs.filter((l) => l.charge).length;
      out.replaceChildren(h('section', { class: 'card' }, h('h2', {}, 'The route'),
        h('p', {}, charges ? chip('work', `${charges} charging stop${charges === 1 ? '' : 's'}`) : chip('ok', 'No charging needed'), ' ', h('span', { class: 'muted' }, `${dist(r.totalKm, 0)} · ${hm(r.drivingMinutes)} driving${r.chargingMinutes ? ` + ${hm(r.chargingMinutes)} charging` : ''} · ${num(r.totalKwh, 0, ' kWh')} · arrive with about ${Math.round(r.arrivePct)}%`)),
        r.savedMinutes > 0 ? h('p', { class: 'muted', style: 'margin:0' }, `Best order: about ${hm(r.savedMinutes)} less driving than the order you gave.`) : null,
        h('p', { class: 'seg', style: 'margin:8px 0' }, ...links.map((href, i) => h('a', { class: 'btn', href, target: '_blank', rel: 'noopener' }, links.length > 1 ? `Open part ${i + 1} of ${links.length} in Google Maps` : 'Open this route in Google Maps'))),
        links.length > 1 ? h('p', { class: 'muted' }, 'Google Maps takes up to ten legs at a time, so a long day is in parts: each carries on from where the one before ended.') : null,
        ...r.warnings.map((x) => h('p', { class: 'banner' }, x)), ...r.legs.map(legRow), ...r.notes.map((x) => h('p', { class: 'muted' }, x)),
        h('p', { class: 'muted' }, `Each leg uses how your car really drives at that speed and temperature, plus the wind and rain forecast for the hours you would be on it. Charging stops assume a charger at the stop and ${g.chargeKw} kW on average. Charging time is not counted in the forecast hours. Keep ${r.reservePct}% in reserve. Google Maps gets your start, stops and destination in this order; it does not know about the charging stops CarTrak suggests, so add a charger to Google's route where you plug in.`)));
    } catch (e) { msg.textContent = e.message; }
  }
  draw();
  return h('section', { class: 'card' }, h('h2', {}, 'Plan a route with stops'),
    h('p', { class: 'muted' }, 'A day of customers? The start is where you begin (and finish, if you tick coming back). Add up to ten destinations: type an address and tap Find, or pick one of your places. Uses your starting charge, reserve, charge target and charging power from below.'),
    list, tick(best, 'ms-best', 'Find the best order for the destinations (least driving)'), tick(back, 'ms-back', 'Come back to the start at the end'),
    h('label', { for: 'ms-when' }, 'Leaving at'), when, h('p', {}, h('button', { onclick: plan }, 'Plan the route')), msg, out);
}
