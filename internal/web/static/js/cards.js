import { h, chip } from './ui.js';

const norm = (s) => String(s).toLowerCase().replace(/_/g, '');

// Items of a list metric (for example doorStatus) from the flattened "extra" map:
// [{value:'CLOSED', vehicledoor:'...'}, ...] with lower-case field names.
export function listItems(extra, metric) {
  const needle = `${norm(metric)}[`;
  const byIdx = {};
  for (const [k, v] of Object.entries(extra || {})) {
    const lk = norm(k);
    const i = lk.indexOf(needle);
    if (i < 0 || (i > 0 && lk[i - 1] !== '.')) continue;
    const rest = lk.slice(i + needle.length);
    const end = rest.indexOf(']');
    if (end < 0) continue;
    const field = rest.slice(end + 1).replace(/^\./, '');
    if (!field) continue;
    (byIdx[rest.slice(0, end)] ||= {})[field] = v;
  }
  return Object.keys(byIdx).sort((a, b) => a - b).map((k) => byIdx[k]);
}

// "dieselParticulateFilterSootLevel" -> "Diesel particulate filter soot level"
export function words(s) {
  const t = String(s).replace(/([a-z])([A-Z])/g, '$1 $2').replace(/_/g, ' ').toLowerCase();
  return t.charAt(0).toUpperCase() + t.slice(1);
}

const doorName = (d) => {
  const side = d.vehicleside && String(d.vehicleside).toUpperCase() !== 'UNKNOWN' ? `${String(d.vehicleside).toLowerCase()} ` : '';
  const door = String(d.vehicledoor || 'door').toLowerCase().replace(/unspecified_/, '').replace(/_/g, ' ');
  return words(`${side}${door}`);
};

const card = (title, ...kids) => h('section', { class: 'card' }, h('h2', {}, title), ...kids);

// Locks, open doors and the hood. Null when the car reported none of these.
export function doorsCard(latest, extra, pickFn) {
  const open = listItems(extra, 'doorStatus').filter((d) => !['CLOSED', 'INVALID', 'UNKNOWN'].includes(String(d.value).toUpperCase()));
  const hood = pickFn(extra, 'hoodStatus');
  const hoodOpen = hood && String(hood).toUpperCase() === 'OPEN';
  const haveDoors = listItems(extra, 'doorStatus').length > 0;
  if (latest.locked == null && !haveDoors && !hood) return null;
  const rows = [];
  if (latest.locked != null) rows.push(h('div', {}, chip(latest.locked ? 'ok' : 'bad', latest.locked ? 'Locked' : 'Unlocked')));
  if (haveDoors || hood) {
    const names = [...open.map(doorName), ...(hoodOpen ? ['Hood'] : [])];
    rows.push(names.length ? h('p', {}, chip('bad', 'Open'), ` ${names.join(', ')}`) : h('p', { class: 'muted' }, 'All doors closed.'));
  }
  return card('Doors and locks', ...rows);
}

// Warning lights and low-pressure alerts. Null when the car reported neither.
export function healthCard(extra) {
  const lights = Object.entries(extra || {}).filter(([k]) => /(^|\.)indicators\.[^.]+\.value$/i.test(k));
  const tireAlerts = listItems(extra, 'tirePressureStatus').filter((t) => t.value && !['NORMAL', 'UNKNOWN', 'UNSUPPORTED'].includes(String(t.value).toUpperCase()));
  const haveTires = listItems(extra, 'tirePressureStatus').length > 0;
  if (!lights.length && !haveTires) return null;
  const on = lights.filter(([, v]) => v === true || String(v).toUpperCase() === 'ON').map(([k]) => words(k.split('.').slice(-2, -1)[0]));
  const problems = [...on, ...tireAlerts.map((t) => `Tire pressure (${words(t.vehiclewheel || 'tire')})`)];
  return card('Health', problems.length
    ? h('div', {}, h('p', {}, chip('bad', `${problems.length} warning${problems.length === 1 ? '' : 's'}`)), h('ul', { style: 'margin:0;padding-left:20px' }, ...problems.map((p) => h('li', {}, p))))
    : h('p', {}, chip('ok', 'No warnings'), h('span', { class: 'muted' }, ' The car reports no warning lights.')));
}
