import { h, catChip } from './ui.js';
import { api } from './api.js';
import { categoryList } from './categories.js';

// Change what a trip was for without opening it: the purpose chip on a trip card opens a row of the categories, and a tap on one
// saves it. Everything here stops the tap from also opening the trip (the card is a link). Admins only.
export function catPicker(t, onChange) {
  const msg = h('span', { class: 'muted', role: 'status' });
  const panel = h('div', { class: 'catrow', hidden: true, role: 'group', 'aria-label': 'Change the purpose of this trip', style: 'display:flex;flex-wrap:wrap;gap:6px;margin-top:8px' });
  const button = h('button', { type: 'button', class: 'catbtn', 'aria-haspopup': 'true', 'aria-expanded': 'false', title: 'Change the purpose of this trip',
    style: 'background:none;border:0;padding:0;min-height:44px;cursor:pointer;display:inline-flex;align-items:center;color:inherit' });
  const stop = (e) => { e.preventDefault(); e.stopPropagation(); };
  const draw = () => {
    button.replaceChildren(catChip(t));
    panel.replaceChildren(...categoryList().map((c) => h('button', { type: 'button', class: `sec sm${c.key === t.category ? ' on' : ''}`, style: 'min-height:40px', 'aria-pressed': String(c.key === t.category),
      onclick: (e) => { stop(e); pick(c.key); } }, c.label)), msg);
  };
  async function pick(key) {
    if (key === t.category) { panel.hidden = true; button.setAttribute('aria-expanded', 'false'); return; }
    msg.textContent = 'Saving…';
    try {
      await api(`/api/trips/${t.id}`, { method: 'PATCH', body: { category: key, notes: t.notes || '' } }); // the notes are sent back so they are kept
      t.category = key;
      t.categorySource = 'manual';
      panel.hidden = true;
      button.setAttribute('aria-expanded', 'false');
      if (onChange) onChange(t); else draw();
    } catch (err) { msg.textContent = err.message; }
  }
  button.addEventListener('click', (e) => {
    stop(e);
    panel.hidden = !panel.hidden;
    button.setAttribute('aria-expanded', String(!panel.hidden));
  });
  draw();
  return { button, panel };
}
