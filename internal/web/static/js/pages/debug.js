import { h } from '../ui.js';
import { api } from '../api.js';

// A plain-text report to copy and send when something looks wrong. It has no positions, sign-ins, keys or VIN.
export default async function debug(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const hours = h('select', { id: 'dh', 'aria-label': 'How far back' },
    ...[[6, 'Last 6 hours'], [24, 'Last 24 hours'], [48, 'Last 2 days'], [72, 'Last 3 days'], [168, 'Last 7 days'], [336, 'Last 14 days']].map(([v, t]) => h('option', { value: v, selected: v === 24 }, t)));
  const box = h('textarea', { readonly: true, rows: 16, 'aria-label': 'Debug report', style: 'width:100%;box-sizing:border-box;font-family:ui-monospace,Menlo,Consolas,monospace;font-size:.75rem;white-space:pre;overflow:auto' });
  const which = h('select', { id: 'dw', 'aria-label': 'Which car' }, h('option', { value: '' }, 'This car'), h('option', { value: '1' }, 'Every car'));
  const many = session.me.siteRole === 'admin' && (session.cars || []).length > 1;
  const msg = h('p', { class: 'muted', role: 'status' });
  const copy = h('button', { class: 'sec', disabled: true, onclick: async () => { try { await navigator.clipboard.writeText(box.value); msg.textContent = 'Copied'; } catch (e) { box.select(); msg.textContent = 'Select all and copy (the browser blocked the button)'; } } }, 'Copy');
  const save = h('a', { class: 'btn sec', hidden: true, download: 'cartrak-debug-report.txt' }, 'Download .txt');
  async function make() {
    msg.textContent = 'Building the report…';
    try {
      const d = await api(`/api/debug-report?hours=${encodeURIComponent(hours.value)}${which.value ? '&all=1' : ''}`);
      box.value = d.text; copy.disabled = false;
      save.href = URL.createObjectURL(new Blob([d.text], { type: 'text/plain' })); save.hidden = false;
      msg.textContent = `${d.readings} readings. Read it before you send it, and delete anything you would rather not share.`;
    } catch (e) { msg.textContent = e.message; }
  }
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Debug report')),
    h('p', { class: 'muted' }, 'When something looks wrong (a trip that is too long, a false "unlocked", a charge that does not add up), make a report right after, copy it and send it. It lists what the car reported, the trips and charging CarTrak made from it, and the alerts it sent. It leaves out where the car was, sign-ins, keys and the VIN. Choose how far back the problem was.'),
    h('section', { class: 'card' }, h('p', {}, many ? which : null, many ? ' ' : null, hours, ' ', h('button', { onclick: make }, 'Make report')), box, h('p', { class: 'seg' }, copy, save), msg));
}
