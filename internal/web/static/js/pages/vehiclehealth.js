import { h, chip, ago } from '../ui.js';
import { api } from '../api.js';
import { rowEl, value, format } from '../healthrows.js';
import { spark } from '../spark.js';

const SOURCE = {
  setting: 'chosen in Admin settings',
  garage: "from Ford's vehicle record",
  detected: 'worked out from the data the car reports',
  unknown: 'not known yet',
};

// Everything the report flags as needing a look, as one short list at the top.
function attention(report) {
  const items = [];
  for (const s of report.sections) {
    for (const r of s.rows) if (r.state === 'warn' || r.state === 'bad') items.push({ r, title: s.title });
  }
  if (!items.length) return h('section', { class: 'card' }, h('p', {}, chip('ok', 'All good'), ' ', h('span', { class: 'muted' }, 'Nothing needs attention right now.')));
  return h('section', { class: 'card' }, h('h2', {}, 'Needs a look'),
    ...items.map(({ r, title }) => h('div', { class: 'hrow' }, h('span', { class: 'hl' }, `${title}: ${r.label}`),
      h('span', { class: 'hv' }, h('b', {}, value(r)), ' ', chip(r.state === 'bad' ? 'bad' : 'work', r.state === 'bad' ? 'Act' : 'Check')),
      r.note ? h('span', { class: 'hn muted' }, r.note) : null)));
}

export default async function vehicleHealth(root, _a, session) {
  const d = await api('/api/vehicle-health');
  const v = d.vehicle;
  const typeChip = chip('work', v.label);
  root.append(h('div', { class: 'page-title' }, h('h2', {}, 'Vehicle health'), typeChip));
  if (!d.ready) return root.append(h('p', { class: 'muted' }, 'No readings yet. This page fills in after the first successful sync.'));

  root.append(h('p', { class: 'muted' }, `${d.model || 'Your car'} · reading from ${ago(d.asOf)}. Ford sends some values only when the car wakes up, so older ones say how old they are.`));
  root.append(attention(d.report));
  for (const s of d.report.sections) {
    root.append(h('section', { class: 'card' }, h('h2', {}, s.title), s.intro ? h('p', { class: 'muted' }, s.intro) : null,
      ...s.rows.map(rowEl), ...(s.series || []).map((x) => spark(x, format))));
  }
  root.append(h('p', { class: 'muted' }, `Vehicle type: ${v.label} (${SOURCE[v.source] || ''}). `,
    session.me.role === 'admin' ? h('a', { href: '#/admin' }, 'Wrong? Change it in Settings.') : ''));
}
