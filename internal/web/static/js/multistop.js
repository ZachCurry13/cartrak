import { h, chip, dist, speed, temp, num, tempBandLabel } from './ui.js';
import { api } from './api.js';
import { routeUrls } from './maplinks.js';
import { routeSummary } from './routesummary.js';
import { matchDetail } from './placematch.js';
import { hmMin as hm } from './format.js';
import { pasteListCard } from './companylist.js';
import { tripChainCard } from './tripchain.js';

const MAX = 11; // the start and up to ten destinations
const clock = (ts) => new Date(ts * 1000).toLocaleString([], { weekday: 'short', hour: 'numeric', minute: '2-digit' });
const localInput = (d) => new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);

// A route with up to ten destinations: where the battery will be at each stop, the forecast on each leg and how your car
// usually drives in that weather, and where a charge is needed. `get()` gives the starting charge, reserve, target and charging power.
export async function multiStopCard(get, latest) {
  const named = (await api('/api/named-places').catch(() => [])) || [];
  const stops = [{ name: '', label: '', lat: null, lon: null }, { name: '', label: '', lat: null, lon: null }];
  if (latest && latest.lat != null && latest.lon != null) Object.assign(stops[0], { name: 'Where the car is', label: 'Where the car is', lat: latest.lat, lon: latest.lon });
  const when = h('input', { type: 'datetime-local', id: 'ms-when', value: localInput(new Date()), 'aria-label': 'Leaving at' });
  const list = h('div'), out = h('div'), msg = h('p', { class: 'muted', role: 'status' });
  const best = h('input', { type: 'checkbox', id: 'ms-best', checked: true });
  const back = h('input', { type: 'checkbox', id: 'ms-back', checked: true });
  const tick = (box, id, text) => h('label', { for: id, style: 'display:flex;gap:8px;align-items:center;margin:4px 0;min-height:44px' }, box, text);

  const rowFor = (s, i) => {
    const q = h('input', { value: s.label, placeholder: i === 0 ? 'Starting point' : 'Business or address', maxlength: 200, 'aria-label': i === 0 ? 'Start' : `Stop ${i}` });
    const found = h('div');
    const set = (name, label, lat, lon) => { Object.assign(s, { name, label, lat, lon }); draw(); };
    const find = async () => {
      found.replaceChildren(h('p', { class: 'muted' }, 'Searching...'));
      try {
        const r = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`);
        found.replaceChildren(...(r.length ? r.map((m) => h('button', { class: 'sec', style: 'display:block;width:100%;text-align:left;margin:4px 0;white-space:normal', onclick: () => set(m.name, m.address && m.address !== m.name ? `${m.name}, ${m.address}` : m.name, m.lat, m.lon) },
          h('b', {}, m.name), h('div', { class: 'muted' }, matchDetail(m))))
          : [h('p', { class: 'muted' }, 'Not found. Add the town, or type the address.')]));
      } catch (e) { found.replaceChildren(h('p', { class: 'muted' }, e.message)); }
    };
    q.addEventListener('keydown', (e) => { if (e.key === 'Enter') find(); });
    q.addEventListener('input', () => { s.lat = s.lon = null; });
    const pick = named.length ? h('select', { 'aria-label': 'Or one of your places', onchange: (e) => { const p = named.find((x) => String(x.id) === e.target.value); if (p) set(p.name, p.name, p.lat, p.lon); e.target.value = ''; } },
      h('option', { value: '' }, 'or one of your places'), ...named.map((p) => h('option', { value: String(p.id) }, p.name))) : null;
    const move = (d) => () => { const j = i + d; if (j >= 0 && j < stops.length) { [stops[i], stops[j]] = [stops[j], stops[i]]; draw(); } };
    const icon = (label, text, onclick, disabled) => h('button', { class: 'sec', 'aria-label': label, disabled, onclick, style: 'flex:0 0 44px;min-width:44px;padding:0' }, text);
    const line = 'gap:6px;align-items:center;flex-wrap:nowrap;margin-top:6px';
    const mark = h('b', { style: 'flex:0 0 1.4em;text-align:center' }, i === 0 ? 'A' : String(i));
    // only the buttons that do something: the start stays first, and the first stop cannot move above it
    const tools = [i > 1 ? icon('Move up', '↑', move(-1)) : null, i > 0 && i < stops.length - 1 ? icon('Move down', '↓', move(1)) : null, i > 0 && stops.length > 2 ? icon('Remove this stop', '✕', () => { stops.splice(i, 1); draw(); }) : null].filter(Boolean);
    if (s.lat != null) { // a chosen stop: its name and a way to change it, not a search box that would look the name up again
      // the name, and under it the address if there is one (not the name again); the buttons on their own line so the name has the width
      const title = s.name || s.label;
      const sub = s.label && s.label.startsWith(`${title}, `) ? s.label.slice(title.length + 2) : s.label && s.label !== title && !s.label.startsWith(title) ? s.label : '';
      return h('div', { class: 'card', style: 'margin:8px 0;padding:10px' },
        h('div', { class: 'row', style: 'gap:8px;align-items:flex-start;flex-wrap:wrap' }, mark, h('div', { style: 'flex:1 1 6em;min-width:0;overflow-wrap:anywhere' }, h('b', {}, title), sub ? h('div', { class: 'muted' }, sub) : null),
          h('div', { style: 'display:flex;gap:6px;margin-left:auto' }, h('button', { class: 'sec', onclick: () => { Object.assign(s, { name: '', label: '', lat: null, lon: null }); draw(); } }, 'Change'), ...tools)));
    }
    return h('div', { class: 'card', style: 'margin:8px 0;padding:10px' },
      h('div', { class: 'row', style: line.replace('margin-top:6px', 'margin-top:0') }, mark, h('div', { style: 'flex:1 1 auto;min-width:0' }, q), h('button', { class: 'sec', style: 'flex:0 0 auto', onclick: find }, 'Find')),
      pick || tools.length ? h('div', { class: 'row', style: line }, h('div', { style: 'flex:1 1 auto;min-width:0' }, pick || ''), ...tools) : null, found);
  };
  function draw() {
    list.replaceChildren(...stops.map(rowFor), h('p', {}, h('button', { class: 'sec', disabled: stops.length >= MAX, onclick: () => { stops.push({ name: '', label: '', lat: null, lon: null }); draw(); } }, 'Add a stop'),
      ' ', h('span', { class: 'muted' }, `${stops.length - 1} of 10 stops`)));
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
      h('div', { class: 'muted' }, `${num(l.kwh, 1, ' kWh')} used${w.tempBand ? ` · your usual efficiency at ${tempBandLabel(w.tempBand)}` : ''}${w.learned ? ' · rain from your own wet trips' : ''}`));
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
      out.replaceChildren(h('section', { class: 'card' }, h('h2', {}, 'The route'), routeSummary(r, g.startPct),
        h('p', {}, charges ? chip('work', `${charges} charging stop${charges === 1 ? '' : 's'}`) : chip('ok', 'No charging needed'), ' ', h('span', { class: 'muted' }, `${dist(r.totalKm, 0)} · ${hm(r.drivingMinutes)} driving${r.chargingMinutes ? ` + ${hm(r.chargingMinutes)} charging` : ''} · ${num(r.totalKwh, 0, ' kWh')} · arrive with about ${Math.round(r.arrivePct)}%`)),
        r.savedMinutes > 0 ? h('p', { class: 'muted', style: 'margin:0' }, `Best order saves about ${hm(r.savedMinutes)} of driving.`) : null,
        h('p', { class: 'seg', style: 'margin:8px 0' }, ...links.map((href, i) => h('a', { class: 'btn', href, target: '_blank', rel: 'noopener' }, links.length > 1 ? `Open part ${i + 1} of ${links.length} in Google Maps` : 'Open this route in Google Maps'))),
        links.length > 1 ? h('p', { class: 'muted' }, 'Google takes 10 legs at a time, so a long day is split in parts.') : null,
        ...r.warnings.map((x) => h('p', { class: 'banner' }, x)), ...r.legs.map(legRow), ...r.notes.map((x) => h('p', { class: 'muted' }, x)),
        h('p', { class: 'muted' }, `Each leg uses your car's real efficiency at that speed and temperature, plus the forecast wind and rain. Charging assumes a charger at the stop and ${g.chargeKw} kW. ${r.reservePct}% is kept in reserve. Google does not know the charging stops: add a charger to its route where you plug in.`)));
    } catch (e) { msg.textContent = e.message; }
  }
  draw();
  const show = (k) => { for (const [id, p] of Object.entries(panels)) p.hidden = id !== k; for (const [id, b] of Object.entries(tabBtn)) { b.classList.toggle('on', id === k); b.setAttribute('aria-pressed', String(id === k)); } };
  const toStops = (note) => { out.replaceChildren(); draw(); msg.textContent = note; show('stops'); };
  const panels = {
    stops: h('div', {}, h('p', { class: 'muted' }, 'Up to 10 stops. Search a business or address, or pick a saved place.'),
      list, tick(best, 'ms-best', 'Best order (least driving)'), tick(back, 'ms-back', 'Return to the start'),
      h('label', { for: 'ms-when' }, 'Leaving at'), when, h('p', {}, h('button', { onclick: plan }, 'Plan the route')), msg, out),
    list: pasteListCard((found) => { stops.splice(1, stops.length - 1, ...found); toStops(`${found.length} stops filled in. Check them, then plan the route.`); }),
    long: tripChainCard(get, () => stops[0], (day, i, pct) => { stops.splice(0, stops.length, ...day); best.checked = false; back.checked = false; toStops(`Day ${i + 1} is filled in${i ? `: set the starting charge to about ${Math.round(pct)}% first` : ''}. Plan the route to see the roads, weather and charging.`); }),
  };
  const tabBtn = {};
  const tabs = h('div', { class: 'seg', role: 'group', 'aria-label': 'What to plan' },
    ...[['stops', 'Stops'], ['list', 'Paste a list'], ['long', 'Long trip']].map(([k, t]) => (tabBtn[k] = h('button', { class: 'sec', type: 'button', onclick: () => show(k) }, t))));
  show('stops');
  return h('section', { class: 'card' }, h('h2', {}, 'Plan a route'), tabs, ...Object.values(panels));
}
