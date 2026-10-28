import { h, chip, when, duration, pct, num } from './ui.js';
import { api } from './api.js';
import { curveBox } from './curve.js';

// One charging session in the Charging list, laid out like a trip card: a picture on the left (the battery, filled from where
// it started to where it ended), what it added and how long, where, the battery bar, and the cost with where it came from.
// What an admin can change (where it was, the charger's kWh, what was paid) folds away behind Edit.

const money = (cur, v) => `${cur}${v.toFixed(2)}`;
const WHERE = { home: 'Home', public: 'Public' };
const FROM = { receipt: 'receipt', place: 'place price', home: 'home price', 'home+night': 'home and night price', public: 'public price' };

const NS = 'http://www.w3.org/2000/svg';
const svgEl = (tag, attrs) => { const e = document.createElementNS(NS, tag); for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v); return e; };

// A small upright battery: the part that was there before in a quiet color, the part this charge added in green.
function batteryThumb(c) {
  const box = h('div', { class: 'tthumb', 'aria-hidden': 'true' });
  const a = Math.max(0, Math.min(100, c.startSoc ?? 0)), b = Math.max(a, Math.min(100, c.endSoc ?? a));
  const top = 10, H = 46, y = (p) => top + H - (H * p) / 100;
  const svg = svgEl('svg', { viewBox: '0 0 76 64' });
  svg.append(svgEl('rect', { x: 32, y: 5, width: 12, height: 5, rx: 1.5, class: 'cbat-cap' }),
    svgEl('rect', { x: 24, y: top, width: 28, height: H, rx: 5, class: 'cbat-shell' }),
    svgEl('rect', { x: 27, y: y(a) - 0.01, width: 22, height: Math.max(0, (H * a) / 100 - 3), class: 'cbat-before' }),
    svgEl('rect', { x: 27, y: y(b) + 3, width: 22, height: Math.max(0, (H * (b - a)) / 100), class: 'cbat-added' }));
  box.append(svg);
  return box;
}

// The battery bar: what was there before, and what the charge added.
function chargeBar(c) {
  if (c.startSoc == null || c.endSoc == null) return null;
  const a = Math.max(0, Math.min(100, c.startSoc)), b = Math.max(a, Math.min(100, c.endSoc));
  return h('div', { class: 'sbar', title: `${pct(c.startSoc)} to ${pct(c.endSoc)}` }, h('i', { class: 'before', style: `width:${a}%` }), h('i', { class: 'added', style: `left:${a}%;width:${b - a}%` }));
}

// What the charger reported and what you paid: from the receipt, to see this session's real efficiency and cost.
function editBox(c, cur, reload) {
  const msg = h('span', { class: 'muted', role: 'status' });
  const where = h('select', { 'aria-label': 'Where it was charged' },
    ...[['', c.where && !c.kind ? `Automatic (${c.where === 'home' ? 'home' : 'away from home'})` : 'Automatic'], ['home', 'Home'], ['public', 'Away from home']].map(([v, t]) => h('option', { value: v, selected: v === (c.kind || '') }, t)));
  const kwh = h('input', { type: 'number', step: '0.1', min: 0, inputmode: 'decimal', placeholder: 'kWh', 'aria-label': 'kWh the charger reported', value: c.chargerKwh != null ? c.chargerKwh : '', style: 'width:7em' });
  const paid = h('input', { type: 'number', step: '0.01', min: 0, inputmode: 'decimal', placeholder: cur || '$', 'aria-label': 'What you paid', value: c.costActual && c.cost != null ? c.cost.toFixed(2) : '', style: 'width:7em' });
  const save = async () => {
    const body = { kind: where.value };
    if (kwh.value) body.chargerKwh = Number(kwh.value); else body.clearCharger = true;
    if (paid.value) body.cost = Number(paid.value); else if (c.costActual) body.clearCost = true;
    try { await api(`/api/charges/${c.id}`, { method: 'PATCH', body }); reload(); } catch (e) { msg.textContent = e.message; }
  };
  const field = (label, el) => h('label', { class: 'cfield' }, h('span', { class: 'muted' }, label), el);
  return h('div', { class: 'cedit' }, field('Where', where), field('Charger said (kWh)', kwh), field('You paid', paid),
    h('div', { class: 'seg' }, h('button', { type: 'button', onclick: save }, 'Save')), msg);
}

export function sessionCard(c, cur, admin, reload) {
  const secs = (c.endTs || Date.now() / 1000) - c.startTs;
  const kw = c.addedKwh > 0 && secs > 300 ? (c.addedKwh / secs) * 3600 : null;
  const perKwh = c.cost != null && (c.wallKwh || c.addedKwh) > 0.5 ? c.cost / (c.wallKwh || c.addedKwh) : null;
  const place = [WHERE[c.where] || 'Charging', c.place].filter(Boolean).join(' · ');
  const edit = admin && !c.open && c.addedKwh != null ? h('div', { hidden: true }, editBox(c, cur, reload)) : null;
  const editBtn = edit ? h('button', { type: 'button', class: 'sec sm', 'aria-expanded': 'false', onclick: (e) => { edit.hidden = !edit.hidden; e.currentTarget.setAttribute('aria-expanded', String(!edit.hidden)); } }, 'Edit') : null;
  return h('section', { class: 'tcard ccard', style: `--cc:var(--ok)` },
    batteryThumb(c),
    h('div', { class: 'tmain' },
      h('div', { class: 'tline' }, h('span', { class: 'tbig' }, c.addedKwh != null ? `+${num(c.addedKwh, 1, ' kWh')}` : 'Charging'), h('span', { class: 'muted' }, duration(c.startTs, c.endTs || Math.floor(Date.now() / 1000))),
        c.open ? chip('ok', 'charging now') : null, c.estimated ? chip('work', 'estimate') : null),
      h('div', { class: 'muted' }, when(c.startTs)),
      h('div', { class: 'muted' }, place),
      h('div', { class: 'tline' }, chargeBar(c), h('span', { class: 'muted' }, `${pct(c.startSoc)} → ${pct(c.endSoc)}`)),
      h('div', { class: 'tline' }, c.peakKw > 0 ? h('span', { class: 'pill' }, `peak ${num(c.peakKw, c.peakKw < 20 ? 1 : 0, ' kW')}`) : null, kw ? h('span', { class: 'pill' }, `avg ${num(kw, kw < 10 ? 1 : 0, ' kW')}`) : null,
        c.measured && c.efficiency != null ? h('span', { class: 'pill good' }, `${Math.round(c.efficiency * 100)}% reached the battery`) : null)),
    h('div', { class: 'tside' },
      h('div', { class: 'ccost' }, c.cost != null ? h('b', {}, money(cur, c.cost)) : h('span', { class: 'muted' }, 'no price'),
        perKwh != null ? h('div', { class: 'muted' }, `${money(cur, perKwh)}/kWh`) : null,
        c.gas && c.cost != null ? h('div', { class: 'muted', title: `The same range in the gas car${c.gas.source ? ` (${c.gas.source})` : ''}` }, `gas ${money(cur, c.gas.cost)}`) : null,
        c.cost != null ? chip(c.costActual ? 'ok' : 'work', c.costActual ? 'receipt' : FROM[c.priceFrom] || 'estimate') : null),
      editBtn),
    h('div', { class: 'cfoot' }, c.open ? null : curveBox(c.id), edit));
}
