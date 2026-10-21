import { h } from './ui.js';
import { api } from './api.js';

// Home: a service visit waiting for an answer ("Was the car serviced at the Dealership?"), with Log it and Not service.
export async function serviceCards(session) {
  if (session.me.role !== 'admin') return [];
  const d = await api('/api/service').catch(() => null);
  if (!d) return [];
  return d.visits.filter((v) => v.status === 'pending').slice(0, 2).map((v) => {
    const card = h('section', { class: 'card' });
    const no = async () => { await api(`/api/service/${v.id}`, { method: 'PATCH', body: { status: 'dismissed' } }).catch(() => null); card.remove(); };
    card.append(h('h2', {}, '🔧 Was the car serviced?'),
      h('p', {}, `It stayed at ${v.placeName} on ${new Date(v.start * 1000).toLocaleDateString([], { weekday: 'long', month: 'short', day: 'numeric' })} for ${Math.round((v.end - v.start) / 60) >= 60 ? `${Math.floor((v.end - v.start) / 3600)} h ${Math.round(((v.end - v.start) % 3600) / 60)} min` : `${Math.round((v.end - v.start) / 60)} min`}.`),
      h('p', { class: 'seg' }, h('a', { class: 'btn', href: `#/service/${v.id}` }, 'Yes, log it'), h('button', { type: 'button', class: 'sec', onclick: no }, 'No')));
    return card;
  });
}
