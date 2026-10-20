import { h, chip, when, ago } from '../ui.js';
import { api } from '../api.js';
import { sharesList } from '../sharecard.js';

// Who has signed in, and which devices are signed in right now. A lost phone can be signed out here.
export default async function security(root, _a, session) {
  if (session.me.siteRole !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const box = h('div');
  const shares = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Security')), box, shares);
  sharesList().then((c) => shares.replaceChildren(c)).catch(() => {});
  async function load() {
    const d = await api('/api/security');
    const where = (x) => [x.agent, x.ip].filter(Boolean).join(' · ');
    box.replaceChildren(
      h('section', { class: 'card' }, h('h2', {}, 'Signed in now'),
        h('p', { class: 'muted' }, 'Sign out a device you no longer use or do not recognise. It has to sign in again.'),
        ...d.sessions.map((x) => h('div', { class: 'row' },
          h('div', { class: 'grow' }, h('b', {}, x.user), ' ', x.current ? chip('ok', 'this device') : null, h('div', { class: 'muted' }, `${where(x)}${x.created ? ` · signed in ${ago(x.created)}` : ''}`)),
          h('button', { class: 'sec', onclick: async () => { await api(`/api/security/sessions/${x.id}`, { method: 'DELETE' }); if (x.current) location.reload(); else load(); } }, x.current ? 'Sign out' : 'Sign out device')))),
      h('section', { class: 'card' }, h('h2', {}, 'Sign-in log'),
        h('p', { class: 'muted' }, 'The last 100 attempts, newest first. A name you do not know, or many failures in a row, is worth a look. Passwords are never stored.'),
        ...(d.signins.length ? d.signins.map((x) => h('div', { class: 'row' },
          h('div', { class: 'grow' }, h('b', {}, x.name || '(empty)'), ' ', chip(x.ok ? 'ok' : 'bad', x.ok ? 'signed in' : 'failed'), h('div', { class: 'muted' }, `${when(x.ts)} · ${where(x)}`)))) : [h('p', { class: 'muted' }, 'Nothing recorded yet.')])));
  }
  await load();
}
