import { h } from './ui.js';
import { api } from './api.js';

const CHOICES = [['', 'Automatic'], ['work', 'Work'], ['personal', 'Personal']];

// "This outing": what the trips from leaving Home until the car is back count as. Automatic follows the rules; Work or Personal
// relabels the trips so far and covers the ones to come until the car is home.
export function outingCard() {
  const box = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });
  async function draw() {
    const o = await api('/api/outing').catch(() => null);
    if (!o) return box.replaceChildren();
    const set = async (category) => {
      msg.textContent = 'Saving…';
      try {
        const r = await api('/api/outing', { method: 'POST', body: { category } });
        msg.textContent = r.relabeled ? `${r.relabeled} trip${r.relabeled === 1 ? '' : 's'} relabeled.` : '';
        draw();
      } catch (e) { msg.textContent = e.message; }
    };
    box.replaceChildren(h('h2', {}, 'This outing'),
      h('p', { class: 'muted' }, o.category ? `Set by you until the car is back at Home${o.trips ? ` (${o.trips} trip${o.trips === 1 ? '' : 's'} so far)` : ''}.` : 'The rules decide. Tap Work or Personal to set the outing from leaving Home until you are back.'),
      h('div', { class: 'seg' }, ...CHOICES.map(([v, t]) => h('button', { class: o.category === v ? 'on' : '', onclick: () => set(v) }, t))), msg);
  }
  draw();
  return box;
}
