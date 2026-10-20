import { h, isMetric } from './ui.js';
import { api } from './api.js';
import { withCar } from './car.js';
import { getEfficiencyUnit } from './energy.js';

const ymd = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const today = new Date();
const PRESETS = [
  ['This month', () => [new Date(today.getFullYear(), today.getMonth(), 1), today]],
  ['Last month', () => [new Date(today.getFullYear(), today.getMonth() - 1, 1), new Date(today.getFullYear(), today.getMonth(), 0)]],
  ['This year', () => [new Date(today.getFullYear(), 0, 1), today]],
  ['Last year', () => [new Date(today.getFullYear() - 1, 0, 1), new Date(today.getFullYear() - 1, 11, 31)]],
];

// Which categories, which days or weeks, and which file: the part of a report that is chosen every time it is run.
function runControls(categories) {
  const all = h('input', { type: 'checkbox', id: 'rc-all', checked: true });
  const boxes = categories.map((c) => ({ c, el: h('input', { type: 'checkbox', id: `rc-${c.key}` }) }));
  const sync = (changed) => {
    if (changed === all && all.checked) boxes.forEach((b) => { b.el.checked = false; });
    else if (changed !== all) all.checked = !boxes.some((b) => b.el.checked);
  };
  all.addEventListener('change', () => sync(all));
  boxes.forEach((b) => b.el.addEventListener('change', () => sync(b.el)));
  const line = (el, text, id) => h('label', { for: id, style: 'display:flex;gap:8px;align-items:center;margin:2px 0' }, el, text);
  const from = h('input', { type: 'date', id: 'rf', value: ymd(new Date(today.getFullYear(), 0, 1)), 'aria-label': 'First day' });
  const to = h('input', { type: 'date', id: 'rt', value: ymd(today), 'aria-label': 'Last day' });
  const weeks = h('input', { id: 'rw', placeholder: 'For example 26-28', 'aria-label': 'Calendar weeks', style: 'width:10em' });
  const year = h('input', { id: 'ry', type: 'number', min: 2000, max: 2100, value: today.getFullYear(), 'aria-label': 'Year', style: 'width:7em' });
  const mode = h('select', { id: 'rm', 'aria-label': 'Choose by' }, h('option', { value: 'dates' }, 'Dates'), h('option', { value: 'weeks' }, 'Calendar week numbers'));
  const dates = h('div', {}, h('p', { class: 'seg' }, ...PRESETS.map(([t, f]) => h('button', { class: 'sec', type: 'button', onclick: () => { const [a, b] = f(); from.value = ymd(a); to.value = ymd(b); } }, t))),
    h('label', { for: 'rf' }, 'First day'), from, h('label', { for: 'rt' }, 'Last day'), to);
  const wk = h('div', { hidden: true }, h('label', { for: 'rw' }, 'Calendar weeks'), weeks, h('label', { for: 'ry' }, 'Year'), year);
  mode.addEventListener('change', () => { dates.hidden = mode.value !== 'dates'; wk.hidden = mode.value !== 'weeks'; });
  const eff = h('select', { id: 'reff', 'aria-label': 'Efficiency shown as' }, ...[['kwh', 'Distance per kWh (mi/kWh, or kWh/100 km)'], ['wh', 'Watt-hours per mile (or km)'], ['pct', 'Distance per percent of battery']].map(([v, t]) => h('option', { value: v }, t)));
  eff.value = getEfficiencyUnit();
  const fmt = h('select', { id: 'rfmt', 'aria-label': 'File type' }, h('option', { value: 'xlsx' }, 'Excel (.xlsx)'), h('option', { value: 'csv' }, 'CSV (.csv)'));
  const el = h('div', {}, h('h3', { style: 'margin:6px 0' }, 'Categories'), line(all, 'All categories', 'rc-all'), ...boxes.map((b) => line(b.el, b.c.label, `rc-${b.c.key}`)),
    h('h3', { style: 'margin:14px 0 6px' }, 'Days'), h('label', { for: 'rm' }, 'Choose by'), mode, dates, wk,
    h('h3', { style: 'margin:14px 0 6px' }, 'File'), fmt,
    h('label', { for: 'reff' }, 'Efficiency shown as'), eff);
  const params = () => {
    const p = new URLSearchParams();
    const picked = boxes.filter((b) => b.el.checked).map((b) => b.c.key);
    p.set('categories', all.checked || !picked.length ? 'all' : picked.join(','));
    if (mode.value === 'weeks') { p.set('weeks', weeks.value.trim()); p.set('year', year.value); } else { p.set('from', from.value); p.set('to', to.value); }
    p.set('units', isMetric() ? 'km' : 'mi');
    p.set('eff', eff.value);
    return p;
  };
  return { el, params, format: () => fmt.value };
}

