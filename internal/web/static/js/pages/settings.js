import { h, chip, ago, isMetric, setMetric, getTheme, setTheme } from '../ui.js';
import { api } from '../api.js';
import { notices } from '../notice.js';
import { aboutCard } from '../aboutcard.js';

export default async function settings(root, _a, session) {
  const admin = session.me.role === 'admin';
  const st = await api('/api/status');
  const p = st.poller;
  const units = h('select', {}, h('option', { value: 'imperial' }, 'Miles, °F, psi'), h('option', { value: 'metric' }, 'Kilometres, °C, kPa'));
  units.value = isMetric() ? 'metric' : 'imperial';
  units.addEventListener('change', () => setMetric(units.value === 'metric'));
  const theme = h('select', {}, h('option', { value: 'auto' }, 'Auto (match phone)'), h('option', { value: 'light' }, 'Light'), h('option', { value: 'dark' }, 'Dark'));
  theme.value = getTheme();
  theme.addEventListener('change', () => setTheme(theme.value));

  root.append(h('h2', {}, 'More'), ...notices(session), aboutCard(session, admin),
    h('section', { class: 'card' }, h('h2', {}, 'Ford connection'),
      h('div', {}, chip(p.linked ? 'ok' : 'bad', p.linked ? 'linked' : 'not linked'), ' ', h('span', { class: 'muted' }, `${p.provider} · last update ${ago(p.lastOk)}`)),
      p.lastError ? h('p', { class: 'muted' }, `Last error: ${p.lastError}`) : null,
      admin ? h('p', { class: 'seg' }, st.canLink ? h('a', { class: 'btn', href: '/ford/connect' }, p.linked ? 'Reconnect Ford' : 'Connect Ford') : null,
        h('button', { class: 'sec', onclick: async (e) => { await api('/api/poll', { method: 'POST' }); e.target.textContent = 'Requested ✓'; } }, 'Sync now')) : null),
    h('section', { class: 'card' }, h('h2', {}, 'Display'), h('label', { for: 'un' }, 'Units (this device)'), (units.id = 'un', units), h('label', { for: 'th' }, 'Theme (this device)'), (theme.id = 'th', theme)),
    h('section', { class: 'card' }, h('a', { href: '#/rules' }, 'Trip rules (work / personal) →')),
    admin ? h('section', { class: 'card' }, h('a', { href: '#/admin' }, 'Admin settings (Ford credentials, time zone) →')) : null,
    admin ? await usersCard() : null,
    h('p', { class: 'muted foot' }, 'CarTrak is an independent open-source project and is not affiliated with or endorsed by Ford Motor Company.'),
    h('p', {}, h('button', { class: 'sec', onclick: async () => { await api('/api/logout', { method: 'POST' }); session.me = null; location.hash = '#/'; location.reload(); } },
      `Sign out ${session.me.name}`)));
}

async function usersCard() {
  const box = h('section', { class: 'card' }, h('h2', {}, 'People'));
  const msg = h('p', { class: 'muted', role: 'alert' });
  async function draw() {
    const us = await api('/api/users');
    const name = h('input', { placeholder: 'Name', autocapitalize: 'none' }), pw = h('input', { type: 'password', placeholder: 'Password (10+ characters)', autocomplete: 'new-password' });
    const role = h('select', {}, h('option', { value: 'viewer' }, 'Viewer (read only)'), h('option', { value: 'admin' }, 'Admin'));
    box.replaceChildren(h('h2', {}, 'People'), ...us.map((u) => h('div', { class: 'row' }, h('span', { class: 'grow' }, u.name), chip('work', u.role),
      h('button', { class: 'danger', onclick: async () => { if (confirm(`Remove ${u.name}?`)) { try { await api(`/api/users/${u.id}`, { method: 'DELETE' }); draw(); } catch (e) { msg.textContent = e.message; } } } }, 'Remove'))),
    h('label', {}, 'Add a person'), name, pw, role,
    h('p', {}, h('button', { onclick: async () => {
      try { await api('/api/users', { method: 'POST', body: { name: name.value, password: pw.value, role: role.value } }); msg.textContent = ''; draw(); } catch (e) { msg.textContent = e.message; }
    } }, 'Add')), msg);
  }
  await draw();
  return box;
}
