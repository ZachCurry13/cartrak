import { h, when, pct } from '../ui.js';
import { api } from '../api.js';
import { volText, priceText, economy, volUnit, toDisplayVol, fromDisplayVol, toDisplayPrice, fromDisplayPrice } from '../fuelunits.js';

// Fuel, for a gas or hybrid car: the fill-ups, what each cost, and the tank size CarTrak works out from them. A jump in the fuel
// gauge is noticed as a fill-up; you enter the gallons and the price per gallon each time.

const card = (title, ...kids) => h('section', { class: 'card' }, h('h2', {}, title), ...kids);
const numberBox = (id, placeholder, step = '0.01') => h('input', { id, type: 'number', inputmode: 'decimal', min: '0', step, placeholder });
const field = (id, label, input) => h('div', {}, h('label', { for: id }, label), input);

// The two boxes that complete a fill-up: what went in and what a gallon (or liter) cost.
function amountBoxes(prefix, fill) {
  const vol = numberBox(`${prefix}v`, volUnit()), price = numberBox(`${prefix}p`, `per ${volUnit()}`, '0.001');
  if (fill && fill.gallons != null) vol.value = toDisplayVol(fill.gallons).toFixed(2);
  if (fill && fill.pricePerGal != null) price.value = toDisplayPrice(fill.pricePerGal).toFixed(3);
  const read = () => ({ gallons: fromDisplayVol(Number(vol.value)), pricePerGal: fromDisplayPrice(Number(price.value)), ok: vol.value > 0 && price.value > 0 });
  return { row: h('div', { class: 'cols2' }, field(`${prefix}v`, `${volUnit() === 'L' ? 'Liters' : 'Gallons'} put in`, vol), field(`${prefix}p`, `Price per ${volUnit()}`, price)), read };
}

function pendingCard(f, admin, reload) {
  const { row, read } = amountBoxes(`pf${f.id}`);
  const msg = h('p', { class: 'muted', role: 'status' });
  const send = (body) => api(`/api/fuel/fills/${f.id}`, { method: 'PATCH', body }).then(reload).catch((e) => { msg.textContent = e.message; });
  return card('Filled up?', h('p', {}, `${when(f.ts)}: the tank went from ${pct(f.startPct)} to ${pct(f.endPct)}.`),
    admin ? [row, h('p', { class: 'seg' },
      h('button', { type: 'button', class: 'on', onclick: () => { const v = read(); if (!v.ok) { msg.textContent = 'Enter both the amount and the price.'; return; } send({ gallons: v.gallons, pricePerGal: v.pricePerGal }); } }, 'Save fill-up'),
      h('button', { type: 'button', class: 'sec', onclick: () => send({ status: 'dismissed' }) }, 'Not a fill-up')), msg]
      : h('p', { class: 'muted' }, 'An admin can enter the gallons and the price.'));
}

function addForm(reload) {
  const { row, read } = amountBoxes('af');
  const when_ = h('input', { id: 'afd', type: 'datetime-local' });
  const from = numberBox('afs', '%', '1'), to = numberBox('afe', '%', '1');
  const now = new Date(); now.setMinutes(now.getMinutes() - now.getTimezoneOffset());
  when_.value = now.toISOString().slice(0, 16);
  const msg = h('p', { class: 'muted', role: 'status' });
  return h('details', {}, h('summary', {}, 'Add a fill-up by hand'),
    field('afd', 'When', when_), row,
    h('div', { class: 'cols2' }, field('afs', 'Tank before (%), optional', from), field('afe', 'Tank after (%), optional', to)),
    h('p', { class: 'muted' }, 'The tank levels let CarTrak learn the tank size from this fill-up. Read them off the gauge before and after.'),
    h('p', {}, h('button', { type: 'button', onclick: async () => {
      const v = read();
      if (!v.ok) { msg.textContent = 'Enter both the amount and the price.'; return; }
      try {
        await api('/api/fuel/fills', { method: 'POST', body: { ts: Math.floor(new Date(when_.value).getTime() / 1000), gallons: v.gallons, pricePerGal: v.pricePerGal, startPct: from.value ? Number(from.value) : null, endPct: to.value ? Number(to.value) : null } });
        reload();
      } catch (e) { msg.textContent = e.message; }
    } }, 'Save fill-up')), msg);
}

