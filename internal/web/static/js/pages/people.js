import { h, chip } from '../ui.js';
import { api } from '../api.js';

export default async function people(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const list = h('div', { class: 'menu' });
  const msg = h('p', { class: 'muted', role: 'alert' });
  const name = h('input', { id: 'pn', placeholder: 'Name', autocapitalize: 'none', autocomplete: 'off' });
  const pw = h('input', { id: 'pp', type: 'password', placeholder: 'Password (10+ characters)', autocomplete: 'new-password' });
  const role = h('select', { id: 'pr' }, h('option', { value: 'viewer' }, 'Viewer (read only)'), h('option', { value: 'admin' }, 'Admin'));

  async function draw() {
    const us = await api('/api/users');
    list.replaceChildren(...us.map((u) => h('div', { class: 'row', style: 'padding:12px 16px' }, h('span', { class: 'grow' }, u.name), chip('work', u.role),
      u.id === session.me.id ? null : h('button', { class: 'danger', onclick: async () => {
        if (!confirm(`Remove ${u.name}?`)) return;
        try { await api(`/api/users/${u.id}`, { method: 'DELETE' }); draw(); } catch (e) { msg.textContent = e.message; }
      } }, 'Remove'))));
  }

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'People')),
    h('p', { class: 'muted' }, 'Viewers can see everything but cannot change anything.'),
    h('section', { class: 'group' }, list),
    h('section', { class: 'card' }, h('h2', {}, 'Add a person'),
      h('label', { for: 'pn' }, 'Name'), name, h('label', { for: 'pp' }, 'Password'), pw, h('label', { for: 'pr' }, 'Access'), role,
      h('p', {}, h('button', { onclick: async () => {
        try {
          await api('/api/users', { method: 'POST', body: { name: name.value, password: pw.value, role: role.value } });
          msg.textContent = '';
          name.value = '';
          pw.value = '';
          draw();
        } catch (e) { msg.textContent = e.message; }
      } }, 'Add')), msg));
  await draw();
}
