import { h, when, chip, emptyState } from '../ui.js';
import { api } from '../api.js';

const NAMES = { gap: 'Gap in readings', 'odo-back': 'Odometer went back', overlap: 'Overlapping trips', tiny: 'Very short trip', fast: 'Impossible speed', long: 'Trip never closed' };

// A read-only look for things in the recorded data that seem wrong. It changes nothing.
export default async function datacheck(root) {
  const d = await api('/api/datacheck');
  root.append(h('h2', {}, 'Data check'),
    h('p', {}, h('a', { href: '#/repair' }, 'Something wrong with a whole day? Fix a day')),
    h('p', { class: 'muted' }, `CarTrak looked through the last ${d.days} days (${d.readings} odometer readings and ${d.trips} trips) for things that seem wrong. It only looks; it never changes your data. Gaps are usually just the car sleeping.`),
    ...(d.findings.length ? [h('section', { class: 'card' }, ...d.findings.map((f) => h('div', { class: 'row' },
      h('div', { class: 'grow' }, h('b', {}, NAMES[f.kind] || f.kind), ' ', chip(f.level === 'warn' ? 'work' : 'ok', f.level === 'warn' ? 'look at this' : 'fyi'),
        h('div', { class: 'muted' }, f.text), f.tripId ? h('a', { href: `#/trip/${f.tripId}` }, 'Open the trip') : null),
      h('span', { class: 'muted' }, when(f.ts)))))]
      : [emptyState('Nothing looks wrong', 'No gaps, odometer jumps, overlapping or impossible trips in this period.')]));
}
