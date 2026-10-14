import { h } from './ui.js';
import { api } from './api.js';

// Combining trips into a journey from the Trips list (admins). It keeps the picked trips, draws the bar
// with the name box and the Combine button, and tells the page when something changed.
export function tripSelect(onChange, onCombined) {
  const picked = new Set();
  const state = { on: false, shown: [] };
  const name = h('input', { id: 'jn', placeholder: 'Name, for example Work trip', maxlength: 80, 'aria-label': 'Journey name' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const bar = h('section', { class: 'card' });

  const eligible = (t) => !t.open && !t.journeyId;
  const toggle = (t) => { if (picked.has(t.id)) picked.delete(t.id); else picked.add(t.id); onChange(); };

  async function combine() {
    msg.textContent = 'Combining…';
    try {
      const r = await api('/api/journeys', { method: 'POST', body: { name: name.value, tripIds: [...picked] } });
      picked.clear();
      state.on = false;
      onCombined(r.id);
    } catch (e) { msg.textContent = e.message; }
  }

  function paintBar() {
    bar.hidden = !state.on;
    if (!state.on) return;
    bar.replaceChildren(h('p', { class: 'muted' }, 'Tap trips to combine ones that belong together (a road trip, or a day with several stops), or tap exactly two and choose Compare.'),
      name,
      h('p', { class: 'seg' },
        h('button', { onclick: combine, disabled: picked.size < 2 }, `Combine ${picked.size} trip${picked.size === 1 ? '' : 's'}`),
        h('button', { class: 'sec', disabled: picked.size !== 2, onclick: () => { location.hash = `#/compare/${[...picked].join(',')}`; } }, 'Compare 2'),
        h('button', { class: 'sec', onclick: () => { state.shown.filter(eligible).forEach((t) => picked.add(t.id)); onChange(); } }, 'Select all shown'),
        h('button', { class: 'sec', onclick: () => { picked.clear(); state.on = false; onChange(); } }, 'Cancel')), msg);
  }

  return {
    state, picked, bar, paintBar, eligible, toggle,
    button: (admin) => (admin ? h('button', { class: 'sec', onclick: () => { state.on = !state.on; picked.clear(); onChange(); } }, state.on ? 'Done selecting' : 'Select trips') : null),
  };
}
