import { h } from './ui.js';

// Banners shown on the home and More pages: "update available" and "just updated".
export function notices(session) {
  const a = session.about;
  const out = [];
  if (a && a.update && a.update.available) {
    out.push(h('div', { class: 'banner notice' },
      h('b', {}, `Update available: v${a.update.latest}`), ` (you have v${a.version}). `,
      h('a', { href: '#/changelog' }, "See what's new"),
      session.me.role === 'admin' ? h('p', { class: 'muted' }, 'On TrueNAS: Apps → CarTrak → Stop, then Start. It downloads the newest version when it starts. Your data and settings are kept.') : null));
  }
  if (a && a.justUpdated) {
    out.push(h('div', { class: 'banner notice' }, `CarTrak was updated to v${a.version}. `, h('a', { href: '#/changelog' }, "See what's new")));
  }
  return out;
}
