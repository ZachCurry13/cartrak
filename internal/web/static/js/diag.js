import { h, chip } from './ui.js';
import { api } from './api.js';

// Admin-only: call every Ford endpoint once and show what came back.
// Location and VIN values are already replaced on the server, so the result is safe to copy and share.
export function diagCard() {
  const out = h('div');
  const msg = h('p', { class: 'muted', role: 'status' });
  let last = null;
  const run = h('button', { class: 'sec', onclick: async () => {
    run.disabled = true; msg.textContent = 'Asking Ford…'; out.replaceChildren();
    try {
      last = await api('/api/admin/ford-probe', { method: 'POST' });
      msg.textContent = last.note;
      out.replaceChildren(...Object.entries(last.results).map(([name, r]) => h('div', { class: 'row' },
        h('span', { class: 'grow' }, name),
        r.error ? h('span', { class: 'muted' }, r.error) : h('span', { class: 'muted' }, `${r.bytes} bytes`),
        chip(r.status === 200 ? 'ok' : 'bad', r.status ? `HTTP ${r.status}` : 'no answer'))));
      copy.hidden = false;
    } catch (e) { msg.textContent = e.message; }
    run.disabled = false;
  } }, 'Test Ford connection');
  const copy = h('button', { class: 'sec', hidden: true, onclick: async () => {
    try { await navigator.clipboard.writeText(JSON.stringify(last.results, null, 2)); msg.textContent = 'Copied. Location and VIN values are redacted.'; }
    catch (e) { msg.textContent = 'Could not copy automatically.'; }
  } }, 'Copy responses');
  return h('section', { class: 'card' }, h('h2', {}, 'Ford diagnostics'),
    h('p', { class: 'muted' }, 'Calls each Ford endpoint once (about six requests) and shows whether it answered. "Copy responses" copies the answers with location and VIN values removed, handy for support or bug reports. Tokens are never included.'),
    h('p', { class: 'seg' }, run, copy), msg, out);
}