// How the price paid compares with the average that week where the car was, when live gas prices are on.
function vsArea(f) {
  if (f.areaPerGal == null || f.pricePerGal == null) return null;
  const c = Math.round((f.pricePerGal - f.areaPerGal) * 100);
  if (c === 0) return `the same as the ${f.areaSource}`;
  return `${Math.abs(c)}¢ a gallon ${c < 0 ? 'under' : 'over'} the ${f.areaSource}`;
}

function fillRow(f, cur, admin, reload) {
  const eco = f.kmPerGal > 0 ? `${economy(f.kmPerGal, 1)} since the last fill-up` : '';
  const del = admin ? h('button', { type: 'button', class: 'danger sm', 'aria-label': 'Delete this fill-up', onclick: async () => { if (confirm('Delete this fill-up?')) { await api(`/api/fuel/fills/${f.id}`, { method: 'DELETE' }); reload(); } } }, 'Delete') : null;
  return h('div', { class: 'row', style: 'align-items:flex-start;flex-wrap:wrap' },
    h('span', { class: 'grow', style: 'flex-basis:12em;min-width:0' }, h('b', {}, volText(f.gallons)), ' ', h('span', { class: 'muted' }, f.pricePerGal != null ? priceText(cur, f.pricePerGal) : 'no price'),
      h('div', { class: 'muted' }, `${when(f.ts)}${f.startPct || f.endPct ? ` · ${pct(f.startPct)} → ${pct(f.endPct)}` : ''}`), eco ? h('div', { class: 'muted' }, eco) : null, vsArea(f) ? h('div', { class: 'muted' }, vsArea(f)) : null),
    h('span', {}, f.cost != null ? h('b', {}, `${cur}${f.cost.toFixed(2)}`) : null), del);
}

export default async function fuel(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Fuel')), box);
  const draw = async () => {
    const d = await api('/api/fuel');
    const cur = d.currency || '$', fills = d.fills.filter((f) => f.status !== 'pending'), pending = d.fills.filter((f) => f.status === 'pending');
    const m = d.months[0];
    box.replaceChildren(...[
      ...pending.map((f) => pendingCard(f, admin, draw)),
      card('Tank size', d.tankGal != null
        ? [h('p', { class: 'bigfig' }, h('b', {}, volText(d.tankGal)), ' ', h('span', { class: 'muted' }, `worked out from ${d.tankFills} fill-up${d.tankFills === 1 ? '' : 's'}`)),
          h('p', { class: 'muted' }, 'It turns the fuel gauge into gallons: how much each drive burned and what it cost. It gets better with every fill-up that starts well below full.')]
        : h('p', { class: 'muted' }, 'Not known yet. When the fuel gauge jumps up, CarTrak keeps a fill-up and asks how many gallons you put in and what a gallon cost. From the gallons and how far the gauge climbed it works out the size of the tank. Until then trips show how far a full tank would go.')),
      m ? card('This month', h('div', { class: 'bigfig' }, h('div', {}, h('b', {}, `${cur}${m.cost.toFixed(2)}`), h('span', {}, `${m.fills} fill-up${m.fills === 1 ? '' : 's'}`)),
        h('div', {}, h('b', {}, volText(m.gallons)), h('span', {}, 'bought')), h('div', {}, h('b', {}, m.gallons > 0 ? priceText(cur, m.cost / m.gallons) : '—'), h('span', {}, 'average price'))),
        h('p', { class: 'muted', style: 'margin-top:8px' }, d.months.length > 1 ? `Last month: ${cur}${d.months[1].cost.toFixed(2)} for ${volText(d.months[1].gallons)}.` : ''),
        h('p', {}, h('a', { href: '#/analytics/fuel' }, 'Fuel in Insights'))) : null,
      admin ? card('Add one', addForm(draw)) : null,
      fills.length ? card('Fill-ups', ...fills.slice().map((f) => fillRow(f, cur, admin, draw))) : h('p', { class: 'muted' }, 'No fill-ups yet.'),
    ].flat().filter(Boolean));
  };
  await draw();
}
