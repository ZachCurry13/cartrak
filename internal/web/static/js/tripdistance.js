import { h, chip, dist, isMetric } from './ui.js';
import { api } from './api.js';

const SRC = { road: 'set from the estimated road', direct: 'set from the drive between start and end', manual: 'entered by you' };

// The straight line from a trip's start to its end, km, or 0 when a position is missing.
export function straightKm(t) {
  if (t.startLat == null || t.endLat == null) return 0;
  const r = Math.PI / 180, a = Math.sin(((t.endLat - t.startLat) * r) / 2) ** 2 + Math.cos(t.startLat * r) * Math.cos(t.endLat * r) * Math.sin(((t.endLon - t.startLon) * r) / 2) ** 2;
  return 12742 * Math.asin(Math.sqrt(a));
}

// A trip whose recorded distance is drastically short: the start and end alone are much farther apart than the odometer says
// (a road is never shorter than the straight line). Small differences are left alone, and a corrected trip is never flagged.
export const shortOdometer = (t) => !t.open && !t.distanceSrc && straightKm(t) > 1.5 && straightKm(t) > t.distanceKm * 1.3;

// "Does the distance look wrong?" for admins. The odometer sometimes updates late, so a drive can come out shorter than it was. This
// shows a road between the recorded positions and the drive straight from start to end, and lets the owner use one, type the real
// distance, or go back to the odometer's. A drastically short trip asks the routing service straight away.
export function tripDistanceCard(t) {
  if (t.open || t.endTs == null) return null;
  const msg = h('p', { class: 'muted', role: 'status' });
  const out = h('div');
  const run = async (fn) => { try { await fn(); } catch (e) { msg.textContent = e.message; } };
  const send = (body) => run(async () => { await api(`/api/trips/${t.id}/distance`, { method: 'POST', body }); location.reload(); });
  const use = (kind, km, main) => h('button', { class: main ? '' : 'sec', onclick: () => send({ use: kind }) }, `Use ${dist(km, 1)}`);
  const check = () => run(async () => {
    msg.textContent = 'Asking the routing service…';
    const r = await api(`/api/trips/${t.id}/road`);
    msg.textContent = '';
    const both = r.directKm > 0 && Math.abs(r.directKm - r.roadKm) > 0.3;
    const directFirst = both && r.directKm > r.roadKm;
    out.replaceChildren(h('p', {}, `A road through the recorded positions is ${dist(r.roadKm, 1)}${both ? `; driving straight from the start to the end is ${dist(r.directKm, 1)}` : ''}. This trip says ${dist(r.distanceKm, 1)}.`),
      h('p', { class: 'seg' }, directFirst ? use('direct', r.directKm, true) : null, use('road', r.roadKm, !directFirst), both && !directFirst ? use('direct', r.directKm, false) : null));
  });
  const short = shortOdometer(t);
  if (short) setTimeout(check, 0);
  const metric = isMetric();
  const num = h('input', { type: 'number', step: '0.1', min: 0.1, inputmode: 'decimal', placeholder: metric ? 'km' : 'miles', 'aria-label': `Distance in ${metric ? 'kilometers' : 'miles'}`, style: 'width:8em' });
  const enter = () => { const v = Number(num.value); if (!(v > 0)) { msg.textContent = 'Enter the distance.'; return; } send({ km: metric ? v : v * 1.609344 }); };
  return h('section', { class: 'card' }, h('h2', {}, short ? 'This distance looks too short' : 'Does the distance look wrong?'),
    short ? h('p', {}, `The odometer says ${dist(t.distanceKm, 1)}, but the start and end are ${dist(straightKm(t), 1)} apart in a straight line, so the drive was longer. CarTrak may also have noticed the trip late; if you know the real distance, type it below.`) : null,
    h('p', { class: 'muted' }, 'The car\'s odometer sometimes updates late, so a drive can be recorded shorter than it was. You can use the length of the road, or type the real distance. Reports and the mileage log use the corrected figure. Nothing changes until you choose.'),
    h('p', {}, `Distance now: ${dist(t.distanceKm, 1)} `, t.distanceSrc ? chip('work', `corrected: ${SRC[t.distanceSrc] || t.distanceSrc}`) : null),
    h('p', { class: 'seg' }, short ? null : h('button', { class: 'sec', onclick: check }, 'Check the road distance'), t.distanceSrc && t.startOdo != null ? h('button', { class: 'sec', onclick: () => send({ use: 'odometer' }) }, 'Go back to the odometer') : null),
    out, h('div', { class: 'row', style: 'border:0;padding:0' }, num, h('button', { class: 'sec', onclick: enter }, 'Use this distance')), msg);
}
