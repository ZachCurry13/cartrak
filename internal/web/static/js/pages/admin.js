import { h, chip } from '../ui.js';
import { api } from '../api.js';
import { diagCard } from '../diag.js';

const SOURCE = { env: 'an environment variable', 'config.yaml': 'config.yaml', admin: 'this page', default: 'a default' };

export default async function admin(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  let data = await api('/api/admin/settings');
  const box = h('div');
  const msg = h('p', { class: 'muted', role: 'status' });
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('h2', {}, 'Admin settings'),
    h('p', { class: 'muted' }, 'These are saved in CarTrak\'s database. A value set in an environment variable or config.yaml wins and is shown locked: change it there. The Ford secret is stored on the server and is never shown again.'),
    box, msg, diagCard());

  function draw() {
    const inputs = {};
    const rows = data.settings.map((s) => {
      const el = h('input', {
        id: `f-${s.key}`, type: s.secret ? 'password' : 'text', autocomplete: 'off', autocapitalize: 'none', spellcheck: 'false',
        value: s.secret ? '' : s.value, disabled: s.locked,
        placeholder: s.secret ? (s.isSet ? '•••••••• saved (leave blank to keep)' : '') : (s.key === 'FORD_REDIRECT_URL' ? data.suggestedRedirect : s.placeholder),
      });
      inputs[s.key] = el;
      return h('div', {},
        h('label', { for: `f-${s.key}` }, s.label, ' ', s.locked ? chip('work', `locked: ${SOURCE[s.source]}`) : s.isSet ? chip('ok', s.source === 'default' ? 'default' : 'saved') : chip('bad', 'not set')),
        el, h('div', { class: 'muted', style: 'margin-top:4px' }, s.help),
        s.key === 'FORD_REDIRECT_URL' && !s.locked ? h('p', { class: 'seg', style: 'margin-top:6px' },
          h('button', { class: 'sec', onclick: () => { el.value = 'http://localhost'; } }, 'Use http://localhost'),
          h('button', { class: 'sec', onclick: () => { el.value = data.suggestedRedirect; } }, 'Use this site + /ford/callback'),
          h('button', { class: 'sec', onclick: () => { el.value = data.suggestedRedirect.replace('/ford/callback', ''); } }, 'Use this site only (no path)')) : null,
        s.secret && s.isSet && !s.locked ? h('button', { class: 'sec', style: 'margin-top:6px', onclick: () => clearSecret(s.key) }, 'Remove saved secret') : null);
    });
    box.replaceChildren(h('section', { class: 'card' }, ...rows,
      h('p', { class: 'seg', style: 'margin-top:14px' }, h('button', { onclick: () => save(inputs) }, 'Save'),
        ready() ? h('a', { class: 'btn sec', href: '#/connect' }, 'Connect Ford') : null)));
  }

  const ready = () => ['FORD_CLIENT_ID', 'FORD_CLIENT_SECRET'].every((k) => data.settings.find((s) => s.key === k).isSet);

  async function save(inputs) {
    const values = {};
    for (const s of data.settings) if (!s.locked) values[s.key] = inputs[s.key].value.trim();
    try {
      data = await api('/api/admin/settings', { method: 'PUT', body: { values } });
      msg.textContent = 'Saved ✓. Changes apply right away.';
      draw();
    } catch (e) { msg.textContent = e.message; }
  }

  async function clearSecret(key) {
    if (!confirm('Remove the saved secret? Ford linking will stop working until you enter it again.')) return;
    try { data = await api('/api/admin/settings', { method: 'PUT', body: { clear: [key] } }); msg.textContent = 'Removed.'; draw(); } catch (e) { msg.textContent = e.message; }
  }

  draw();
}
