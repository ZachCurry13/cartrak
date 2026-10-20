import { h, chip } from '../ui.js';
import { api } from '../api.js';
import { loadCategories } from '../categories.js';

// The labels a trip can have. Personal, Work and Medical are built in; the owner can rename them, give each a colour and a mileage
// rate, and add their own (a charity, a side business, a club).
function row(c, admin, reload) {
  const label = h('input', { value: c.label, maxlength: 24, 'aria-label': 'Name', disabled: !admin });
  const color = h('input', { type: 'color', value: c.color || '#64748b', 'aria-label': 'Colour', disabled: !admin, style: 'min-width:56px;padding:2px' });
  const rate = h('input', { type: 'number', step: '0.001', min: 0, inputmode: 'decimal', value: c.rate == null ? '' : c.rate, placeholder: 'none', 'aria-label': 'Mileage rate per mile', disabled: !admin, style: 'width:7em' });
  const msg = h('span', { class: 'muted', role: 'status' });
  const save = async () => {
    try {
      const body = { label: label.value, color: color.value };
      if (rate.value === '') body.clearRate = true; else body.rate = Number(rate.value);
      await api(`/api/categories/${c.key}`, { method: 'PATCH', body });
      await loadCategories();
      msg.textContent = 'Saved.';
    } catch (e) { msg.textContent = e.message; }
  };
  const remove = async () => {
    if (!confirm(`Remove ${c.label}? Its trips become Personal, and its rules and modes are removed.`)) return;
    try { await api(`/api/categories/${c.key}`, { method: 'DELETE' }); await loadCategories(); reload(); } catch (e) { msg.textContent = e.message; }
  };
  return h('section', { class: 'card' }, h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, c.label), c.builtin ? chip('work', 'built in') : null),
    h('label', {}, 'Name'), label, h('label', {}, 'Colour'), color,
    h('label', {}, 'Mileage rate per mile (optional)'), rate,
    h('p', { class: 'muted' }, 'Used by the Mileage log to put a deduction beside the distance. Work uses the general rate in More, Settings unless you set one here.'),
    admin ? h('p', { class: 'seg' }, h('button', { onclick: save }, 'Save'), c.builtin ? null : h('button', { class: 'danger', onclick: remove }, 'Remove'), msg) : null);
}

export default async function categories(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  const add = h('div');
  root.append(h('p', {}, h('a', { href: '#/rules' }, '← Trip rules')), h('div', { class: 'page-title' }, h('h2', {}, 'Trip categories')),
    h('p', { class: 'muted' }, 'Every trip has one of these labels. Trip rules and modes (vacation, work trip, medical mode) pick the label for you.'), box, add);
  async function load() {
    const cs = await api('/api/categories');
    box.replaceChildren(...cs.map((c) => row(c, admin, load)));
    const name = h('input', { id: 'nc', maxlength: 24, placeholder: 'For example Charity', autocomplete: 'off' });
    const color = h('input', { type: 'color', value: '#7c3aed', 'aria-label': 'Colour', style: 'min-width:56px;padding:2px' });
    const msg = h('p', { class: 'muted', role: 'alert' });
    add.replaceChildren(admin ? h('section', { class: 'card' }, h('h2', {}, 'Add a category'),
      h('label', { for: 'nc' }, 'Name'), name, h('label', {}, 'Colour'), color,
      h('p', {}, h('button', { onclick: async () => {
        try { await api('/api/categories', { method: 'POST', body: { label: name.value, color: color.value } }); await loadCategories(); load(); } catch (e) { msg.textContent = e.message; }
      } }, 'Add')), msg) : '');
  }
  await load();
}
