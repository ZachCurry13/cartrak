import { h, chip, ago, item, group } from '../ui.js';
import { carWords } from '../carwords.js';
import { api } from '../api.js';
import { forgetOffline } from '../offlinestore.js';
import { notices } from '../notice.js';
import { buildIndex, search } from '../searchindex.js';
import { isFuelCar } from '../navlayout.js';

// The More page: a short, grouped menu. Each row opens its own page.
// Signing out takes two taps, a few seconds apart, so a stray tap never does it.
function signOutButton(session) {
  const label = `Sign out ${session.me.name}`;
  let armed = null;
  const b = h('button', { class: 'sec', onclick: async () => {
    if (!armed) {
      b.textContent = 'Tap again to sign out';
      armed = setTimeout(() => { armed = null; b.textContent = label; }, 4000);
      return;
    }
    clearTimeout(armed);
    await api('/api/logout', { method: 'POST' }); await forgetOffline(); session.me = null; location.hash = '#/'; location.reload();
  } }, label);
  return b;
}

export default async function settings(root, _a, session) {
  const W = carWords(session);
  const admin = session.me.role === 'admin';
  const site = session.me.siteRole === 'admin'; // the whole installation: people, sign-ins, remote access
  const [st, remote, users, health, phones] = await Promise.all([
    api('/api/status'),
    site ? api('/api/admin/remote').catch(() => null) : null,
    site ? api('/api/users').catch(() => []) : [],
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
        group('Reports', item('#/analytics', 'Insights', 'Wrapped, everywhere you have been, achievements and charts'), item('#/reports', 'Reports', 'Year in review, statistics, efficiency, mileage and exports'),
          item('#/trophies', 'Trophies', 'Your collection, with what is still to earn'),
          isFuelCar(session) ? item('#/fuel', 'Fuel', 'Fill-ups, price per gallon and the size of the tank') : null),
        !admin && session.me.ownCar ? group('Cars', item('#/cars', 'Cars', `${session.cars.length} ${session.cars.length === 1 ? 'car' : 'cars'}`)) : null,
        admin ? group('Connections',
          item('#/ford', 'Ford connection', p.linked ? `Updated ${ago(p.lastOk)}` : 'Not linked yet', chip(p.linked ? 'ok' : 'bad', p.linked ? 'Linked' : 'Not linked')),
          site ? item('#/remote', 'Remote access', remote && remote.connectorUpdate ? 'A newer Cloudflare connector is available' : 'Cloudflare tunnel for use away from home', remoteChip) : null,
          admin ? item('#/phone', 'Phone tracking', 'Automatic trip start and a finer route from your phone', phones && phones.devices.length ? chip('ok', String(phones.devices.length)) : chip('work', 'Optional')) : null) : null,
        admin ? group('Trips', item('#/plan', 'Road-trip planner', W.plan), item('#/maintenance', 'Maintenance', 'Service reminders by distance or time'), item('#/places', 'Places', 'Name Home, the office and other places'), item('#/insights', 'Efficiency insights', W.insights), item('#/year', 'Year in review', W.year), item('#/history', 'History', W.history), item('#/stats', 'Statistics', W.stats), item('#/time', 'Time', W.time), item('#/parkedwatch', 'Parked watch', 'Bumps and alarms on a parked car (experimental)'), item('#/rules', 'Trip rules', 'Label trips work or personal automatically'),
          admin ? item('#/mileage', 'Mileage log', 'Copy a business mileage log by week') : null) : group('Trips', item('#/plan', 'Road-trip planner', W.plan), item('#/insights', 'Efficiency insights', W.insights), item('#/year', 'Year in review', W.year), item('#/history', 'History', W.history), item('#/stats', 'Statistics', W.stats), item('#/time', 'Time', W.time)),
        group('This device', item('#/display', 'Display and data', 'Units, theme, trip times, data use'),
          item('#/notifications', 'Notifications', (W.fuel ? 'Alerts for fuel, tires, trips and more' : 'Alerts for charging, tires, trips and more')))),
      h('div', {},
        admin ? group('Admin',
          site ? item('#/people', 'People', `${users.length} ${users.length === 1 ? 'account' : 'accounts'}`) : null,
          item('#/cars', 'Cars', `${session.cars.length} ${session.cars.length === 1 ? 'car' : 'cars'}`, session.cars.length > 1 ? chip('work', 'Switch') : null),
          site ? item('#/security', 'Sign-ins and devices', 'Who signed in, from where, and every signed-in device') : null,
          item('#/debug', 'Debug report', 'Copy what the car reported, to send when something looks wrong'),
          item('#/fieldcheck', 'Field check', 'Things to try in the car and on your phone, with one button to copy the results'),
          item('#/backup', 'Backup', 'Download a copy of your data'),
          item('#/admin', 'Settings', 'Ford credentials, time zone, city names, diagnostics')) : null,
        group('Help',
          item('#/status', 'Status', 'Is everything working?', health ? chip(health.overall === 'ok' ? 'ok' : health.overall === 'bad' ? 'bad' : 'work', health.overall === 'ok' ? 'OK' : health.overall === 'bad' ? 'Fix' : 'Check') : null),
          item('#/setup', 'Setup checklist', 'Suggested next steps, all optional'),
          item('#/help', 'Help and tour', 'The tour, a guide to everything, and quick answers'),
          item('#/datacheck', 'Data check', 'Look for gaps and odd trips in your data'),
          admin ? item('#/repair', 'Fix a day', 'Redo a day of trips and charging from the stored readings') : null,
          item('#/about', 'About and updates', a.update && a.update.available ? `Update available: v${a.update.latest}` : "What's new, version, update checks")))),
    h('p', { class: 'signout' }, signOutButton(session)));
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
