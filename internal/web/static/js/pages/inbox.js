import { h, chip, emptyState } from '../ui.js';
import { api } from '../api.js';
import { markInboxSeen, inboxSeenBefore } from '../inboxstate.js';

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
    rows.push(x.url ? h('a', { class: cls, href: x.url }, ...body) : h('div', { class: cls }, ...body));
  }
  root.append(...[h('div', { class: 'page-title' }, h('h2', {}, 'Notifications')),
    problem ? h('a', { class: 'card', href: '#/status', style: 'display:block;text-decoration:none;color:inherit' }, h('b', {}, 'Something needs a look. '), h('span', { class: 'muted' }, 'Open Status to see what.')) : null,
    ...(rows.length ? rows : [emptyState('Nothing yet', 'Alerts about trips, charging, tires and more appear here as they happen, even if this device has notifications off.')]),
    h('p', { class: 'muted', style: 'margin-top:16px' }, h('a', { href: '#/notifications' }, 'Choose which alerts reach this device'))].filter(Boolean));
  markInboxSeen(items);
}
