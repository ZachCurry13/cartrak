import { h } from './ui.js';
import { api } from './api.js';

const fmt = (d) => new Date(`${d}T12:00:00`).toLocaleDateString([], { month: 'short', day: 'numeric' });
const addDays = (day, n) => { const d = new Date(`${day}T12:00:00`); d.setDate(d.getDate() + n); return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`; };

// Vacation: trips on these days are Personal without changing each one by hand. Rules still apply the rest of the time.
export async function vacationCard(admin) {
  const box = h('section', { class: 'card' });
  async function draw() {
    const d = await api('/api/time-off');
    const from = h('input', { type: 'date', id: 'vf', value: d.today, 'aria-label': 'First day' });
    const to = h('input', { type: 'date', id: 'vt', value: addDays(d.today, 6), 'aria-label': 'Last day' });
    const label = h('input', { id: 'vl', maxlength: 60, placeholder: 'Where (optional)', 'aria-label': 'Label' });
    const msg = h('p', { class: 'muted', role: 'alert' });
    const now = d.items.find((x) => x.from <= d.today && x.to >= d.today);
    box.replaceChildren(...[
      h('h2', {}, 'Vacation'),
      h('p', { class: 'muted' }, 'On vacation, every trip counts as Personal, so you do not have to change them one by one. Trips you set by hand keep what you chose.'),
      now ? h('p', { class: 'banner notice' }, `On vacation until ${fmt(now.to)}${now.label ? ` (${now.label})` : ''}.`) : null,
      ...d.items.map((x) => h('div', { class: 'row' }, h('span', { class: 'grow' }, `${fmt(x.from)} to ${fmt(x.to)}`, x.label ? h('span', { class: 'muted' }, ` · ${x.label}`) : null),
        admin ? h('button', { class: 'sec', onclick: async () => { await api(`/api/time-off/${x.id}`, { method: 'DELETE' }); draw(); } }, x.from <= d.today && x.to >= d.today ? 'End now' : 'Remove') : null)),
      admin ? h('div', {}, h('label', { for: 'vf' }, 'First day'), from, h('label', { for: 'vt' }, 'Last day'), to, label,
        h('p', {}, h('button', { onclick: async () => {
          try { await api('/api/time-off', { method: 'POST', body: { from: from.value, to: to.value, label: label.value } }); draw(); } catch (e) { msg.textContent = e.message; }
        } }, 'Start vacation')), msg) : null,
    ].filter(Boolean));
  }
  await draw();
  return box;
}

// A short note for Home while on vacation, or null.
export async function vacationNote() {
  const d = await api('/api/time-off').catch(() => null);
  const now = d && d.items.find((x) => x.from <= d.today && x.to >= d.today);
  return now ? h('a', { class: 'banner notice', href: '#/rules', style: 'display:block;color:inherit;text-decoration:none' }, `On vacation until ${fmt(now.to)}: trips count as Personal.`) : null;
}
