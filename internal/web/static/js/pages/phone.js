import { h, chip, ago } from '../ui.js';
import { api } from '../api.js';
import { copyButton, recipes } from '../phonehelp.js';

const EVENT_WORDS = { drive_start: 'Trip started', drive_end: 'Trip ended', ping: 'Check-in' };

export default async function phone(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const list = h('div', { class: 'menu' });
  const recent = h('div', { class: 'menu' });
  const reveal = h('div');
  const msg = h('p', { class: 'muted', role: 'alert' });
  const name = h('input', { id: 'dn', placeholder: 'For example: My phone', maxlength: 40, autocomplete: 'off' });
  const formatBox = h('section', { class: 'card' });
  let address = location.origin;

  async function draw() {
    const d = await api('/api/devices');
    address = d.address || location.origin;
    formatBox.replaceChildren(h('h2', {}, 'Request format'), recipes(address, ''));
    const names = Object.fromEntries(d.devices.map((x) => [x.id, x.name]));
    list.replaceChildren(...(d.devices.length ? d.devices.map((x) => h('div', { class: 'row', style: 'padding:12px 16px' },
      h('span', { class: 'grow' }, h('b', {}, x.name), h('br'),
        h('span', { class: 'muted' }, x.lastSeen ? `Heard from ${ago(x.lastSeen)} · ${x.points} points` : 'Has not sent anything yet')),
      x.name.startsWith('Browser (') ? chip('work', 'Live page') : null,
      h('button', { class: 'danger', onclick: async () => {
        if (!confirm(`Remove ${x.name}? Its token stops working. Points it already sent are kept.`)) return;
        try { await api(`/api/devices/${x.id}`, { method: 'DELETE' }); draw(); } catch (e) { msg.textContent = e.message; }
      } }, 'Remove'))) : [h('p', { class: 'muted', style: 'padding:12px 16px' }, 'No phones yet.')]));
    recent.replaceChildren(...(d.events.length ? d.events.map((e) => h('div', { class: 'row', style: 'padding:10px 16px' },
      h('span', { class: 'grow' }, EVENT_WORDS[e.type] || e.type), h('span', { class: 'muted' }, `${names[e.device] || 'Removed device'} · ${ago(e.ts)}`)))
      : [h('p', { class: 'muted', style: 'padding:12px 16px' }, 'Nothing yet. Events from your phone show here, so you can check an automation fired.')]));
  }

  async function add() {
    msg.textContent = '';
    try {
      const d = await api('/api/devices', { method: 'POST', body: { name: name.value } });
      name.value = '';
      const test = h('button', { class: 'sec', onclick: async () => {
        try {
          await fetch('/api/ingest/event', { method: 'POST', headers: { Authorization: `Bearer ${d.token}`, 'Content-Type': 'application/json' }, body: '{"type":"ping"}' });
          draw();
          test.textContent = 'Sent. See the list above.';
        } catch (e) { test.textContent = 'Could not reach CarTrak'; }
      } }, 'Send a test check-in');
      reveal.replaceChildren(h('section', { class: 'card' }, h('h2', {}, `Token for ${d.name}`),
        h('p', { class: 'banner notice' }, 'Copy this now. It is shown only once and cannot be looked up later. If you lose it, remove the phone and add it again.'),
        h('pre', { class: 'code' }, d.token), h('p', {}, copyButton(d.token, 'Copy token'), ' ', test),
        h('h3', {}, 'What to send'), recipes(address, d.token)));
      draw();
    } catch (e) { msg.textContent = e.message; }
  }

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Phone tracking')),
    h('p', { class: 'muted' }, 'Pair a phone so CarTrak knows the moment a trip starts or ends, and can draw a finer route from its GPS. Ford still decides what counts as a trip, so odometer and battery numbers stay exact. Nothing is sent unless you set up an automation on the phone.'),
    h('section', { class: 'group' }, list),
    h('section', { class: 'card' }, h('h2', {}, 'Add a phone'), h('label', { for: 'dn' }, 'Name'), name,
      h('p', {}, h('button', { onclick: add }, 'Create token')), msg),
    reveal,
    h('section', { class: 'card' }, h('h2', {}, 'Without an automation app'),
      h('p', { class: 'muted' }, 'Open Live on the phone and use Record GPS from this phone. It only works while that page is open and the screen is on.'),
      h('p', {}, h('a', { href: '#/live' }, 'Open Live'))),
    h('div', { class: 'page-title' }, h('h2', {}, 'Recent messages')), h('section', { class: 'group' }, recent),
    formatBox);
  await draw();
}
