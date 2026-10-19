import { h, chip, ago, item, group } from '../ui.js';
import { api } from '../api.js';
import { notices } from '../notice.js';
import { buildIndex, search } from '../searchindex.js';

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
    finder(admin),
    h('div', { class: 'menu-cols' },
      h('div', {},
        admin ? group('Connections',
          item('#/ford', 'Ford connection', p.linked ? `Updated ${ago(p.lastOk)}` : 'Not linked yet', chip(p.linked ? 'ok' : 'bad', p.linked ? 'Linked' : 'Not linked')),
          admin ? item('#/remote', 'Remote access', remote && remote.connectorUpdate ? 'A newer Cloudflare connector is available' : 'Cloudflare tunnel for use away from home', remoteChip) : null,
          admin ? item('#/phone', 'Phone tracking', 'Automatic trip start and a finer route from your phone', phones && phones.devices.length ? chip('ok', String(phones.devices.length)) : chip('work', 'Optional')) : null) : null,
        admin ? group('Trips', item('#/plan', 'Road-trip planner', 'How many charging stops a trip needs'), item('#/maintenance', 'Maintenance', 'Service reminders by distance or time'), item('#/places', 'Places', 'Name Home, the office and other places'), item('#/insights', 'Efficiency insights', 'What changes your miles per kWh'), item('#/year', 'Year in review', 'Miles, energy and charging for the year'), item('#/history', 'History', 'Driving, charging and parked time as one feed'), item('#/stats', 'Statistics', 'Bars by day, month and year, and where you charge'), item('#/time', 'Time', 'Parked, driving and charging, and where'), item('#/parkedwatch', 'Parked watch', 'Bumps and alarms on a parked car (experimental)'), item('#/rules', 'Trip rules', 'Label trips work or personal automatically'),
          admin ? item('#/mileage', 'Mileage log', 'Copy a business mileage log by week') : null) : group('Trips', item('#/plan', 'Road-trip planner', 'How many charging stops a trip needs'), item('#/insights', 'Efficiency insights', 'What changes your miles per kWh'), item('#/year', 'Year in review', 'Miles, energy and charging for the year'), item('#/history', 'History', 'Driving, charging and parked time as one feed'), item('#/stats', 'Statistics', 'Bars by day, month and year, and where you charge'), item('#/time', 'Time', 'Parked, driving and charging, and where')),
        group('This device', item('#/display', 'Display and data', 'Units, theme, trip times, data use'),
          item('#/notifications', 'Notifications', 'Alerts for charging, tires, trips and more'))),
      h('div', {},
        admin ? group('Admin',
          item('#/people', 'People', `${users.length} ${users.length === 1 ? 'account' : 'accounts'}`),
          item('#/security', 'Security', 'Sign-in log and signed-in devices'),
          item('#/repair', 'Fix a day', 'Redo the trips and charging of a day that was recorded wrongly'),
          item('#/debug', 'Debug report', 'Copy what the car reported, to send when something looks wrong'),
          item('#/backup', 'Backup', 'Download a copy of your data'),
          item('#/admin', 'Settings', 'Ford credentials, time zone, prices, diagnostics')) : null,
        group('Help',
          item('#/status', 'Status', 'Is everything working?', health ? chip(health.overall === 'ok' ? 'ok' : health.overall === 'bad' ? 'bad' : 'work', health.overall === 'ok' ? 'OK' : health.overall === 'bad' ? 'Fix' : 'Check') : null),
          item('#/datacheck', 'Data check', 'Look for gaps and odd trips in your data'),
          item('#/guide', 'What CarTrak can do', 'Everything, and where to find it'),
          admin ? item('#/start', 'Getting started', 'A short checklist') : null,
          item('#/about', 'About and updates', a.update && a.update.available ? `Update available: v${a.update.latest}` : "What's new, version, update checks")))),
    h('p', { class: 'signout' }, h('button', { class: 'sec', onclick: async () => { await api('/api/logout', { method: 'POST' }); session.me = null; location.hash = '#/'; location.reload(); } },
      `Sign out ${session.me.name}`)));
}

// finder is a search box. It suggests where a thing is, by name or by what it does ("vacation" finds Trip rules), and
// also hides menu rows that do not match what was typed.
function finder(admin) {
  const none = h('p', { class: 'muted', hidden: true }, 'Nothing matches. Try a different word.');
  const box = h('input', { type: 'search', id: 'mfind', placeholder: 'Search settings and pages', 'aria-label': 'Search settings and pages', autocomplete: 'off' });
  const sugg = h('div', { class: 'sugg', role: 'listbox', hidden: true });
  let index = null;
  const run = async () => {
    const q = box.value.trim();
    if (!index) index = await buildIndex(admin);
    const hits = q ? search(index, q) : [];
    sugg.hidden = !hits.length;
    sugg.replaceChildren(...hits.map((e) => h('a', { class: 'sg', href: e.href, role: 'option' },
      h('b', {}, e.title), h('span', { class: 'muted' }, e.where), e.text ? h('span', { class: 'muted sgtext' }, e.text) : null)));
    let shown = 0;
    const low = q.toLowerCase();
    for (const g of document.querySelectorAll('.menu-cols .group')) {
      let any = 0;
      for (const it of g.querySelectorAll('.item')) {
        const hit = !low || it.textContent.toLowerCase().includes(low);
        it.hidden = !hit;
        if (hit) any++;
      }
      g.hidden = !any;
      shown += any;
    }
    none.hidden = shown > 0 || hits.length > 0;
  };
  box.addEventListener('input', run);
  return h('div', { class: 'finder' }, box, sugg, none);
}
