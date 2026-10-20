import { h, chip } from '../ui.js';
import { api } from '../api.js';

export default async function people(root, _a, session) {
  if (session.me.siteRole !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const list = h('div', { class: 'menu' });
  const msg = h('p', { class: 'muted', role: 'alert' });
  const name = h('input', { id: 'pn', placeholder: 'Name', autocapitalize: 'none', autocomplete: 'off' });
  const pw = h('input', { id: 'pp', type: 'password', placeholder: 'Password (10+ characters)', autocomplete: 'new-password' });
  const role = h('select', { id: 'pr' }, h('option', { value: 'viewer' }, 'Viewer (read only)'), h('option', { value: 'admin' }, 'Admin'));
  const seeMain = h('input', { type: 'checkbox', id: 'psm', checked: true });
  const ownCar = h('input', { type: 'checkbox', id: 'poc' });

  async function draw() {
    const us = await api('/api/users');
    const flag = async (u, body) => { try { await api(`/api/users/${u.id}`, { method: 'PATCH', body }); draw(); } catch (e) { msg.textContent = e.message; } };
    list.replaceChildren(...us.map((u) => h('div', { class: 'row', style: 'padding:12px 16px;flex-wrap:wrap' }, h('span', { class: 'grow' }, u.name), chip('work', u.role),
      u.role === 'admin' ? null : h('label', { class: 'muted', style: 'display:flex;gap:6px;align-items:center' }, h('input', { type: 'checkbox', checked: u.seesFirst, onchange: (e) => flag(u, { seeMain: e.target.checked }) }), 'sees the first car'),
      u.role === 'admin' ? null : h('label', { class: 'muted', style: 'display:flex;gap:6px;align-items:center' }, h('input', { type: 'checkbox', checked: u.ownCar, onchange: (e) => flag(u, { ownCar: e.target.checked }) }), 'may add a car'),
      u.id === session.me.id ? null : h('button', { class: 'danger', onclick: async () => {
        if (!confirm(`Remove ${u.name}?`)) return;
        try { await api(`/api/users/${u.id}`, { method: 'DELETE' }); draw(); } catch (e) { msg.textContent = e.message; }
      } }, 'Remove'))));
  }

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'People')),
    h('p', { class: 'muted' }, 'Viewers can see their cars but cannot change anything. Each person sees only the cars they were given; an administrator can open any car, and its owner is told.'),
    h('section', { class: 'group' }, list),
    h('section', { class: 'card' }, h('h2', {}, 'Add a person'),
      h('label', { for: 'pn' }, 'Name'), name, h('label', { for: 'pp' }, 'Password'), pw, h('label', { for: 'pr' }, 'Access'), role,
      h('label', { class: 'muted', style: 'display:flex;gap:6px;align-items:center' }, seeMain, 'Can see the first car (administrators always can)'),
      h('label', { class: 'muted', style: 'display:flex;gap:6px;align-items:center' }, ownCar, 'May add a car of their own, linked to their own Ford account'),
      h('p', {}, h('button', { onclick: async () => {
        try {
          await api('/api/users', { method: 'POST', body: { name: name.value, password: pw.value, role: role.value, seeMain: seeMain.checked, ownCar: ownCar.checked } });
          msg.textContent = '';
          name.value = '';
          pw.value = '';
          draw();
        } catch (e) { msg.textContent = e.message; }
      } }, 'Add')), msg));
  await draw();
}
