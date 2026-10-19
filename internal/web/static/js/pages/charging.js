import { h, emptyState, chip, when, duration, pct, num, dist, isMetric } from '../ui.js';
import { api } from '../api.js';
import { forecastCard } from '../chargeforecast.js';
import { chargeNow } from '../chargenow.js';
import { curveBox, chargersCompared, recordsCard } from '../curve.js';

const money = (cur, v) => `${cur}${v.toFixed(2)}`;
const monthName = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short', year: 'numeric' }); };
const WHERE = { home: 'Home', public: 'Away from home' };

function perDistance(cur, costPerKm) {
  return `${money(cur, isMetric() ? costPerKm : costPerKm * 1.609344)} per ${isMetric() ? 'km' : 'mi'}`;
}

// What the charger reported: type it in to see this session's real efficiency, and to teach CarTrak yours.
function chargerBox(c, reload) {
  if (c.open || c.addedKwh == null) return null;
  const input = h('input', { type: 'number', step: '0.1', min: 0, inputmode: 'decimal', placeholder: 'kWh from the charger', 'aria-label': 'kWh the charger reported', value: c.chargerKwh != null ? c.chargerKwh : '', style: 'max-width:11em' });
  const msg = h('span', { class: 'muted', role: 'status' });
  const save = async () => {
    try { await api(`/api/charges/${c.id}`, { method: 'PATCH', body: input.value ? { chargerKwh: Number(input.value) } : { clearCharger: true } }); reload(); } catch (e) { msg.textContent = e.message; }
  };
  return h('div', { style: 'margin-top:8px;display:flex;gap:8px;align-items:center;flex-wrap:wrap' }, input, h('button', { class: 'sec', onclick: save }, 'Save'),
    c.measured && c.efficiency != null ? chip('ok', `${Math.round(c.efficiency * 100)}% reached the battery`) : null, msg);
}

function summaryCard(sum) {
  const cur = sum.currency;
  const first = sum.months[0];
  const rows = sum.months.map((m) => h('div', { class: 'row' },
    h('span', { class: 'grow' }, monthName(m.month), ' ', h('span', { class: 'muted' }, `${m.sessions} session${m.sessions === 1 ? '' : 's'} · ${num(m.kwh, 0, ' kWh')}${m.km > 0 ? ` · ${dist(m.km, 0)} driven` : ''}`)),
    h('span', {}, m.priced ? h('b', {}, money(cur, m.cost)) : '—', m.costPerKm != null ? h('div', { class: 'muted' }, perDistance(cur, m.costPerKm)) : null)));
  return h('section', { class: 'card' }, h('h2', {}, 'Charging cost'),
    sum.priceSet ? null : h('p', { class: 'banner notice' }, 'Add your price per kWh in More, Settings (Price per kWh at home) to see what charging costs.'),
    sum.thisMonth ? h('p', {}, 'This month so far ', h('b', {}, money(cur, sum.thisMonth.cost)), sum.thisMonth.projected != null ? h('span', { class: 'muted' }, ` · on pace for about ${money(cur, sum.thisMonth.projected)}`) : null) : null,
    first ? h('p', { class: 'muted' }, `By month, from the energy that went into the battery. On a slow charger the cost includes what is lost as heat: ${Math.round((1 - sum.efficiency.value) * 100)}% ${sum.efficiency.source === 'measured' ? `(measured from ${sum.efficiency.samples} sessions where you entered the charger's number)` : sum.efficiency.samples ? `(your setting; ${sum.efficiency.samples} of the 3 sessions needed to measure it have the charger's number)` : "(your setting; enter the charger's kWh on a few sessions below to measure it)"}.`) : null,
    ...rows);
}

function sessionCard(c, cur, admin, reload) {
  const added = c.addedKwh != null ? `${num(c.addedKwh, 1, ' kWh')}${c.estimated ? ' (estimate)' : ''} added` : 'energy not known';
  const where = admin
    ? h('select', { 'aria-label': 'Where it was charged', onchange: async (e) => { await api(`/api/charges/${c.id}`, { method: 'PATCH', body: { kind: e.target.value } }); reload(); } },
      ...[['', c.where && !c.kind ? `Automatic (${(WHERE[c.where] || '').toLowerCase()})` : 'Automatic'], ['home', 'Home'], ['public', 'Away from home']].map(([v, t]) => h('option', { value: v, selected: v === (c.kind || '') }, t)))
    : (c.where ? chip('work', WHERE[c.where]) : null);
  return h('section', { class: 'card' },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, when(c.startTs)), c.open ? chip('ok', 'charging') : null, c.cost != null ? h('b', {}, money(cur, c.cost)) : null),
    h('div', { class: 'muted' }, `${pct(c.startSoc)} → ${pct(c.endSoc)} · ${added} · ${duration(c.startTs, c.endTs)} · peak ${num(c.peakKw, 1, ' kW')}`),
    where ? h('div', { style: 'margin-top:8px' }, where) : null,
    c.open ? null : curveBox(c.id),
    admin ? chargerBox(c, reload) : null);
}

export default async function charging(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  const now = chargeNow();
  root.append(h('h2', {}, 'Charging'), now.el, forecastCard(), box);
  async function load() {
    const [cs, sum] = await Promise.all([api('/api/charges'), api('/api/charges/summary')]);
    if (!cs.length) return box.replaceChildren(emptyState('No charging sessions yet', 'Sessions appear on their own after the car charges. Set your electricity price to see what each one costs.', admin ? '#/admin' : null, 'Open Settings'));
    box.replaceChildren(...[summaryCard(sum), recordsCard(cs), chargersCompared(cs), h('p', {}, h('a', { class: 'btn sec', href: '/api/export/charging.csv' }, 'Export CSV')), h('h2', {}, 'Sessions'), h('div', { class: 'cols' }, ...cs.map((c) => sessionCard(c, sum.currency, admin, load)))].filter(Boolean));
  }
  await load();
  return { cleanup: now.stop };
}
