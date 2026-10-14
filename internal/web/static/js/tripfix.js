import { h } from './ui.js';
import { api } from './api.js';

const STUCK_SEC = 30 * 60; // the server refuses to close a trip that readings are still arriving for

// "Fix this trip" for admins: close a trip that never ended, or delete a junk or duplicate one.
export function tripFixCard(t) {
  const msg = h('p', { class: 'muted', role: 'status' });
  const last = t.endTs || t.startTs;
  const stuck = t.open && Date.now() / 1000 - last > STUCK_SEC;
  const run = async (fn) => { try { await fn(); } catch (e) { msg.textContent = e.message; } };
  const close = () => run(async () => { await api(`/api/trips/${t.id}/close`, { method: 'POST' }); location.reload(); });
  const del = () => run(async () => {
    if (!confirm('Delete this trip? Any public link made from it stops working. This cannot be undone.')) return;
    await api(`/api/trips/${t.id}`, { method: 'DELETE' });
    location.hash = '#/trips';
  });
  if (t.open && !stuck) return null;
  return h('section', { class: 'card' }, h('h2', {}, 'Fix this trip'),
    stuck ? h('p', { class: 'muted' }, 'This trip has had no readings for a while and was never closed. Close it to keep what was recorded, or Data check may show it as a very long trip.') : h('p', { class: 'muted' }, 'Delete a duplicate or a trip that is not real. The odometer and battery history are not changed.'),
    h('p', { class: 'seg' }, stuck ? h('button', { onclick: close }, 'Close this trip') : null, !t.open ? h('button', { class: 'sec', onclick: del }, 'Delete this trip') : null), msg);
}
