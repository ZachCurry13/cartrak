import { h, dist, emptyState, chip } from '../ui.js';
import { api } from '../api.js';
import { serviceForm } from '../serviceform.js';

// Service history (#/service), and one visit to answer or change (#/service/12). Visits CarTrak found at a dealership, tire
// shop or repair shop wait here until you say what was done; you can also log one by hand.

const KIND = { dealer: 'Dealership', tire: 'Tire shop', repair: 'Repair shop' };
const day = (ts) => new Date(ts * 1000).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
const stay = (v) => { const m = Math.round((v.end - v.start) / 60); return m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`; };

export default async function service(root, args, session) {
  const admin = session.me.role === 'admin';
  const d = await api('/api/service');
  const cur = d.currency || '$';
  const money = (v) => `${cur}${v.toFixed(2)}`;
  const id = Number(args[0]);
  root.append(h('p', {}, h('a', { href: '#/maintenance' }, '← Service reminders')), h('div', { class: 'page-title' }, h('h2', {}, 'Service history')));
  const visit = id ? d.visits.find((v) => v.id === id) : null;
  if (visit && admin) {
    root.append(h('section', { class: 'card' }, h('h2', {}, visit.status === 'pending' ? `Was the car serviced at ${visit.placeName}?` : `${visit.placeName || 'Service'} · ${day(visit.start)}`),
      h('p', { class: 'muted' }, `${day(visit.start)} · there ${stay(visit)}${visit.odoKm ? ` · ${dist(visit.odoKm, 0)} on the odometer` : ''}`),
      serviceForm({ visit, reminders: d.reminders, cur, onSaved: () => { location.hash = '#/service'; } })));
    return;
  }
  const logged = d.visits.filter((v) => v.status === 'logged'), pending = d.visits.filter((v) => v.status === 'pending');
  const year = new Date().getFullYear();
  const thisYear = logged.filter((v) => new Date(v.start * 1000).getFullYear() === year);
  const spent = thisYear.reduce((a, v) => a + (v.cost || 0), 0);
  const form = h('div', { hidden: true });
  root.append(...[
    pending.length ? h('section', { class: 'card' }, h('h2', {}, 'Waiting for an answer'), ...pending.map((v) => h('a', { class: 'row', href: `#/service/${v.id}` },
      h('span', { class: 'grow' }, h('b', {}, v.placeName), h('div', { class: 'muted' }, `${day(v.start)} · there ${stay(v)}`)), chip('work', 'was it serviced?')))) : null,
    h('section', { class: 'card' }, h('div', { class: 'bigfig' }, h('div', {}, h('b', {}, String(thisYear.length)), h('span', {}, `visits in ${year}`)), h('div', {}, h('b', {}, money(spent)), h('span', {}, `spent in ${year}`))),
      admin ? h('p', { class: 'seg', style: 'margin-top:12px' }, h('button', { type: 'button', class: 'on', onclick: () => { form.hidden = !form.hidden; } }, '+ Log a service')) : null,
      form),
    logged.length ? h('section', { class: 'card' }, h('h2', {}, 'Every visit'), ...logged.map((v) => h(admin ? 'a' : 'div', { class: 'row', ...(admin ? { href: `#/service/${v.id}` } : {}), style: 'align-items:flex-start' },
      h('span', { class: 'grow', style: 'flex-basis:12em' }, h('b', {}, v.items.join(', ')), h('div', { class: 'muted' }, [day(v.start), v.placeName || KIND[v.kind], v.odoKm ? dist(v.odoKm, 0) : null].filter(Boolean).join(' · ')), v.notes ? h('div', { class: 'muted' }, v.notes) : null),
      v.cost != null ? h('b', {}, money(v.cost)) : null)))
      : emptyState('No service logged yet', 'Mark your dealership, tire shop or repair shop under More, Places, and CarTrak asks after a long stop there. Or log one here.'),
  ].filter(Boolean));
  if (admin) form.append(serviceForm({ visit: null, reminders: d.reminders, cur, onSaved: () => dispatchEvent(new Event('hashchange')) }));
}
