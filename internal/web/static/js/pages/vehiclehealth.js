import { h, chip, ago } from '../ui.js';
import { api, AuthError } from '../api.js';
import { rowEl, value, format } from '../healthrows.js';
import { spark } from '../spark.js';
import { batteryCard, truthCard } from '../batterycard.js';
import { tireCard } from '../tirecard.js';
import { softwareCard } from '../softwarecard.js';

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

// The 12 V battery at rest: the average voltage once the car has sat for half an hour without driving or charging. The reading while the
// car is awake is the charging system's (about 14 V); this is the one that says how the battery itself holds up.
function restingEl(r) {
  if (!r || r.n < 2) return h('p', { class: 'muted' }, 'Resting voltage shows here once the car has sat idle for half an hour or more a few times.');
  const [kind, word] = r.avg >= 12.4 ? ['ok', 'Healthy'] : r.avg >= 12.0 ? ['work', 'Fair'] : ['bad', 'Low'];
  return h('div', { class: 'hrow' }, h('span', { class: 'hl' }, 'Resting voltage'), h('span', { class: 'hv' }, h('b', {}, `${r.avg.toFixed(2)} V`), ' ', chip(kind, word)),
    h('span', { class: 'hn muted' }, `The average of ${r.n} reading${r.n === 1 ? '' : 's'} in the last ${r.spanDays} days, taken after the car sat at least ${r.idleMin} minutes without driving or charging (from ${r.min.toFixed(1)} to ${r.max.toFixed(1)} V). About 12.4 V or more is healthy; under 12 V is low. A shop test is still the better measure.`));
}

// The owner can record that the 12 V battery was tested and is fine, which pauses its warnings and alert.
function testedBox(until, admin) {
  const active = until * 1000 > Date.now();
  const text = active
    ? `Tested and fine. Warnings are paused until ${new Date(until * 1000).toLocaleDateString([], { month: 'short', day: 'numeric' })}.`
    : 'Had this battery tested and it was fine? Pause the 12 V warnings and alert for 90 days.';
  return h('p', {}, h('span', { class: 'muted' }, text), ' ', admin
    ? h('button', { class: 'sec', onclick: async (e) => {
      e.currentTarget.disabled = true;
      await api('/api/health/battery12v-tested', { method: active ? 'DELETE' : 'POST', body: { days: 90 } });
      location.reload();
    } }, active ? 'Resume warnings' : 'It was tested and is fine') : null);
}

export default async function vehicleHealth(root, _a, session) {
  // Say so while loading, so the tab never looks dead; a failure gets a message and a retry.
  const loading = h('p', { class: 'muted', role: 'status' }, 'Loading vehicle health…');
  root.append(h('div', { class: 'page-title' }, h('h2', {}, 'Vehicle health')), loading);
  let d;
  try {
    d = await api('/api/vehicle-health');
  } catch (e) {
    if (e instanceof AuthError) throw e;
    return loading.replaceWith(h('div', { class: 'banner' }, `Could not load vehicle health: ${e.message} `, h('button', { class: 'sec', onclick: () => location.reload() }, 'Try again')));
  }
  loading.remove();
  const v = d.vehicle;
  root.querySelector('.page-title').append(chip('work', v.label));
  if (!d.ready) return root.append(h('p', { class: 'muted' }, 'No readings yet. This page fills in after the first successful sync.'));

  root.append(h('p', { class: 'muted' }, `${d.model || 'Your car'} · reading from ${ago(d.asOf)}. Ford sends some values only when the car wakes, so older ones show their age.`));
  root.append(attention(d.report));
  for (const s of d.report.sections) {
    root.append(h('section', { class: 'card' }, h('h2', {}, s.title), s.intro ? h('p', { class: 'muted' }, s.intro) : null,
      ...s.rows.map(rowEl), s.id === '12v' ? restingEl(d.resting) : null, s.id === '12v' ? testedBox(d.tested12vUntil, session.me.role === 'admin') : null, ...(s.series || []).map((x) => spark(x, format))));
  }
  const bat = await batteryCard();
  if (bat) root.append(bat);
  const tires = await tireCard();
  if (tires) root.append(tires);
  const truth = await truthCard();
  if (truth) root.append(truth);
  const sw = await softwareCard(session.me.role === 'admin');
  if (sw) root.append(sw);
  root.append(h('p', { class: 'muted' }, `Vehicle type: ${v.label} (${SOURCE[v.source] || ''}). `,
    session.me.role === 'admin' ? h('a', { href: '#/admin' }, 'Wrong? Change it in Settings.') : ''));
}
