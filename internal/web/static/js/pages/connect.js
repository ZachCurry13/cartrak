import { withCar } from '../car.js';
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
  const red = s.settings.find((x) => x.key === 'FORD_REDIRECT_URL') || {};
  const fixMsg = h('p', { class: 'muted', role: 'status' });
  const here = location.origin; // the https address you are using right now
  // One tap to match whatever is registered on the Ford app: the address alone, with /ford/callback, or http://localhost.
  const choices = [here, `${here}/ford/callback`, 'http://localhost'].map((val) => h('button', {
    class: redirect === val ? '' : 'sec',
    onclick: async () => {
      try { await api('/api/admin/settings', { method: 'PUT', body: { values: { FORD_REDIRECT_URL: val } } }); location.reload(); } catch (e) { fixMsg.textContent = e.message; }
    } }, val + (redirect === val ? ' ✓' : '')));

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

  // Ask Ford which redirect address is registered on this app, and switch to one it accepts.
  const check = h('section', { class: 'card' }, h('h2', {}, 'Ford address check'), h('p', { class: 'muted' }, 'Asking Ford which redirect address is registered on your app…'));
  api('/api/admin/ford/detect-redirect', { method: 'POST' }).then((r) => {
    if (r.changed) { sessionStorage.setItem('redirectSwitched', r.chosen); location.reload(); return; }
    const note = sessionStorage.getItem('redirectSwitched');
    sessionStorage.removeItem('redirectSwitched');
    check.replaceChildren(...[h('h2', {}, 'Ford address check'),
      note ? h('p', {}, `CarTrak switched the redirect to ${note} because Ford accepts it.`) : null,
      ...r.results.map((x) => h('div', { class: 'row' }, h('span', { class: 'grow', style: 'overflow-wrap:anywhere' }, x.uri), chip(x.accepted ? 'ok' : 'bad', x.accepted ? 'registered' : 'not registered'))),
      r.chosen ? h('p', { class: 'muted' }, `In use: ${r.chosen}`)
        : h('div', { class: 'banner' }, 'Ford accepts none of these addresses. In the Ford developer portal, add one under Account Linking Info (the plus sign), exactly as written above, then reload this page.'),
      r.locked ? h('p', { class: 'muted' }, 'The redirect URL is fixed by an environment variable or config.yaml, so CarTrak did not change it.') : null].filter(Boolean));
  }).catch((e) => {
    if (/cannot check/.test(e.message)) { check.remove(); return; }
    check.replaceChildren(h('h2', {}, 'Ford address check'), h('p', { class: 'muted' }, e.message));
  });
  root.append(check);

  const insecure = redirect.startsWith('http://') && !redirect.startsWith('http://localhost') && !redirect.startsWith('http://127.0.0.1');
  root.append(...(insecure ? [h('div', { class: 'banner' }, h('b', {}, 'Ford will refuse this address. '), 'It is plain http, and Ford only accepts https (or http://localhost). Set up remote access first (More, Remote access) and use its https address, or register http://localhost on your Ford app and use the paste step.', h('p', {}, h('a', { class: 'btn', href: '#/remote' }, 'Set up remote access')))] : []),
    h('section', { class: 'card' }, h('h2', {}, '1. Sign in at Ford'),
      h('p', { class: 'muted' }, 'CarTrak will ask Ford to send you back to:'),
      h('p', {}, h('code', { style: 'overflow-wrap:anywhere' }, redirect)),
      red.locked
        ? h('p', { class: 'muted' }, `This value is fixed by ${red.source === 'env' ? 'an environment variable' : 'config.yaml'}. Change it there, then restart.`)
        : h('div', {}, h('p', { class: 'muted' }, 'It must match a redirect URI registered on your Ford app exactly. Tap the one that matches yours:'), h('div', { class: 'seg' }, ...choices), fixMsg),
      h('p', { class: 'muted' }, 'On Ford pages: sign in (use your separate FordPass account), choose the vehicle, then press Consent. Ford sends you back here.'),
      h('a', { class: 'btn', href: withCar('/ford/connect') }, 'Open Ford sign-in')),
    h('section', { class: 'card' }, h('h2', {}, '2. Paste the address Ford sends you to'),
      h('p', { class: 'muted' }, 'If your redirect is CarTrak\'s own address, you are connected automatically and can skip this. If you registered http://localhost, the page that opens will show an error or nothing, and that is expected. Copy the whole address from the browser\'s address bar (it contains code=…) and paste it here. It works once and only within 15 minutes of starting the sign-in.'),
      h('label', { for: 'pasted' }, 'Address from the browser'), paste,
      h('p', {}, finish), msg, done));
}
