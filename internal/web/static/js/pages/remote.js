import { h, chip } from '../ui.js';
import { api } from '../api.js';

const STATE = {
  connected: ['ok', 'Connected'], starting: ['work', 'Starting…'], reconnecting: ['bad', 'Reconnecting…'],
  off: ['work', 'Off'], 'not-installed': ['bad', 'Connector not installed'],
};

// Remote access: run the Cloudflare connector inside CarTrak.
export default async function remote(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  let v = await api('/api/admin/remote');
  const msg = h('p', { class: 'muted', role: 'status' });
  const statusBox = h('div');
  const logPre = h('pre', { style: 'white-space:pre-wrap;word-break:break-all;font-size:.75rem;max-height:260px;overflow:auto;margin:6px 0 0' });
  const steps = h('ol', { hidden: true, class: 'muted', style: 'padding-left:20px' });
  const token = h('input', { id: 'rt', type: 'password', autocomplete: 'off', spellcheck: 'false', autocapitalize: 'none' });
  const addr = h('input', { id: 'ra', autocomplete: 'off', autocapitalize: 'none', spellcheck: 'false', placeholder: 'cartrak.example.com' });
  const on = h('input', { id: 'ro', type: 'checkbox' });
  const link = h('p');
  const extras = h('div');

  function paintStatus() {
    const [cls, label] = STATE[v.status.state] || ['work', v.status.state];
    statusBox.replaceChildren(...[chip(cls, label), v.status.message && v.status.state !== 'off' ? h('p', { class: 'muted' }, v.status.message) : null,
      !v.installed ? h('p', { class: 'muted' }, 'This build does not include the Cloudflare connector (cloudflared). The official Docker image does.') : null].filter(Boolean));
    logPre.textContent = v.log.length ? v.log.join('\n') : 'No connector output yet.';
  }

  function paintForm() {
    token.value = '';
    token.disabled = v.locked.token;
    token.placeholder = v.tokenSet ? 'Saved. Paste a new one to replace it' : 'Paste the tunnel token';
    addr.value = v.address;
    addr.disabled = v.locked.address;
    on.checked = v.enabled;
    on.disabled = v.locked.enabled;
    link.replaceChildren(...[v.url ? h('a', { href: v.url, target: '_blank', rel: 'noopener' }, `Open ${v.url}`) : null].filter(Boolean));
    extras.replaceChildren(...[
      v.redirect ? h('div', {}, h('p', { class: 'muted' }, 'Using this address with Ford? Use it as the redirect URL, then register the same address on your Ford app.'),
        h('p', { class: 'seg' }, h('button', { class: 'sec', onclick: useForFord }, 'Use as Ford redirect URL'))) : null,
      v.tokenSet && !v.locked.token ? h('p', {}, h('button', { class: 'sec', onclick: removeToken }, 'Remove saved token and turn off')) : null].filter(Boolean));
  }

  steps.replaceChildren(
    h('li', {}, 'In Cloudflare: Zero Trust → Networks → Tunnels → Create a tunnel → Cloudflared.'),
    h('li', {}, 'Copy the token Cloudflare shows (the long string after --token). You can paste the whole command too.'),
    h('li', {}, `Add a public hostname, for example cartrak.yourdomain.com. Service type HTTP, URL localhost:${v.port}.`),
    h('li', {}, 'Paste the token and the public hostname below, tick Turn on remote access, and press Save & connect.'),
    h('li', {}, 'Optional: add a Cloudflare Access policy to the hostname for a second lock.'));

  const toggleSteps = h('button', { class: 'sec', onclick: () => { steps.hidden = !steps.hidden; toggleSteps.textContent = steps.hidden ? 'Show setup steps' : 'Hide setup steps'; } }, 'Show setup steps');

  async function save() {
    msg.textContent = 'Saving…';
    const body = { enabled: on.checked };
    if (token.value.trim()) body.token = token.value;
    if (!v.locked.address) body.address = addr.value;
    if (v.locked.enabled) delete body.enabled;
    try { v = await api('/api/admin/remote', { method: 'PUT', body }); msg.textContent = on.checked ? 'Saved. Connecting…' : 'Saved.'; paintAll(); } catch (e) { msg.textContent = e.message; }
  }
  async function removeToken() {
    if (!confirm('Remove the saved tunnel token and turn remote access off?')) return;
    try { v = await api('/api/admin/remote', { method: 'PUT', body: { clearToken: true, enabled: false } }); msg.textContent = 'Removed.'; paintAll(); } catch (e) { msg.textContent = e.message; }
  }
  async function useForFord() {
    try {
      await api('/api/admin/settings', { method: 'PUT', body: { values: { FORD_REDIRECT_URL: v.redirect } } });
      msg.replaceChildren(`Ford redirect URL set to ${v.redirect}. Register this same address on your Ford app, then `, h('a', { href: '#/connect' }, 'connect Ford'), '.');
    } catch (e) { msg.textContent = e.message; }
  }
  const paintAll = () => { paintStatus(); paintForm(); };

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')),
    h('section', { class: 'card' }, h('h2', {}, 'Remote access (Cloudflare Tunnel)'), statusBox,
      h('p', { class: 'muted' }, 'Use CarTrak away from home through a free Cloudflare Tunnel, with no router changes. CarTrak runs the connector for you.'),
      h('p', {}, toggleSteps), steps,
      h('label', { for: 'rt' }, 'Tunnel token'), token,
      h('label', { for: 'ra' }, 'Public address'), addr,
      h('label', { for: 'ro', style: 'display:flex;gap:8px;align-items:center;color:var(--text)' }, on, 'Turn on remote access'),
      h('p', {}, h('button', { onclick: save }, 'Save & connect')), msg, link, extras,
      h('details', {}, h('summary', { class: 'muted' }, 'Connector log'), logPre)));
  paintAll();

  const timer = setInterval(async () => {
    if (document.hidden) return;
    try { v = { ...v, ...(await api('/api/admin/remote')) }; paintStatus(); } catch (e) { /* ignore a missed refresh */ }
  }, 4000);
  return { cleanup: () => clearInterval(timer) };
}
