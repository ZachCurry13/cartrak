import { h, dist, speed, when, duration, num, isMetric, catChip } from '../ui.js';
import { api } from '../api.js';
import { effSpan, kwhText } from '../energy.js';

// Two trips side by side. A star marks the better of the two where "better" means something.
const eff = (t) => (t.energyKwh >= 0.3 && t.distanceKm >= 1.6 ? t.distanceKm / t.energyKwh : null); // km per kWh
const mins = (t) => (t.endTs && t.startTs ? (t.endTs - t.startTs) / 60 : null);
const avgKph = (t) => (mins(t) > 0 ? t.distanceKm / (mins(t) / 60) : null);
const used = (t) => (t.startSoc != null && t.endSoc != null ? t.startSoc - t.endSoc : null);

function row(label, a, b, text, better) {
  const win = (x, y) => (better && x != null && y != null && x !== y && (better === 'high' ? x > y : x < y));
  const cell = (t, x, y) => h('b', { style: 'text-align:right' }, text(t), win(x, y) ? ' ★' : '');
  return h('div', { class: 'row' }, h('span', { class: 'grow muted' }, label), cell(a.t, a.v, b.v), h('span', { style: 'width:12px' }), cell(b.t, b.v, a.v));
}

export default async function compare(root, [ids]) {
  const [ia, ib] = String(ids || '').split(',').map((x) => encodeURIComponent(x));
  root.append(h('p', {}, h('a', { href: '#/trips' }, '← Trips')), h('h2', {}, 'Compare trips'));
  if (!ia || !ib) return root.append(h('p', { class: 'muted' }, 'Pick two trips on the Trips page (Select trips, tap two, then Compare).'));
  const [a, b] = (await Promise.all([ia, ib].map((i) => api(`/api/trips/${i}?samples=0`)))).map((r) => r.trip);
  const x = (t, v) => ({ t, v });
  const header = h('div', { class: 'row' }, h('span', { class: 'grow' }),
    ...[a, b].flatMap((t, i) => [i ? h('span', { style: 'width:12px' }) : null, h('a', { href: `#/trip/${t.id}`, style: 'text-align:right' }, when(t.startTs, t.tz))].filter(Boolean)));
  const ea = eff(a), eb = eff(b);
  const say = ea && eb && Math.abs(ea - eb) / Math.max(ea, eb) >= 0.03
    ? `${ea > eb ? 'The first' : 'The second'} trip used about ${Math.round((Math.abs(ea - eb) / Math.max(ea, eb)) * 100)}% less energy per ${isMetric() ? 'km' : 'mile'}.`
    : ea && eb ? 'Both trips used about the same energy for the distance.' : null;
  root.append(
    h('section', { class: 'card' }, header,
      row('Route', x(a, null), x(b, null), (t) => (t.startPlace || t.endPlace ? `${t.startPlace || '…'} → ${t.endPlace || '…'}` : '—')),
      row('Type', x(a, null), x(b, null), (t) => catChip(t)),
      row('Distance', x(a, a.distanceKm), x(b, b.distanceKm), (t) => dist(t.distanceKm)),
      row('Time', x(a, mins(a)), x(b, mins(b)), (t) => duration(t.startTs, t.endTs)),
      row('Average speed', x(a, avgKph(a)), x(b, avgKph(b)), (t) => speed(avgKph(t))),
      row('Top speed', x(a, a.maxSpeedKph), x(b, b.maxSpeedKph), (t) => speed(t.maxSpeedKph)),
      row('Battery used', x(a, used(a)), x(b, used(b)), (t) => (used(t) != null ? num(used(t), 0, '%') : '—')),
      row('Energy used', x(a, a.energyKwh), x(b, b.energyKwh), (t) => kwhText(t.energyKwh, t.energyEstimated)),
      row('Efficiency', x(a, ea), x(b, eb), (t) => (eff(t) ? effSpan(t.distanceKm, t.energyKwh, t.energyEstimated ? '≈ ' : '') : '—'), 'high')),
    say ? h('p', {}, say) : null,
    h('p', { class: 'muted' }, '★ marks the better of the two. Efficiency needs a real distance and amount of energy, so very short trips show a dash.'));
}
