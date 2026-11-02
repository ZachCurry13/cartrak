import { h } from './ui.js';
import { api } from './api.js';

const day = (iso) => new Date(`${iso}T12:00:00`).toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' });

// The software version the car is on, typed in by the owner (Ford does not send it), so a change in efficiency or battery size can
// be lined up with an update.
export async function softwareCard(admin) {
  const d = await api('/api/software').catch(() => null);
  if (!d) return null;
  const box = h('section', { class: 'card' });
  const draw = (notes) => {
    const date = h('input', { type: 'date', id: 'sd', value: new Date().toLocaleDateString('en-CA'), 'aria-label': 'First day on this version' });
    const ver = h('input', { id: 'sv', placeholder: 'As the car shows it', maxlength: 40, 'aria-label': 'Software version' });
    const msg = h('span', { class: 'muted', role: 'status' });
    const act = async (path, opts) => { try { draw((await api(path, opts)).notes || []); } catch (e) { msg.textContent = e.message; } };
    box.replaceChildren(h('h2', {}, 'Software version'),
      h('p', { class: 'muted' }, 'Ford does not send the car\'s software version, so write it down when you see it (on the car: Settings, Software Updates). A note lets you line an update up with changes in efficiency or battery size.'),
      ...(notes.length ? notes.map((n, i) => h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, n.version), h('div', { class: 'muted' }, `${i === 0 ? 'Now, since' : 'From'} ${day(n.date)}`)),
        admin ? h('button', { class: 'sec', 'aria-label': `Remove ${n.version}`, onclick: () => act(`/api/software/${n.id}`, { method: 'DELETE' }) }, 'Remove') : null))
        : [h('p', { class: 'muted' }, 'Nothing written down yet.')]),
      admin ? h('div', {}, h('label', { for: 'sv' }, 'Version'), ver, h('label', { for: 'sd' }, 'First day on it'), date,
        h('p', {}, h('button', { onclick: () => act('/api/software', { method: 'POST', body: { date: date.value, version: ver.value } }) }, 'Add'), ' ', msg)) : null);
  };
  draw(d.notes || []);
  return box;
}
