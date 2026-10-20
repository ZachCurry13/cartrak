import { h, dist, num } from '../ui.js';
import { api } from '../api.js';
import { withCar } from '../car.js';
import { runControls, standardTab, customTab, download } from '../reportbuilder.js';
import { reportEditor } from '../reporteditor.js';

const tile = (href, title, text, extra) => h('a', { class: 'card', href, style: 'display:block;text-decoration:none;color:inherit;margin:0' },
  h('b', {}, title), h('div', { class: 'muted' }, text), extra ? h('div', { style: 'margin-top:6px' }, extra) : null);
const group = (title, ...tiles) => h('section', {}, h('h3', { style: 'margin:18px 0 8px' }, title), h('div', { class: 'cols' }, ...tiles.filter(Boolean)));
const downloadTile = (href, title, text) => h('a', { class: 'card', href: withCar(href), download: '', style: 'display:block;text-decoration:none;color:inherit;margin:0' }, h('b', {}, title), h('div', { class: 'muted' }, text));

// Everything that adds your driving, charging and costs up. At the top, a report you can download: the usual mileage log, or a
// custom one, for the categories and days you choose.
export default async function reports(root, _a, session) {
  const admin = session.me.role === 'admin';
  const [year, sum, opts] = await Promise.all([api('/api/report/year').catch(() => null), api('/api/charges/summary').catch(() => null), api('/api/reports/options')]);
  const yearLine = year && year.driving ? `${dist(year.odometerKm || year.driving.km, 0)} in ${year.driving.trips} trips so far` : 'Miles, energy and charging for the year';
  const m = sum && sum.thisMonth;
  const monthLine = m && m.kwh > 0 ? `${num(m.kwh, 0, ' kWh')} charged this month${m.cost > 0 ? `, ${sum.currency}${m.cost.toFixed(2)}` : ''}` : 'Cost and energy by month, and by where you charge';

  const builder = h('section', { class: 'card' });
  const controls = runControls(opts.categories);
  let tab = 'standard';
  let editing = null;
  const run = (extra) => {
    const p = controls.params();
    for (const [k, v] of Object.entries(extra)) p.set(k, v);
    download(controls.format(), p);
  };
  async function reload(fresh) {
    if (fresh) Object.assign(opts, await api('/api/reports/options'));
    editing = null;
    draw();
  }
  function draw() {
    if (editing !== null) { builder.replaceChildren(reportEditor(opts, editing || null, (saved) => reload(saved))); return; }
    const tabs = h('div', { class: 'seg' }, ...[['standard', 'Standard report'], ['custom', 'Custom reports']].map(([k, t]) => h('button', { class: tab === k ? 'on' : '', onclick: () => { tab = k; draw(); } }, t)));
    builder.replaceChildren(h('h2', {}, 'Download a report'), tabs,
      tab === 'standard' ? standardTab(opts, run) : customTab(opts, run, admin, () => reload(true), (t) => { editing = t || false; draw(); }),
      h('hr', { style: 'border:0;border-top:1px solid var(--line);margin:14px 0' }), controls.el);
  }
  draw();

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Reports')), builder,
    group(`${year ? year.year : 'This year'} and beyond`,
      tile('#/year', 'Year in review', yearLine, 'Copy a summary to share'),
      tile('#/stats', 'Statistics', 'Bars by day, month and year, and where you charge'),
      tile('#/history', 'History', 'Driving, charging and parked time as one feed'),
      tile('#/time', 'Time', 'Where the car spends its time')),
    group('Driving and efficiency',
      tile('#/insights', 'Efficiency insights', 'What changes your miles per kWh: weather, speed, habits'),
      tile('#/compare', 'Compare two trips', 'Side by side, with what made the difference'),
      tile('#/day', 'Day view', 'A whole day on one map, with a written summary'),
      tile('#/plan', 'Road-trip planner', 'How many charging stops a trip needs')),
    group('Money and business',
      tile('#/charging', 'Charging cost', monthLine),
      admin ? tile('#/mileage', 'Mileage log', 'Business miles by week, ready to copy') : null),
    group('Quick downloads',
      downloadTile('/api/export/charging.csv', 'Charging (spreadsheet)', 'Every session with energy, cost and charger type'),
      admin ? downloadTile('/api/export/all.zip', 'Everything (zip)', 'Trips, charging, places and service reminders') : null));
}
