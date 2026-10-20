import { h, chip, dist, isMetric } from './ui.js';
import { api } from './api.js';

const SRC = { road: 'set from the estimated road', manual: 'entered by you' };

// "Does the distance look wrong?" for admins. The odometer sometimes updates late, so a drive can come out shorter than it was. This
// shows a road between the recorded positions, and lets the owner use it, type the real distance, or go back to the odometer's.
export function tripDistanceCard(t) {
  if (t.open || t.endTs == null) return null;
  const msg = h('p', { class: 'muted', role: 'status' });
  const out = h('div');
  const run = async (fn) => { try { await fn(); } catch (e) { msg.textContent = e.message; } };
  const send = (body) => run(async () => { await api(`/api/trips/${t.id}/distance`, { method: 'POST', body }); location.reload(); });
  const check = () => run(async () => {
    msg.textContent = 'Asking the routing service…';
    const r = await api(`/api/trips/${t.id}/road`);
    msg.textContent = '';
    out.replaceChildren(h('p', {}, `A road between the recorded positions is ${dist(r.roadKm, 1)}. This trip says ${dist(r.distanceKm, 1)}.`),
      h('p', { class: 'seg' }, h('button', { onclick: () => send({ use: 'road' }) }, `Use ${dist(r.roadKm, 1)}`)));
  });
  const metric = isMetric();
  const num = h('input', { type: 'number', step: '0.1', min: 0.1, inputmode: 'decimal', placeholder: metric ? 'km' : 'miles', 'aria-label': `Distance in ${metric ? 'kilometres' : 'miles'}`, style: 'width:8em' });
  const enter = () => { const v = Number(num.value); if (!(v > 0)) { msg.textContent = 'Enter the distance.'; return; } send({ km: metric ? v : v * 1.609344 }); };
  return h('section', { class: 'card' }, h('h2', {}, 'Does the distance look wrong?'),
    h('p', { class: 'muted' }, 'The car\'s odometer sometimes updates late, so a drive can be recorded shorter than it was. You can use the length of the road between the recorded positions, or type the real distance. Reports and the mileage log use the corrected figure.'),
    h('p', {}, `Distance now: ${dist(t.distanceKm, 1)} `, t.distanceSrc ? chip('work', `corrected: ${SRC[t.distanceSrc] || t.distanceSrc}`) : null),
    h('p', { class: 'seg' }, h('button', { class: 'sec', onclick: check }, 'Check the road distance'), t.distanceSrc && t.startOdo != null ? h('button', { class: 'sec', onclick: () => send({ use: 'odometer' }) }, 'Go back to the odometer') : null),
    out, h('div', { class: 'row', style: 'border:0;padding:0' }, num, h('button', { class: 'sec', onclick: enter }, 'Use this distance')), msg);
}
