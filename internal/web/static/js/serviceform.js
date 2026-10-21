import { h, isMetric } from './ui.js';
import { api } from './api.js';

// The form for a service visit: what was done (tap the usual jobs, or write your own), what it cost, and when each job is due
// again. Due again fills in from the reminder of the same name, or a common interval; leave it empty for no reminder.

// [name, every miles, every months]: the usual intervals, as a starting point only
export const JOBS = [
  ['Tire rotation', 7500, null], ['New tires', 40000, null], ['Wheel alignment', null, 12], ['Brakes', null, null], ['Brake fluid', null, 36],
  ['Cabin air filter', 20000, 24], ['Wiper blades', null, 12], ['12 V battery', null, 60], ['Coolant', null, null], ['Software update', null, null],
  ['Recall', null, null], ['Inspection', null, 12], ['Repair', null, null], ['Car wash', null, null],
];

const KM = 1.609344;

export function serviceForm({ visit, reminders, cur, onSaved }) {
  const chosen = new Set(visit && visit.items ? visit.items : []);
  const custom = h('input', { placeholder: 'Something else…', maxlength: 60, 'aria-label': 'Another job' });
  const cost = h('input', { type: 'number', step: '0.01', min: 0, inputmode: 'decimal', placeholder: `${cur} total`, value: visit && visit.cost != null ? visit.cost : '', style: 'width:9em', 'aria-label': 'What it cost' });
  const place = visit ? null : h('input', { placeholder: 'Where (optional)', maxlength: 60, 'aria-label': 'Where' });
  const notes = h('textarea', { rows: 2, maxlength: 500, placeholder: 'Notes (optional)', 'aria-label': 'Notes', style: 'width:100%;box-sizing:border-box' }, visit ? visit.notes || '' : '');
  const dueBox = h('div');
  const msg = h('p', { class: 'muted', role: 'status' });
  const dueInputs = new Map();
  const reminderFor = (name) => (reminders || []).find((m) => m.name.toLowerCase() === name.toLowerCase());
  const unit = isMetric() ? 'km' : 'mi';
  function drawDue() {
    dueBox.replaceChildren(...[...chosen].map((name) => {
      const job = JOBS.find((j) => j[0] === name), rem = reminderFor(name);
      const miles = rem && rem.everyKm ? Math.round(isMetric() ? rem.everyKm : rem.everyKm / KM) : job && job[1] ? Math.round(isMetric() ? job[1] * KM : job[1]) : '';
      const months = rem && rem.everyDays ? Math.round(rem.everyDays / 30.4) : job && job[2] ? job[2] : '';
      const prev = dueInputs.get(name);
      const d = prev ? prev.d : h('input', { type: 'number', min: 1, inputmode: 'numeric', value: miles, placeholder: unit, style: 'width:6.5em', 'aria-label': `${name} due again in ${unit}` });
      const m = prev ? prev.m : h('input', { type: 'number', min: 1, inputmode: 'numeric', value: months, placeholder: 'months', style: 'width:5.5em', 'aria-label': `${name} due again in months` });
      dueInputs.set(name, { d, m });
      return h('div', { class: 'row', style: 'gap:6px;flex-wrap:wrap' }, h('span', { class: 'grow', style: 'flex-basis:9em' }, name), h('span', { class: 'muted' }, 'again in'), d, h('span', { class: 'muted' }, `${unit} or`), m, h('span', { class: 'muted' }, 'months'));
    }));
  }
  const chips = h('div', { class: 'tags' });
  function drawChips() {
    const names = [...new Set([...JOBS.map((j) => j[0]), ...chosen])];
    chips.replaceChildren(...names.map((n) => h('button', { type: 'button', class: `chipbtn${chosen.has(n) ? ' on' : ''}`, 'aria-pressed': String(chosen.has(n)), onclick: () => { if (chosen.has(n)) chosen.delete(n); else chosen.add(n); drawChips(); drawDue(); } }, n)));
  }
  const addCustom = () => { const v = custom.value.trim(); if (v) { chosen.add(v); custom.value = ''; drawChips(); drawDue(); } };
  custom.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); addCustom(); } });
  const save = async (status) => {
    const due = [...chosen].map((item) => {
      const x = dueInputs.get(item) || {};
      const d = Number(x.d && x.d.value), m = Number(x.m && x.m.value);
      return { item, everyKm: d > 0 ? (isMetric() ? d : d * KM) : null, everyDays: m > 0 ? Math.round(m * 30.4) : null };
    });
    const body = { status, items: [...chosen], notes: notes.value, cost: cost.value === '' ? null : Number(cost.value), due };
    if (place) body.placeName = place.value;
    try {
      if (status !== 'dismissed' && !chosen.size) { msg.textContent = 'Tap what was done, or add your own.'; return; }
      await api(visit ? `/api/service/${visit.id}` : '/api/service', { method: visit ? 'PATCH' : 'POST', body });
      onSaved(status);
    } catch (e) { msg.textContent = e.message; }
  };
  drawChips();
  drawDue();
  return h('div', {}, place,
    h('h3', { style: 'margin:12px 0 6px' }, 'What was done'), chips,
    h('div', { class: 'row', style: 'border:0;padding:0;gap:8px' }, custom, h('button', { type: 'button', class: 'sec sm', onclick: addCustom }, 'Add')),
    h('h3', { style: 'margin:12px 0 6px' }, 'What it cost'), cost,
    h('h3', { style: 'margin:12px 0 6px' }, 'When each is due again'), h('p', { class: 'muted', style: 'margin:0 0 6px' }, 'Sets your service reminders. Leave empty for no reminder.'), dueBox,
    h('div', { style: 'margin-top:10px' }, notes),
    h('p', { class: 'seg', style: 'margin-top:12px' }, h('button', { type: 'button', class: 'on', onclick: () => save('logged') }, 'Save'),
      visit && visit.status === 'pending' ? h('button', { type: 'button', onclick: () => save('dismissed') }, 'It was not a service visit') : null), msg);
}
