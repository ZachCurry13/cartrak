import { h } from './ui.js';
import { api } from './api.js';
import { current, page, icon } from './navlayout.js';
import { openEditor } from './navedit.js';
import { openSearch } from './searchsheet.js';

// The tiles are the person's own choice (Customize, below); everything else is in its natural home: charging prices on Charging,
// trip rules on Trips, alert choices behind the bell, people, cars and backups under Settings and the Admin part of More.
let open = null;

export function openMore(session) {
  if (open) return;
  const close = () => { bg.remove(); open = null; document.removeEventListener('keydown', onKey); };
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  const tile = (key) => { const [, href, label, d] = page(key); return h('a', { class: 'mtile', href, onclick: close }, icon(d), h('span', {}, label)); };
  const row = (d, label, attrs) => h('a', { class: 'mrow', ...attrs }, icon(d), h('span', {}, label));
  const sheet = h('div', { class: 'sheet more', role: 'dialog', 'aria-label': 'More' },
    h('div', { class: 'mhandle', 'aria-hidden': 'true' }),
    h('div', { class: 'mhead' }, h('h2', {}, 'More'), h('button', { class: 'ico', type: 'button', 'aria-label': 'Close', onclick: close }, icon('M6 6l12 12M18 6L6 18'))),
    // the way to any page in one step: type what you are looking for (the logs showed More, then "Everything else", then a hunt)
    h('button', { class: 'msearch', type: 'button', onclick: () => { close(); openSearch(session.me.role === 'admin'); } }, icon('o11,11,6.5|M16 16l4 4'), h('span', {}, 'Find a page or setting')),
    h('div', { class: 'mgrid' }, ...current(session).tiles.map(tile)),
    h('hr', { style: 'border:0;border-top:1px solid var(--line);margin:14px 0 6px' }),
    row('M4 6h16M4 12h16M4 18h10', 'All pages and settings', { href: '#/settings', onclick: close }),
    row('M4 7h10M18 7h2M4 17h2M10 17h10|o16,7,2|o8,17,2', 'Customize the bar and these tiles', { href: '#', onclick: (e) => { e.preventDefault(); close(); openEditor(session, () => dispatchEvent(new CustomEvent('layoutchange'))); } }),
    row('M8 8h8v6a4 4 0 01-8 0zM9 4l1.5 2M15 4l-1.5 2M4 12h4M16 12h4M5 19l3-2M19 19l-3-2', 'Report a problem or idea', { href: 'https://github.com/ZachCurry13/cartrak/issues/new', target: '_blank', rel: 'noopener', onclick: close }));
  const bg = h('div', { class: 'sheetbg bottom' }, sheet);
  bg.addEventListener('click', (e) => { if (e.target === bg) close(); });
  document.addEventListener('keydown', onKey);
  document.body.append(bg);
  open = bg;
}
