import { h } from './ui.js';
import { api } from './api.js';

const DAYS = [[1, 'Mon'], [2, 'Tue'], [3, 'Wed'], [4, 'Thu'], [5, 'Fri'], [6, 'Sat'], [7, 'Sun']];
const GROUP = '/api/admin/settings?group=workday';
const KEYS = { on: 'CARTRAK_WORKDAY', days: 'CARTRAK_WORKDAY_DAYS', stay: 'CARTRAK_WORKDAY_STAY' };

// The work day rule on the Trip rules page: the first outing of a work day is work, the rest of the day is personal.
export async function workDayCard() {
  const d = await api(GROUP);
  const val = (k) => (d.settings.find((s) => s.key === k) || {}).value || '';
  const days = new Set((val(KEYS.days) || '1,2,3,4,5').split(/[,\s]+/).filter(Boolean).map(Number));
  const on = h('input', { type: 'checkbox', id: 'wd-on', checked: val(KEYS.on) === 'on' });
  const boxes = DAYS.map(([n, t]) => h('label', { class: 'check' }, h('input', { type: 'checkbox', checked: days.has(n), 'data-day': n }), ` ${t}`));
  const stay = h('input', { type: 'number', min: 0, max: 600, value: val(KEYS.stay) || 45, 'aria-label': 'Minutes at Home', style: 'max-width:6em' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const save = async () => {
    const chosen = boxes.map((b) => b.querySelector('input')).filter((i) => i.checked).map((i) => i.dataset.day).join(',');
    try {
      await api(GROUP, { method: 'PUT', body: { values: { [KEYS.on]: on.checked ? 'on' : '', [KEYS.days]: chosen, [KEYS.stay]: String(stay.value) } } });
      msg.textContent = 'Saved. New trips follow it. Tap Re-apply below to sort the trips already recorded.';
    } catch (e) { msg.textContent = e.message; }
  };
  return h('section', { class: 'card' }, h('h2', {}, 'Work days'),
    h('p', { class: 'muted' }, 'On a work day, everything from the first time the car leaves Home until it is back counts as work, whatever the time. Outings later that day count as personal. For a weekend concert or a happy hour, tap Work under This outing on Home. A place named Home (or Home 2) is needed.'),
    h('p', {}, h('label', { for: 'wd-on' }, on, ' Use the work day rule')),
    h('p', { class: 'seg' }, ...boxes),
    h('p', {}, 'Getting home for lunch and leaving again within ', stay, ' minutes is still the same work day.'),
    h('p', {}, h('button', { onclick: save }, 'Save')), msg);
}
