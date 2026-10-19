import { h, when, chip, emptyState } from '../ui.js';
import { api } from '../api.js';

const NAMES = { gap: 'Gap in readings', 'odo-back': 'Odometer went back', overlap: 'Overlapping trips', tiny: 'Very short trip', fast: 'Impossible speed', long: 'Trip never closed' };

// A read-only look for things in the recorded data that seem wrong. It changes nothing.
export default async function datacheck(root, _a, session) {
  const d = await api('/api/datacheck');
  const empty = session.me.role === 'admin' ? await api('/api/trips-empty').catch(() => null) : null;
  const clean = empty && empty.count > 0 ? h('section', { class: 'card' }, h('h2', {}, 'Empty trips'),
    h('p', {}, `${empty.count} finished trip${empty.count === 1 ? '' : 's'} went nowhere: no distance and no real speed, usually from sitting in the car with the ignition on. Trips with a note or in a journey are kept.`),
    h('p', {}, h('button', { class: 'danger', onclick: async (e) => {
      if (!confirm(`Delete ${empty.count} empty trip${empty.count === 1 ? '' : 's'}? This cannot be undone.`)) return;
      e.target.disabled = true;
      await api('/api/trips-empty', { method: 'POST', body: { confirm: 'DELETE' } });
      location.reload();
    } }, `Delete ${empty.count} empty trip${empty.count === 1 ? '' : 's'}`))) : null;
  root.append(...[h('h2', {}, 'Data check'), clean,
    h('p', {}, h('a', { href: '#/repair' }, 'Something wrong with a whole day? Fix a day')),
    h('p', { class: 'muted' }, `CarTrak looked through the last ${d.days} days (${d.readings} odometer readings and ${d.trips} trips) for things that seem wrong. It only looks; it never changes your data. Gaps are usually just the car sleeping.`),
    ...(d.findings.length ? [h('section', { class: 'card' }, ...d.findings.map((f) => h('div', { class: 'row' },
      h('div', { class: 'grow' }, h('b', {}, NAMES[f.kind] || f.kind), ' ', chip(f.level === 'warn' ? 'work' : 'ok', f.level === 'warn' ? 'look at this' : 'fyi'),
        h('div', { class: 'muted' }, f.text), f.tripId ? h('a', { href: `#/trip/${f.tripId}` }, 'Open the trip') : null),
      h('span', { class: 'muted' }, when(f.ts)))))]
      : [emptyState('Nothing looks wrong', 'No gaps, odometer jumps, overlapping or impossible trips in this period.')]].filter(Boolean));
}
