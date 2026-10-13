import { h, chip, dist, isMetric } from '../ui.js';
import { api } from '../api.js';

// Service reminders by distance, by time, or both. They count from when they are added or marked done.
const PRESETS = [
  ['Tire rotation', 6000, 180], ['Cabin air filter', null, 730], ['Brake fluid', null, 1095],
  ['12 V battery check', null, 365], ['Wiper blades', null, 365], ['Service visit', 12000, 365],
];
const toKm = (v) => (isMetric() ? v : v / 0.621371);
const fromKm = (km) => Math.round(isMetric() ? km : km * 0.621371);

function left(it) {
  const parts = [];
  if (it.kmLeft != null) parts.push(it.kmLeft > 0 ? `${dist(it.kmLeft, 0)} left` : `${dist(-it.kmLeft, 0)} over`);
  if (it.daysLeft != null) parts.push(it.daysLeft >= 0 ? `${it.daysLeft} day${it.daysLeft === 1 ? '' : 's'} left` : `${-it.daysLeft} day${it.daysLeft === -1 ? '' : 's'} over`);
  return parts.join(' · ') || 'Waiting for an odometer reading';
}

export default async function maintenance(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Maintenance')),
    h('p', { class: 'muted' }, 'Reminders for service by distance, by time or both. Each one counts from when you add it or mark it done, and you get an alert when it comes due (turn on Service due under More, Notifications).'), box);

  async function load() {
    const d = await api('/api/maintenance');
    const unit = isMetric() ? 'km' : 'mi';
    const name = h('input', { id: 'mn', placeholder: 'For example Tire rotation', maxlength: 60 });
    const km = h('input', { id: 'mk', type: 'number', min: 1, inputmode: 'numeric', placeholder: unit === 'mi' ? '6000' : '10000' });
    const days = h('input', { id: 'md', type: 'number', min: 7, inputmode: 'numeric', placeholder: '365' });
    const msg = h('p', { class: 'muted', role: 'status' });
    const preset = ([n, k, dd]) => h('button', { class: 'sec', type: 'button', onclick: () => { name.value = n; km.value = k ? fromKm(k) : ''; days.value = dd || ''; } }, n);
    async function add() {
      try {
        const body = { name: name.value };
        if (km.value) body.everyKm = toKm(Number(km.value));
        if (days.value) body.everyDays = Number(days.value);
        await api('/api/maintenance', { method: 'POST', body });
        load();
      } catch (e) { msg.textContent = e.message; }
    }
    box.replaceChildren(...[
      h('section', { class: 'card' }, h('h2', {}, 'Reminders'),
        ...(d.items.length ? d.items.map((it) => h('div', { class: 'row' },
          h('div', { class: 'grow' }, h('b', {}, it.name), ' ', chip(it.state === 'overdue' ? 'bad' : it.state === 'soon' ? 'work' : 'ok', it.state === 'overdue' ? 'due' : it.state === 'soon' ? 'soon' : 'ok'),
            h('div', { class: 'muted' }, `${left(it)} · every ${[it.everyKm ? dist(it.everyKm, 0) : null, it.everyDays ? `${it.everyDays} days` : null].filter(Boolean).join(' or ')}`)),
          admin ? h('span', { class: 'seg', style: 'margin:0' },
            h('button', { onclick: async () => { await api(`/api/maintenance/${it.id}/done`, { method: 'POST' }); load(); } }, 'Done'),
            h('button', { class: 'sec', onclick: async () => { if (confirm(`Remove the ${it.name} reminder?`)) { await api(`/api/maintenance/${it.id}`, { method: 'DELETE' }); load(); } } }, 'Remove')) : null))
          : [h('p', { class: 'muted' }, 'No reminders yet.')])),
      admin ? h('section', { class: 'card' }, h('h2', {}, 'Add a reminder'),
        h('p', { class: 'seg' }, ...PRESETS.map(preset)),
        h('label', { for: 'mn' }, 'What'), name,
        h('div', { class: 'cols2' }, h('div', {}, h('label', { for: 'mk' }, `Every (${unit}, optional)`), km), h('div', {}, h('label', { for: 'md' }, 'Or every (days, optional)'), days)),
        h('p', {}, h('button', { onclick: add }, 'Add reminder')), msg,
        h('p', { class: 'muted' }, 'The examples are only starting points. Use the intervals in your car\'s service schedule.')) : null].filter(Boolean));
  }
  await load();
}
