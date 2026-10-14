import { h, emptyState, dist, when, duration } from '../ui.js';
import { api } from '../api.js';

// A quick pass over trips CarTrak could not label (no rule matched, and you have not chosen): tap Work or Personal.
export default async function review(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/trips' }, '← Trips')), h('div', { class: 'page-title' }, h('h2', {}, 'Trips to review')), box);

  async function load() {
    const all = await api('/api/trips?limit=200');
    const todo = all.filter((t) => !t.open && t.categorySource === 'default');
    if (!todo.length) return box.replaceChildren(emptyState('All caught up', 'Every trip is labelled Work or Personal.', '#/trips', 'See trips'));
    box.replaceChildren(h('p', { class: 'muted' }, `${todo.length} trip${todo.length === 1 ? '' : 's'} with no label yet. Trip rules (More, Trip rules) can label them for you in future.`),
      ...todo.map((t) => {
        const set = (category) => async () => { await api(`/api/trips/${t.id}`, { method: 'PATCH', body: { category, notes: t.notes || '' } }); load(); };
        return h('section', { class: 'card' },
          h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, dist(t.distanceKm)), h('a', { href: `#/trip/${t.id}` }, 'Open')),
          h('div', { class: 'muted' }, `${t.startPlace || t.endPlace ? `${t.startPlace || '…'} → ${t.endPlace || '…'} · ` : ''}${when(t.startTs, t.tz)} · ${duration(t.startTs, t.endTs)}`),
          admin ? h('p', { class: 'seg', style: 'margin:8px 0 0' }, h('button', { onclick: set('work') }, 'Work'), h('button', { class: 'sec', onclick: set('personal') }, 'Personal')) : null);
      }));
  }
  await load();
}
