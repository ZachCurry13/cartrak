import { h } from './ui.js';
import { api } from './api.js';
import { startTour } from './tourui.js';

const NS = 'http://www.w3.org/2000/svg';
const icon = (d) => {
  const s = document.createElementNS(NS, 'svg');
  s.setAttribute('viewBox', '0 0 24 24');
  s.setAttribute('aria-hidden', 'true');
  for (const part of d.split('|')) {
    const p = document.createElementNS(NS, part.startsWith('o') ? 'circle' : 'path');
    if (part.startsWith('o')) { const [cx, cy, r] = part.slice(1).split(','); p.setAttribute('cx', cx); p.setAttribute('cy', cy); p.setAttribute('r', r); } else p.setAttribute('d', part);
    s.append(p);
  }
  return s;
};

// What the More sheet offers: a grid of the places people go most, then a few plain rows. Tiles that need an administrator are
// left out for everyone else.
const TILES = [
  ['#/reports', 'Reports', 'M4 20V10M10 20V4M16 20v-7M22 20H2', null],
  ['#/cars', 'Cars', 'M3 13l2-5h14l2 5v5h-3v-2H6v2H3zM7 13h.01M17 13h.01', null],
  ['#/charging-settings', 'Charging prices', 'M13 2L4 14h7l-1 8 9-12h-7l1-8z', 'admin'],
  ['#/places', 'Places', 'M12 21s7-6.2 7-11a7 7 0 10-14 0c0 4.8 7 11 7 11z|o12,10,2.5', null],
  ['#/rules', 'Trip rules', 'M4 6h16M7 12h10M10 18h4', 'admin'],
  ['#/notifications', 'Notifications', 'M6 17V11a6 6 0 0112 0v6l2 2H4zM10 21h4', null],
  ['#/admin', 'Settings', 'o12,12,3.5|M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M18.4 5.6l-2.1 2.1M7.7 16.3l-2.1 2.1', 'admin'],
  ['#/people', 'People', 'o12,8,4|M4 21a8 8 0 0116 0', 'site'],
  ['#/backup', 'Backup', 'M12 3v12M7 10l5 5 5-5M5 21h14', 'admin'],
];

let open = null;

export function openMore(session) {
  if (open) return;
  const role = session.me.role === 'admin', site = session.me.siteRole === 'admin';
  const show = (need) => !need || (need === 'admin' && role) || (need === 'site' && site);
  const close = () => { bg.remove(); open = null; document.removeEventListener('keydown', onKey); };
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  const tile = ([href, label, d]) => h('a', { class: 'mtile', href, onclick: close }, icon(d), h('span', {}, label));
  const row = (d, label, attrs) => h('a', { class: 'mrow', ...attrs }, icon(d), h('span', {}, label));
  const sheet = h('div', { class: 'sheet more', role: 'dialog', 'aria-label': 'More' },
    h('div', { class: 'mhandle', 'aria-hidden': 'true' }),
    h('div', { class: 'mhead' }, h('h2', {}, 'More'), h('button', { class: 'ico', type: 'button', 'aria-label': 'Close', onclick: close }, icon('M6 6l12 12M18 6L6 18'))),
    h('div', { class: 'mgrid' }, ...TILES.filter((t) => show(t[3])).map(tile)),
    h('hr', { style: 'border:0;border-top:1px solid var(--line);margin:14px 0 6px' }),
    row('o12,12,9|M9.5 9.5a2.5 2.5 0 114 2c-.9.6-1.5 1.1-1.5 2.2M12 17.2v.1', 'How to use CarTrak (guided tour)', { href: '#', onclick: (e) => { e.preventDefault(); close(); startTour(session); } }),
    row('M4 6h16M4 12h16M4 18h10', 'Help and quick settings', { href: '#/help', onclick: close }),
    row('M4 6h16M4 12h16M4 18h10', 'Everything else in More', { href: '#/settings', onclick: close }),
    row('M8 8h8v6a4 4 0 01-8 0zM9 4l1.5 2M15 4l-1.5 2M4 12h4M16 12h4M5 19l3-2M19 19l-3-2', 'Report a problem or idea', { href: 'https://github.com/ZachCurry13/cartrak/issues/new', target: '_blank', rel: 'noopener', onclick: close }),
    row('M9 21H5V3h4M16 17l5-5-5-5M21 12H9', 'Sign out', { href: '#', onclick: async (e) => { e.preventDefault(); try { await api('/api/logout', { method: 'POST' }); } catch (err) { /* signed out anyway */ } location.hash = '#/'; location.reload(); } }));
  const bg = h('div', { class: 'sheetbg bottom' }, sheet);
  bg.addEventListener('click', (e) => { if (e.target === bg) close(); });
  document.addEventListener('keydown', onKey);
  document.body.append(bg);
  open = bg;
}
