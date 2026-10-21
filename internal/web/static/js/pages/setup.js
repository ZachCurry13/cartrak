import { h } from '../ui.js';
import { api } from '../api.js';
import { setupItems, setupRow } from '../setup.js';

// More, Setup checklist: every suggestion, done or not, and putting the Home card back.
export default async function setup(root, _a, session) {
  const st = await api('/api/checklist');
  const items = setupItems(st, session);
  const done = items.filter((x) => x.done).length;
  const toggle = h('button', { type: 'button', class: 'sec', onclick: async () => {
    const r = await api(`/api/checklist/hide${st.hidden ? '?show=1' : ''}`, { method: 'POST' });
    st.hidden = r.hidden;
    toggle.textContent = st.hidden ? 'Show the card on Home' : 'Hide the card on Home';
  } }, st.hidden ? 'Show the card on Home' : 'Hide the card on Home');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Setup checklist')),
    h('p', { class: 'muted' }, `Suggestions to get more from CarTrak, none of them required. ${done} of ${items.length} done. Swipe one left, or tap ×, to skip it.`),
    h('section', { class: 'card' }, ...items.map((x) => setupRow(x, false, () => dispatchEvent(new Event('hashchange'))))),
    session.me.role === 'admin' ? h('p', { class: 'muted' }, 'Connecting Ford and remote access are under ', h('a', { href: '#/start' }, 'Getting started'), '.') : null,
    h('p', { class: 'seg' }, toggle));
}
