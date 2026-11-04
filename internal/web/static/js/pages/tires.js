import { h } from '../ui.js';
import { api } from '../api.js';
import { tireCompareCard } from '../tirecompare.js';
import { forgetTireSets, KIND_LABEL } from '../tiresets.js';

const dateInput = (ts) => { const d = ts ? new Date(ts * 1000) : new Date(), p = (n) => String(n).padStart(2, '0'); return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`; };
const secs = (v) => Math.floor(new Date(`${v}T12:00:00`).getTime() / 1000);
const since = (ts) => new Date(ts * 1000).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });

// Which tires the car wears and since when, and what the winter set does to its range at the same temperature.
export default async function tires(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Tires')),
    h('p', { class: 'muted' }, 'Log each set and the day it went on, and every trip is labelled with the tires it was on. The comparison separates the tires from the weather.'), box);

  async function load() {
    const [d, cmp] = await Promise.all([api('/api/tire-sets'), tireCompareCard()]);
    const sets = [...d.sets].reverse();
    let editing = null;
    const name = h('input', { id: 'tn', maxlength: 60, placeholder: 'For example Winter tires', autocomplete: 'off' });
    const kind = h('select', { id: 'tk' }, ...d.kinds.map((k) => h('option', { value: k }, KIND_LABEL[k] || k)));
    const from = h('input', { id: 'tf', type: 'date', value: dateInput() });
    const note = h('input', { id: 'tt', maxlength: 300, placeholder: 'Optional: brand, wheels, pressure', autocomplete: 'off' });
    const msg = h('p', { class: 'muted', role: 'status' });
    const done = () => { forgetTireSets(); load(); };
    const save = h('button', { onclick: async () => {
      try {
        const body = { name: name.value, kind: kind.value, fromTs: secs(from.value), note: note.value };
        if (editing) await api(`/api/tire-sets/${editing}`, { method: 'PATCH', body }); else await api('/api/tire-sets', { method: 'POST', body });
        done();
      } catch (e) { msg.textContent = e.message; }
    } }, 'Add');
    const quick = (label, nm, k) => h('button', { class: 'sec', type: 'button', onclick: async () => {
      try { await api('/api/tire-sets', { method: 'POST', body: { name: nm, kind: k, fromTs: Math.floor(Date.now() / 1000), note: '' } }); done(); } catch (e) { msg.textContent = e.message; }
    } }, label);
    const edit = (s) => { editing = s.id; name.value = s.name; kind.value = s.kind; from.value = dateInput(s.fromTs); note.value = s.note || ''; save.textContent = 'Save changes'; name.focus(); };
    const remove = async (s) => { if (confirm(`Remove ${s.name}?`)) { await api(`/api/tire-sets/${s.id}`, { method: 'DELETE' }); done(); } };
    const cur = d.current;
    const row = (s) => h('div', { class: 'row' },
      h('div', { class: 'grow' }, h('b', {}, s.name), ' ', h('span', { class: 'chip' }, KIND_LABEL[s.kind] || s.kind), h('div', { class: 'muted' }, `on since ${since(s.fromTs)}${s.note ? ` · ${s.note}` : ''}`)),
      admin ? h('button', { class: 'text sm', type: 'button', onclick: () => edit(s) }, 'Edit') : null,
      admin ? h('button', { class: 'text sm', type: 'button', onclick: () => remove(s) }, 'Remove') : null);
    box.replaceChildren(...[
      h('section', { class: 'card' }, h('h2', {}, 'On the car now'),
        cur ? h('p', {}, h('b', {}, cur.name), ' ', h('span', { class: 'chip' }, KIND_LABEL[cur.kind] || cur.kind), h('span', { class: 'muted' }, ` since ${since(cur.fromTs)}`)) : h('p', { class: 'muted' }, 'No tires logged yet.'),
        admin ? h('div', { class: 'seg' }, quick('Winter tires on today', 'Winter tires', 'winter'), quick('Back to summer tires today', 'Summer tires', 'summer')) : null),
      cmp,
      h('section', { class: 'card' }, h('h2', {}, 'Every set'),
        ...(sets.length ? sets.map(row) : [h('p', { class: 'muted' }, 'Nothing here yet. Add the tires the car is wearing now, with the date they went on (the date can be in the past).')])),
      admin ? h('section', { class: 'card' }, h('h2', {}, 'Add a set'),
        h('label', { for: 'tn' }, 'Name'), name, h('label', { for: 'tk' }, 'Kind'), kind, h('label', { for: 'tf' }, 'Went on'), from, h('label', { for: 'tt' }, 'Note'), note,
        h('p', {}, save), msg,
        h('p', { class: 'muted' }, 'A set stays on until the next one is added. Removing a set puts the one before it back on for those dates.')) : null,
    ].filter(Boolean));
  }
  await load();
}
