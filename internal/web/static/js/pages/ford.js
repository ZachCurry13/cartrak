import { h, chip, ago, item } from '../ui.js';
import { api } from '../api.js';
import { vehiclePickCard } from '../vehiclepick.js';

export default async function ford(root, _a, session) {
  const admin = session.me.role === 'admin';
  const st = await api('/api/status');
  const p = st.poller;
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Ford connection')),
    h('section', { class: 'card' },
      h('div', {}, chip(p.linked ? 'ok' : 'bad', p.linked ? 'Linked' : 'Not linked'), ' ',
        h('span', { class: 'muted' }, p.linked ? `Last update ${ago(p.lastOk)}` : 'Not connected to Ford yet')),
      p.lastError ? h('p', { class: 'muted' }, `Last error: ${p.lastError}`) : null,
      admin ? h('p', { class: 'seg' },
        st.canLink ? h('a', { class: 'btn', href: '#/connect' }, p.linked ? 'Reconnect Ford' : 'Connect Ford') : null,
        p.linked ? h('button', { class: 'sec', onclick: async (e) => { await api('/api/poll', { method: 'POST' }); e.target.textContent = 'Requested ✓'; } }, 'Sync now') : null) : null),
    admin ? h('section', { class: 'group' }, h('div', { class: 'menu' },
      item('#/admin', 'Ford settings', 'Client ID, secret, redirect address, time zone'),
      item('#/connect', 'Connect and address check', 'Sign in at Ford and check the redirect'))) : null);
  if (admin && p.linked) { const pick = await vehiclePickCard(); if (pick) root.querySelector('.card').after(pick); }
}
