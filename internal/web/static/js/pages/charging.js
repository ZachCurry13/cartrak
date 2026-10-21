import { savingsCard, perDistance } from '../savingscard.js';
import { h, emptyState, num, dist } from '../ui.js';
import { sessionCard } from '../chargesession.js';
import { coachCard } from '../coachcard.js';
import { api } from '../api.js';
import { forecastCard } from '../chargeforecast.js';
import { chargeNow } from '../chargenow.js';
import { chargersCompared, recordsCard } from '../curve.js';

const money = (cur, v) => `${cur}${v.toFixed(2)}`;
const monthName = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short', year: 'numeric' }); };

function summaryCard(sum) {
  const cur = sum.currency;
  const first = sum.months[0];
  const rows = sum.months.map((m) => h('div', { class: 'row' },
    h('span', { class: 'grow' }, monthName(m.month), ' ', h('span', { class: 'muted' }, `${m.sessions} session${m.sessions === 1 ? '' : 's'} · ${num(m.kwh, 0, ' kWh')}${m.km > 0 ? ` · ${dist(m.km, 0)} driven` : ''}`)),
    h('span', {}, m.priced ? h('b', {}, money(cur, m.cost)) : '—', m.costPerKm != null ? h('div', { class: 'muted' }, perDistance(cur, m.costPerKm)) : null)));
  return h('section', { class: 'card' }, h('h2', {}, 'Charging cost'),
    sum.priceSet ? null : h('p', { class: 'banner notice' }, 'Add your price per kWh under ', h('a', { href: '#/charging-settings' }, 'Charging prices'), ' to see what charging costs.'),
    sum.thisMonth ? h('p', {}, 'This month so far ', h('b', {}, money(cur, sum.thisMonth.cost)), sum.thisMonth.projected != null ? h('span', { class: 'muted' }, ` · on pace for about ${money(cur, sum.thisMonth.projected)}`) : null) : null,
    first ? h('p', { class: 'muted' }, `By month, from the energy that went into the battery. On a slow charger the cost includes what is lost as heat: ${Math.round((1 - sum.efficiency.value) * 100)}% ${sum.efficiency.source === 'measured' ? `(measured from ${sum.efficiency.samples} sessions where you entered the charger's number)` : sum.efficiency.samples ? `(your setting; ${sum.efficiency.samples} of the 3 sessions needed to measure it have the charger's number)` : "(your setting; enter the charger's kWh on a few sessions below to measure it)"}.`) : null,
    ...rows);
}

export default async function charging(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  const now = chargeNow();
  root.append(h('div', { class: 'page-title' }, h('h2', {}, 'Charging')), h('nav', { class: 'linkrow', 'aria-label': 'More about charging' }, h('a', { href: '#/charging-settings' }, 'Prices and settings'), h('a', { href: '#/stats' }, 'Statistics'), h('a', { href: '#/history' }, 'History'), h('a', { href: '#/plan' }, 'Plan a trip')), now.el, forecastCard(), box);
  async function load() {
    const [cs, sum, savings, tips] = await Promise.all([api('/api/charges'), api('/api/charges/summary'), savingsCard(admin), coachCard()]);
    if (!cs.length) return box.replaceChildren(emptyState('No charging sessions yet', 'Sessions appear on their own after the car charges. Set your electricity price to see what each one costs.', admin ? '#/admin' : null, 'Open Settings'));
    box.replaceChildren(...[summaryCard(sum), tips, savings, recordsCard(cs), chargersCompared(cs), h('p', {}, h('a', { class: 'btn sec', href: '/api/export/charging.csv' }, 'Export CSV')), h('h2', {}, 'Sessions'), h('div', { class: 'cols' }, ...cs.map((c) => sessionCard(c, sum.currency, admin, load)))].filter(Boolean));
  }
  await load();
  return { cleanup: now.stop };
}
