import { h } from './ui.js';

// A short "needs attention" strip for the top of Home. Shows the most important problem only.
export function healthStrip(health, admin) {
  if (!health || health.overall === 'ok') return null;
  const it = health.items.find((i) => i.state === 'bad') || health.items.find((i) => i.state === 'warn');
  if (!it) return null;
  return h('div', { class: it.state === 'bad' ? 'banner' : 'banner notice' },
    h('b', {}, `${it.title}: `), it.detail,
    h('p', { class: 'seg' }, admin && it.href ? h('a', { class: 'btn', href: it.href }, it.action || 'Fix') : null,
      h('a', { class: 'btn sec', href: '#/status' }, 'See status')));
}
