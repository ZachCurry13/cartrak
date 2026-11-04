import { h } from './ui.js';
import { api } from './api.js';

const QUICK = [['', 'Automatic'], ['work', 'Work'], ['personal', 'Personal']];

// "This outing": what the trips from leaving Home until the car is back count as. Automatic (the rules decide) is the default; Work,
// Personal or any other category (Medical, Charity, ones you added) relabels the trips so far and covers the ones to come until
// the car is home.
export function outingCard() {
  const box = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });
  async function draw() {
    const [o, cats] = await Promise.all([api('/api/outing').catch(() => null), api('/api/categories').catch(() => [])]);
    if (!o) return box.replaceChildren();
    const label = (key) => (cats.find((c) => c.key === key) || {}).label || key;
    const others = cats.filter((c) => !QUICK.some(([k]) => k === c.key));
    const set = async (category) => {
      msg.textContent = 'Saving…';
      try {
        const r = await api('/api/outing', { method: 'POST', body: { category } });
        msg.textContent = r.relabeled ? `${r.relabeled} trip${r.relabeled === 1 ? '' : 's'} relabeled.` : '';
        draw();
      } catch (e) { msg.textContent = e.message; }
    };
    const more = others.length ? h('select', { 'aria-label': 'Another category', onchange: (e) => { if (e.target.value) set(e.target.value); } },
      h('option', { value: '' }, 'Another category…'), ...others.map((c) => h('option', { value: c.key, selected: o.category === c.key }, c.label))) : null;
    const note = o.category ? `Set to ${label(o.category)} by you until the car is back at Home${o.trips ? ` (${o.trips} trip${o.trips === 1 ? '' : 's'} so far)` : ''}.`
      : 'The rules decide. Tap a category to set the outing from leaving Home until you are back.';
    box.replaceChildren(h('h2', {}, 'This outing'), h('p', { class: 'muted' }, note),
      h('div', { class: 'seg' }, ...QUICK.map(([v, t]) => h('button', { class: o.category === v ? 'on' : '', onclick: () => set(v) }, t))),
      more ? h('div', { style: 'margin-top:var(--sp-2)' }, more) : null,
      h('p', { class: 'muted' }, h('a', { href: '#/categories' }, 'Add or change categories')), msg);
  }
  draw();
  return box;
}
