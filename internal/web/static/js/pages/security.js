import { h, chip, when, ago } from '../ui.js';
import { api } from '../api.js';
import { sharesList } from '../sharecard.js';

// Sign-ins and devices: everyone at a glance, every device signed in right now (a lost phone can be signed out here), and the
// log of sign-in attempts.
export default async function security(root, _a, session) {
  if (session.me.siteRole !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const box = h('div');
  const shares = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Sign-ins and devices')), box, shares);
  sharesList().then((c) => shares.replaceChildren(c)).catch(() => {});
  const active = (ts, now) => (!ts ? 'never used' : now - ts < 600 ? 'active now' : `active ${ago(ts)}`);
  async function load() {
    const d = await api('/api/security');
    const where = (x) => [x.agent, x.ip].filter(Boolean).join(' · ');
    box.replaceChildren(
      h('section', { class: 'card' }, h('h2', {}, 'Everyone'),
        h('p', { class: 'muted' }, 'Each account: the devices signed in now, when it was last used, and its last sign-in.'),
        ...(d.people || []).map((p) => h('div', { class: 'row' },
          h('div', { class: 'grow', style: 'flex-basis:12em' }, h('b', {}, p.name), ' ', chip(p.role === 'admin' ? 'work' : 'info', p.role === 'admin' ? 'admin' : 'viewer'),
            h('div', { class: 'muted' }, [p.devices ? `${p.devices} device${p.devices === 1 ? '' : 's'} signed in · ${active(p.lastActive, d.now)}` : 'not signed in anywhere',
              p.lastSignin ? `last signed in ${when(p.lastSignin)}` : null].filter(Boolean).join(' · '))),
          p.failed ? chip('bad', `${p.failed} failed`) : null))),
      h('section', { class: 'card' }, h('h2', {}, 'Signed in now'),
        h('p', { class: 'muted' }, 'Sign out a device you no longer use or do not recognise. It has to sign in again.'),
        ...d.sessions.map((x) => h('div', { class: 'row' },
          h('div', { class: 'grow', style: 'flex-basis:12em' }, h('b', {}, x.user), ' ', x.current ? chip('ok', 'this device') : null,
            h('div', { class: 'muted' }, [where(x), active(x.lastActive, d.now), x.created ? `signed in ${ago(x.created)}` : null].filter(Boolean).join(' · '))),
          h('button', { class: 'sec', onclick: async () => { await api(`/api/security/sessions/${x.id}`, { method: 'DELETE' }); if (x.current) location.reload(); else load(); } }, x.current ? 'Sign out' : 'Sign out device')))),
      h('section', { class: 'card' }, h('h2', {}, 'Sign-in log'),
        h('p', { class: 'muted' }, 'The last 100 attempts, newest first. A name you do not know, or many failures in a row, is worth a look. Passwords are never stored.'),
        ...(d.signins.length ? d.signins.map((x) => h('div', { class: 'row' },
          h('div', { class: 'grow' }, h('b', {}, x.name || '(empty)'), ' ', chip(x.ok ? 'ok' : 'bad', x.ok ? 'signed in' : 'failed'), h('div', { class: 'muted' }, `${when(x.ts)} · ${where(x)}`)))) : [h('p', { class: 'muted' }, 'Nothing recorded yet.')])));
  }
  await load();
}
