import { h, chip, ago } from '../ui.js';
import { api } from '../api.js';
import { currentCar, setCar } from '../car.js';

const LEVEL = { owner: 'Owner', viewer: 'Can look', admin: 'Administrator' };

// Switch to a car: remember it, then start the app again so every page loads that car.
export function openCar(id, hash = '#/') {
  setCar(id);
  location.hash = hash;
  location.reload();
}

// Who can open a car: an owner or administrator picks, for each person, no access, look only, or owner (admins only).
async function accessBox(car, session) {
  const rows = await api(`/api/cars/${car.id}/access`);
  const site = session.me.role === 'admin';
  const msg = h('p', { class: 'muted', role: 'status' });
  return h('div', { class: 'card' }, h('b', {}, `Who can open ${car.name}`),
    ...rows.filter((r) => r.userId !== session.me.id || site).map((r) => {
      const sel = h('select', { 'aria-label': `Access for ${r.name}`, onchange: async () => {
        try { await api(`/api/cars/${car.id}/access`, { method: 'PUT', body: { userId: r.userId, level: sel.value } }); msg.textContent = 'Saved.'; } catch (e) { msg.textContent = e.message; sel.value = r.level; }
      } }, ...[['', 'No access'], ['viewer', 'Can look'], ...(site ? [['owner', 'Owner']] : [])].map(([v, t]) => h('option', { value: v, selected: v === r.level }, t)));
      return h('div', { class: 'row' }, h('span', { class: 'grow' }, r.name), sel);
    }), msg);
}

function carCard(car, session, reload) {
  const manage = car.level === 'owner' || session.me.role === 'admin';
  const msg = h('p', { class: 'muted', role: 'status' });
  const extra = h('div');
  const name = h('input', { value: car.name, 'aria-label': 'Name of the car', maxlength: 40 });
  const save = h('button', { onclick: async () => {
    try { await api(`/api/cars/${car.id}`, { method: 'PATCH', body: { name: name.value } }); reload(); } catch (e) { msg.textContent = e.message; }
  } }, 'Save name');
  const rename = h('div', { class: 'row', hidden: true }, name, save);
  const views = Object.entries(car.adminViews || {});
  return h('section', { class: 'card' },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, car.name),
      String(car.id) === currentCar() ? chip('ok', 'Showing') : null, chip('work', car.theirs ? LEVEL.admin : LEVEL[car.level]),
      chip(car.linked ? 'ok' : 'bad', car.linked ? 'Ford linked' : 'Not linked')),
    car.linked && car.lastOk ? h('p', { class: 'muted' }, `Ford last answered ${ago(car.lastOk)}.`) : null,
    car.theirs ? h('p', { class: 'muted' }, 'You are not an owner of this car. You can open it as an administrator, and its owner can see that you did.') : null,
    views.length ? h('p', { class: 'muted' }, `An administrator looked at this car: ${views.map(([n, t]) => `${n} ${ago(t)}`).join(', ')}.`) : null,
    h('p', { class: 'seg' }, String(car.id) === currentCar() ? null : h('button', { onclick: () => openCar(car.id) }, 'Open this car'),
      manage ? h('button', { class: 'sec', onclick: () => { rename.hidden = !rename.hidden; if (!rename.hidden) name.focus(); } }, 'Rename') : null,
      manage ? h('button', { class: 'sec', onclick: async () => { extra.replaceChildren(await accessBox(car, session)); } }, 'Who can see it') : null,
      manage && car.id !== 1 ? h('button', { class: 'danger', onclick: async () => {
        if (!confirm(`Stop following ${car.name}? Its history stays on the disk but disappears from the app.`)) return;
        try { await api(`/api/cars/${car.id}`, { method: 'DELETE' }); if (String(car.id) === currentCar()) setCar(1); reload(true); } catch (e) { msg.textContent = e.message; }
      } }, 'Remove') : null),
    rename, extra, msg);
}

export default async function cars(root, _a, session) {
  const box = h('div');
  const add = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Cars')), box, add);
  async function load(reloadApp) {
    const d = await api('/api/cars');
    if (reloadApp) return location.reload();
    session.cars = d.cars;
    box.replaceChildren(...(d.cars.length ? d.cars.map((c) => carCard(c, session, load)) : [h('p', { class: 'muted' }, 'You have no car yet. Add yours below.')]));
    const nm = h('input', { id: 'newcar', placeholder: 'A name for the car, such as Mustang', maxlength: 40, autocomplete: 'off' });
    const msg = h('p', { class: 'muted', role: 'alert' });
    add.replaceChildren(d.canAdd ? h('section', { class: 'card' }, h('h2', {}, 'Add a car'),
      h('p', { class: 'muted' }, 'Each car is tracked on its own: its own history, settings and alerts. After adding it you connect it to its Ford account.'),
      h('label', { for: 'newcar' }, 'Name'), nm,
      h('p', {}, h('button', { onclick: async () => {
        try { const r = await api('/api/cars', { method: 'POST', body: { name: nm.value } }); openCar(r.id, '#/connect'); } catch (e) { msg.textContent = e.message; }
      } }, 'Add')), msg)
      : h('p', { class: 'muted' }, session.me.role === 'admin' ? `CarTrak follows up to ${d.max} cars.` : 'Ask an administrator if you want to add a car of your own.'));
  }
  await load();
}
