import { h, chip, ago, item, group } from '../ui.js';
import { api } from '../api.js';
import { notices } from '../notice.js';

// The More page: a short, grouped menu. Each row opens its own page.
export default async function settings(root, _a, session) {
  const admin = session.me.role === 'admin';
  const [st, remote, users, health, phones] = await Promise.all([
    api('/api/status'),
    admin ? api('/api/admin/remote').catch(() => null) : null,
    admin ? api('/api/users').catch(() => []) : [],
    api('/api/health').catch(() => null),
    admin ? api('/api/devices').catch(() => null) : null,
  ]);
  const p = st.poller;
  const a = session.about;

  const remoteChip = remote && (remote.status.state === 'connected' ? chip('ok', 'On') : remote.enabled ? chip('bad', remote.status.state === 'not-installed' ? 'Unavailable' : 'Connecting') : chip('work', 'Off'));

  root.append(
    h('div', { class: 'profile' },
      h('span', { class: 'avatar', 'aria-hidden': 'true' }, (session.me.name || '?').slice(0, 1).toUpperCase()),
      h('div', {}, h('b', {}, session.me.name), h('span', { class: 'muted' }, `${session.me.role === 'admin' ? 'Admin' : 'Viewer'} · CarTrak v${a.version}`))),
    ...notices(session),
    h('div', { class: 'menu-cols' },
      h('div', {},
        group('Connections',
          item('#/ford', 'Ford connection', p.linked ? `Updated ${ago(p.lastOk)}` : 'Not linked yet', chip(p.linked ? 'ok' : 'bad', p.linked ? 'Linked' : 'Not linked')),
          admin ? item('#/remote', 'Remote access', 'Cloudflare tunnel for use away from home', remoteChip) : null,
          admin ? item('#/phone', 'Phone tracking', 'Automatic trip start and a finer route from your phone', phones && phones.devices.length ? chip('ok', String(phones.devices.length)) : chip('work', 'Optional')) : null),
        group('Trips', item('#/rules', 'Trip rules', 'Label trips work or personal automatically'),
          admin ? item('#/mileage', 'Mileage log', 'Copy a business mileage log by week') : null),
        group('This device', item('#/display', 'Display and data', 'Units, theme, trip times, data use'),
          item('#/notifications', 'Notifications', 'Alerts for charging, tires, trips and more'))),
      h('div', {},
        admin ? group('Admin',
          item('#/people', 'People', `${users.length} ${users.length === 1 ? 'account' : 'accounts'}`),
          item('#/admin', 'Settings', 'Ford credentials, time zone, diagnostics')) : null,
        group('Help',
          item('#/status', 'Status', 'Is everything working?', health ? chip(health.overall === 'ok' ? 'ok' : health.overall === 'bad' ? 'bad' : 'work', health.overall === 'ok' ? 'OK' : health.overall === 'bad' ? 'Fix' : 'Check') : null),
          item('#/start', 'Getting started', 'A short checklist'),
          item('#/about', 'About and updates', a.update && a.update.available ? `Update available: v${a.update.latest}` : "What's new, version, update checks")))),
    h('p', { class: 'signout' }, h('button', { class: 'sec', onclick: async () => { await api('/api/logout', { method: 'POST' }); session.me = null; location.hash = '#/'; location.reload(); } },
      `Sign out ${session.me.name}`)));
}
