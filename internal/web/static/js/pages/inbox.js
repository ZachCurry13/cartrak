import { h, chip, emptyState } from '../ui.js';
import { api } from '../api.js';
import { markInboxSeen, inboxSeenBefore } from '../inboxstate.js';

// Where a tap on each kind of notification opens, whatever page it was logged with.
const OPENS = { left_open: '#/live', unlocked: '#/live', low_battery: '#/charging', arrive_place: '#/day', leave_place: '#/day', achievement: '#/trophies', low_disk: '#/status' };

const dayName = (ts) => {
  const d = new Date(ts * 1000), t = new Date();
  const same = (a, b) => a.toDateString() === b.toDateString();
  if (same(d, t)) return 'Today';
  const y = new Date(t); y.setDate(t.getDate() - 1);
  return same(d, y) ? 'Yesterday' : d.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' });
};
const clock = (ts) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });

// What CarTrak has told you lately, about every car you can open, whether or not notifications are on for this device.
export default async function inbox(root, _a, session) {
  const before = inboxSeenBefore(); // what was already read when this page opened: newer rows are marked
  const [{ items }, health] = await Promise.all([api('/api/inbox'), api('/api/health').catch(() => null)]);
  const problem = health && health.overall && health.overall !== 'ok';
  const rows = [];
  let day = '';
  for (const x of items) {
    const dn = dayName(x.ts);
    if (dn !== day) { day = dn; rows.push(h('h3', { style: 'margin:16px 0 6px' }, dn)); }
    const body = [h('div', { class: 'ibt' }, h('b', {}, x.title), x.carName ? chip('work', x.carName) : null, h('span', { class: 'muted grow', style: 'text-align:right' }, clock(x.ts))),
      x.body ? h('div', { class: 'muted' }, x.body) : null].filter(Boolean);
    const cls = `ibrow${x.ts > before ? ' new' : ''}`;
    const url = (OPENS[x.kind] && (!x.url || x.url === '#/' || x.url === '#/analytics' || x.url === '#/backup')) ? OPENS[x.kind] : x.url; // older entries were logged with a weaker page
    rows.push(url ? h('a', { class: cls, href: url }, ...body) : h('div', { class: cls }, ...body));
  }
  root.append(...[h('div', { class: 'page-title' }, h('h2', { class: 'grow' }, 'Notifications'), h('a', { class: 'btn sec sm', href: '#/notifications' }, '⚙ Alert settings')),
    problem ? h('a', { class: 'card cardlink', href: '#/status' }, h('b', {}, 'Something needs a look. '), h('span', { class: 'muted' }, 'Open Status to see what.')) : null,
    ...(rows.length ? rows : [emptyState('Nothing yet', 'Alerts about trips, charging, tires and more appear here as they happen, even if this device has notifications off.')]),
    h('p', { class: 'muted', style: 'margin-top:16px' }, h('a', { href: '#/notifications' }, 'Choose which alerts reach this device'))].filter(Boolean));
  markInboxSeen(items);
}
