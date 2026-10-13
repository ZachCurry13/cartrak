import { h, ago } from './ui.js';
import { format } from './healthrows.js';

// One value on the Car data page: its name, the value, how old it is, and a "?" that explains it.
const short = (sec) => (sec < 5400 ? `${Math.round(sec / 60)} min` : sec < 129600 ? `${Math.round(sec / 3600)} h` : `${Math.round(sec / 86400)} d`);

// Ford updates some values only when the car wakes, so say when a value is old compared with the reading.
function oldNote(row, readingTs, isLatest) {
  const t = row.at ? Date.parse(row.at) / 1000 : 0;
  if (!t || readingTs - t <= 7200) return '';
  return isLatest ? `updated ${ago(t)}` : `updated ${short(readingTs - t)} before this reading`;
}

export function dataRow(row, readingTs, isLatest) {
  const val = row.kind === 'text' || row.num == null ? row.text || '—' : format(row.kind, row.num);
  const detail = row.kind !== 'text' && row.text ? row.text : '';
  const old = oldNote(row, readingTs, isLatest);
  const help = row.help || row.raw
    ? h('div', { class: 'drhelp muted', hidden: true }, row.help || '', row.raw ? h('div', { class: 'raw' }, `Ford's name for it: ${row.raw}`) : null)
    : null;
  const why = help ? h('button', { type: 'button', class: 'why', 'aria-label': `What is ${row.label}?`, 'aria-expanded': 'false', onclick: (e) => {
    help.hidden = !help.hidden;
    e.currentTarget.setAttribute('aria-expanded', String(!help.hidden));
  } }, '?') : null;
  return h('div', { class: 'drow' },
    h('span', { class: 'dl' }, row.label, ' ', why),
    h('span', { class: 'dv' }, h('b', {}, val), detail ? h('span', { class: 'muted' }, ` ${detail}`) : null),
    old ? h('span', { class: 'dn muted' }, old) : null,
    help);
}
