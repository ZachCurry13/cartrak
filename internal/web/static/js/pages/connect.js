import { h, chip } from '../ui.js';
import { api } from '../api.js';

// Two steps: sign in at Ford, then (if Ford sends the browser to a page that
// does not load, such as http://localhost) paste that page's address here.
export default async function connect(root, _a, session) {
  const admin = session.me.role === 'admin';
  const st = await api('/api/status');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('h2', {}, 'Connect Ford'));
  if (!admin) return root.append(h('p', { class: 'muted' }, 'Ask the admin to connect the Ford account.'));
  if (!st.configured) {
    return root.append(h('div', { class: 'banner' }, 'Enter your Ford client ID and secret first. ', h('a', { class: 'btn', href: '#/admin', style: 'margin-top:8px' }, 'Open Admin settings')));
  }
  const s = await api('/api/admin/settings');
  const redirect = (s.settings.find((x) => x.key === 'FORD_REDIRECT_URL') || {}).value || s.suggestedRedirect;

  const paste = h('textarea', { id: 'pasted', rows: 3, placeholder: 'http://localhost/?code=…&state=…', autocapitalize: 'none', spellcheck: 'false',
    style: 'width:100%;border-radius:10px;border:1px solid var(--line);background:var(--card);color:var(--text);padding:10px;font:inherit' });
  const msg = h('p', { role: 'status', class: 'muted' });
  const done = h('div');
  const finish = h('button', { onclick: async () => {
    finish.disabled = true; msg.textContent = 'Finishing…';
    try {
      await api('/api/ford/complete', { method: 'POST', body: { url: paste.value } });
      msg.textContent = '';
      done.replaceChildren(h('div', { class: 'card' }, chip('ok', 'connected'), h('p', {}, 'Ford is linked. The first reading arrives within a minute or two.'),
        h('a', { class: 'btn', href: '#/' }, 'Go Home')));
    } catch (e) { msg.textContent = e.message; }
    finish.disabled = false;
  } }, 'Finish connecting');

  root.append(
    h('section', { class: 'card' }, h('h2', {}, '1. Sign in at Ford'),
      h('p', { class: 'muted' }, 'CarTrak will ask Ford to send you back to:'),
      h('p', {}, h('code', { style: 'overflow-wrap:anywhere' }, redirect)),
      h('p', { class: 'muted' }, 'This must exactly match a redirect URI registered on your Ford app (change it in Admin settings if not). Use your separate FordPass account.'),
      h('a', { class: 'btn', href: '/ford/connect' }, 'Open Ford sign-in')),
    h('section', { class: 'card' }, h('h2', {}, '2. Paste the address Ford sends you to'),
      h('p', { class: 'muted' }, 'If your redirect is CarTrak\'s own address, you are connected automatically and can skip this. If you registered http://localhost, the page that opens will show an error or nothing, and that is expected. Copy the whole address from the browser\'s address bar (it contains code=…) and paste it here. It works once and only within 15 minutes of starting the sign-in.'),
      h('label', { for: 'pasted' }, 'Address from the browser'), paste,
      h('p', {}, finish), msg, done));
}
