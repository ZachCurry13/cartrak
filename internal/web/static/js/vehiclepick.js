import { h, chip } from './ui.js';
import { api } from './api.js';

// "Which vehicle is this?" When the Ford account linked to this car authorized more than one vehicle, choose the one this car
// follows. A vehicle another car already follows is named, so the same one is not followed twice. Nothing shows for one vehicle.
export async function vehiclePickCard() {
  const d = await api('/api/ford/vehicles').catch(() => null);
  if (!d || d.vehicles.length < 2) return null;
  const card = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const draw = (list) => card.replaceChildren(h('h2', {}, 'Which vehicle is this?'),
    h('p', { class: 'muted' }, 'The Ford account linked here shares more than one vehicle. This car follows one of them; add another car (More, Cars) for each of the others.'),
    ...list.map((v) => h('div', { class: 'row' },
      h('span', { class: 'grow', style: 'flex-basis:12em' }, h('b', {}, v.label), v.this ? [' ', chip('ok', 'this car')] : null,
        v.usedBy.length ? h('div', { class: 'muted' }, `Followed by ${v.usedBy.join(', ')}`) : null),
      v.this ? null : h('button', { type: 'button', class: 'sec sm', onclick: async () => {
        try { await api('/api/ford/vehicle', { method: 'POST', body: { vin: v.vin } }); msg.textContent = `This car now follows ${v.label}, from the next reading.`; draw(list.map((x) => ({ ...x, this: x.vin === v.vin }))); } catch (e) { msg.textContent = e.message; }
      } }, 'Follow this one'))), msg);
  draw(d.vehicles);
  return card;
}
