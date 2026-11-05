import { h, chip, dist } from './ui.js';
import { eventRow, money } from './eventrow.js';
import { hmMin as hm } from './format.js';

const bendKm = (a, b) => {
  const r = 6371, p = Math.PI / 180, dp = (b.lat - a.lat) * p, dl = (b.lon - a.lon) * p;
  const x = Math.sin(dp / 2) ** 2 + Math.cos(a.lat * p) * Math.cos(b.lat * p) * Math.sin(dl / 2) ** 2;
  return 2 * r * Math.asin(Math.sqrt(x)) * 1.3; // a straight line made 30% longer, as the planner does
};

// A planned day shown the way the Day view shows a real one: a row for each drive and each charge, with clock times, the battery
// on the way in and out, the energy added and what it should cost. ctx: { currency, review(site), avoid(site), use() }.
export function tripDay(d, i, n, ctx) {
  const at = (min) => d.departTs + Math.round(min * 60);
  const rows = [];
  d.stops.forEach((s, j) => {
    if (j > 0) {
      const p = d.stops[j - 1];
      rows.push(eventRow({ kind: 'driving', start: at(p.departMin), end: at(s.arriveMin), fromPlace: p.name, place: s.name, distanceKm: s.legKm || bendKm(p, s), startSoc: p.departPct, endSoc: s.arrivePct }, undefined, ctx.currency));
    }
    if (s.chargeMin > 0) {
      const row = eventRow({ kind: 'charging', where: '', place: s.net || s.name.startsWith("Supercharger: ") ? s.name : `Supercharger: ${s.name}`, start: at(s.arriveMin), end: at(s.departMin), addedKwh: s.kwh, cost: s.cost == null ? null : s.cost, costActual: false, startSoc: s.arrivePct, endSoc: s.departPct }, undefined, ctx.currency, j);
      rows.push(plain(row), h('div', { class: 'seg' }, h('a', { class: 'text sm', href: ctx.review(s), target: '_blank', rel: 'noopener' }, 'Reviews on PlugShare'),
        h('button', { class: 'text sm', type: 'button', onclick: () => ctx.avoid(s) }, 'Avoid this charger')));
    }
  });
  const last = d.stops[d.stops.length - 1];
  const cost = d.stops.reduce((t, s) => t + (s.cost || 0), 0);
  return h('section', { class: 'card' },
    h('h3', {}, n > 1 ? `Day ${i + 1}` : 'The trip', ' ', h('span', { class: 'muted' }, `${new Date(d.departTs * 1000).toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })}`)),
    h('p', { class: 'muted' }, `${dist(d.km, 0)} · ${hm(d.driveMin)} driving${d.chargeMin ? ` + about ${hm(d.chargeMin)} charging` : ''} · ${d.chargeStops} charge stop${d.chargeStops === 1 ? '' : 's'}${cost > 0 ? ` · about ${money(ctx.currency, cost)}` : ''} · starts at ${Math.round(d.startPct)}%`),
    ...rows,
    last.kind === 'overnight' ? h('div', { class: 'row' }, chip('work', 'Overnight'), h('span', { class: 'grow' }, last.name),
      last.near ? h('span', { class: 'muted' }, `Closest charger: ${last.near.name}, ${dist(last.near.km, 1)} away`) : null) : null,
    h('p', { class: 'seg' }, h('button', { type: 'button', onclick: () => ctx.use(d, i) }, 'Plan this day in the planner'), ...ctx.links(d, i, n)));
}

// The Day view's row is a link to the charging page; a planned charge goes nowhere yet, so it is drawn without the link.
function plain(el) {
  if (el.tagName !== 'A') return el;
  const div = h('div', { class: el.className });
  div.append(...el.childNodes);
  return div;
}
