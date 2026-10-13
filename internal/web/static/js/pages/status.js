import { h, chip } from '../ui.js';
import { api } from '../api.js';

const LABEL = { ok: ['ok', 'OK'], warn: ['work', 'Check'], bad: ['bad', 'Fix this'], info: ['work', 'Optional'] };

// "Is it working?": plain-language checks with a next step for anything wrong.
export default async function status(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Status')), box);

  async function load() {
    const hd = await api('/api/health');
    const overall = { ok: 'Everything looks good.', warn: 'Working, with something to check.', bad: 'Something needs fixing.' }[hd.overall];
    box.replaceChildren(
      h('p', { class: 'muted' }, overall),
      h('section', { class: 'group' }, h('div', { class: 'menu' }, ...hd.items.map((it) => {
        const [cls, text] = LABEL[it.state] || ['work', it.state];
        return h('div', { class: 'setting' },
          h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, it.title), chip(cls, text)),
          h('p', { class: 'muted' }, it.detail),
          admin && it.href ? h('a', { class: 'btn sec', href: it.href }, it.action || 'Open') : null);
      }))),
      h('p', { class: 'muted' }, 'This page refreshes by itself.'));
  }
  await load();
  const timer = setInterval(() => { if (!document.hidden) load().catch(() => {}); }, 15000);
  return { cleanup: () => clearInterval(timer) };
}