export const download = (path, params) => { location.href = withCar(`/api/reports/trips.${path}?${params}`); };

// The standard report: the usual columns of a mileage log, any of which can be left out.
function standardTab(opts, run) {
  const boxes = opts.columns.map((c) => ({ c, el: h('input', { type: 'checkbox', id: `sc-${c.key}`, checked: c.default }) }));
  const set = (f) => boxes.forEach((b) => { b.el.checked = f(b.c); });
  return h('div', {}, h('p', { class: 'muted' }, 'The usual mileage log: one row per trip with its value, a summary above it and a totals sheet. Untick any column you do not want.'),
    h('h3', { style: 'margin:6px 0' }, 'Columns'),
    h('p', { class: 'seg' }, h('button', { class: 'sec', type: 'button', onclick: () => set(() => true) }, 'All'), h('button', { class: 'sec', type: 'button', onclick: () => set(() => false) }, 'None'), h('button', { class: 'sec', type: 'button', onclick: () => set((c) => c.default) }, 'Usual')),
    ...boxes.map((b) => h('label', { for: `sc-${b.c.key}`, style: 'display:flex;gap:8px;align-items:center;margin:2px 0' }, b.el, b.c.label)),
    h('p', {}, h('button', { onclick: () => { const cols = boxes.filter((b) => b.el.checked).map((b) => b.c.key); if (!cols.length) return alert('Choose at least one column.'); run({ columns: cols.join(',') }); } }, 'Download the report')));
}

// Custom reports: a saved layout, such as the report an employer asks for, run for any categories and days.
function customTab(opts, run, admin, reload, edit) {
  const desc = h('input', { id: 'fxd', value: 'Customer Visits', maxlength: 60, 'aria-label': 'Description column' });
  const round = h('select', { id: 'fxr', 'aria-label': 'Rounding' }, h('option', { value: 'up' }, 'Round each trip up to the next mile'), h('option', { value: 'nearest' }, 'Round the day to the nearest mile'));
  const fedex = h('section', { class: 'card', style: 'margin:0 0 10px' }, h('b', {}, 'FedEx report'),
    h('div', { class: 'muted' }, 'The mileage log: one row a day with the origin city, the cities the car was parked in, a description and the miles, and the total. Work trips unless you choose other categories below.'),
    h('label', { for: 'fxd' }, 'Description column'), desc, h('label', { for: 'fxr' }, 'Miles'), round,
    h('p', {}, h('button', { onclick: () => run({ template: 'fedex', description: desc.value.trim(), rounding: round.value }) }, 'Run the FedEx report')));
  return h('div', {}, h('p', { class: 'muted' }, 'Each custom report keeps its own layout. Run it for the categories and days chosen below.'), fedex,
    ...(opts.templates.length ? opts.templates.map((t) => h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, t.name), h('div', { class: 'muted' }, `${t.config.columns.length} columns${t.config.fixed && t.config.fixed.length ? `, ${t.config.fixed.length} fixed` : ''}`)),
      h('button', { onclick: () => run({ template: String(t.id) }) }, 'Run'),
      admin ? h('button', { class: 'sec', onclick: () => edit(t) }, 'Edit') : null,
      admin ? h('button', { class: 'danger', onclick: async () => { if (confirm(`Delete ${t.name}?`)) { await api(`/api/reports/templates/${t.id}`, { method: 'DELETE' }); reload(); } } }, 'Delete') : null))
      : [h('p', { class: 'muted' }, admin ? 'No custom reports yet. Make the first one.' : 'No custom reports yet. An administrator can make them.')]),
    admin ? h('p', {}, h('button', { class: 'sec', onclick: () => edit(null) }, 'New custom report')) : null);
}

export { runControls, standardTab, customTab };
