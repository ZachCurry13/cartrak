import { h, isMetric } from './ui.js';
import { api } from './api.js';
import { lineChart } from './chart.js';
import { stackBars } from './charts2.js';
import { windowPicker } from './insightsmore.js';
import { volText, priceText, economy, economyFromKmPerGal } from './fuelunits.js';

// Fuel in Insights (#/analytics/fuel), for a gas or hybrid car: what it spent, how far a gallon went, by month.

const card = (title, ...kids) => h('section', { class: 'card' }, h('h2', {}, title), ...kids);
const monthShort = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short' }); };

export function fuel(body) {
  const out = h('div');
  windowPicker(body, async (days) => {
    const d = await api(`/api/analytics/fuel?days=${days}`);
    const cur = d.currency || '$', t = d.total;
    if (!t.fills && !t.gal) {
      out.replaceChildren(h('p', { class: 'muted' }, 'No fuel in this stretch yet. Log your fill-ups under Fuel (More) and this fills in.'), h('p', {}, h('a', { href: '#/fuel' }, 'Open Fuel')));
      return;
    }
    const months = d.months.slice().reverse(); // oldest first for the charts
    const econ = months.filter((m) => m.kmPerGal > 0);
    out.replaceChildren(
      card('In this stretch', h('div', { class: 'bigfig' },
        h('div', {}, h('b', {}, `${cur}${t.spent.toFixed(0)}`), h('span', {}, `${t.fills} fill-up${t.fills === 1 ? '' : 's'}`)),
        h('div', {}, h('b', {}, volText(t.galBought)), h('span', {}, 'bought')),
        h('div', {}, h('b', {}, t.avgPrice > 0 ? priceText(cur, t.avgPrice) : '—'), h('span', {}, 'average price')),
        h('div', {}, h('b', {}, t.kmPerGal > 0 ? economy(t.kmPerGal, 1) : '—'), h('span', {}, 'fuel economy'))),
        t.costPerKm > 0 ? h('p', { class: 'muted', style: 'margin-top:8px' }, `About ${cur}${(t.costPerKm * (isMetric() ? 1 : 1.609344)).toFixed(2)} a ${isMetric() ? 'km' : 'mile'} in fuel.`) : null,
        d.tankGal ? h('p', { class: 'muted' }, `Tank size: ${volText(d.tankGal)}, worked out from ${d.tankFills} fill-up${d.tankFills === 1 ? '' : 's'}.`) : null,
        h('p', {}, h('a', { href: '#/fuel' }, 'Fuel and fill-ups'))),
      months.some((m) => m.spent > 0) ? card('Spent by month', stackBars(months.map((m) => ({ label: monthShort(m.period), parts: [m.spent],
        text: `${cur}${m.spent.toFixed(2)} for ${volText(m.galBought)} in ${m.fills} fill-up${m.fills === 1 ? '' : 's'}` })), { classes: ['cost'], legend: [`Spent (${cur})`], summary: 'Tap a bar' })) : null,
      econ.length >= 2 ? card('Fuel economy by month', lineChart({ points: econ.map((m, i) => ({ x: i, y: m.kmPerGal })), xFmt: (x) => monthShort(econ[Math.round(x)].period),
        yFmt: (y) => (economyFromKmPerGal(y) || '').split(' ')[0], summary: 'Average over the trips of each month, from the tank gauge', yTitle: (economyFromKmPerGal(1) || '').split(' ').slice(1).join(' ') }),
        h('p', { class: 'muted' }, 'The fuel gauge moves in coarse steps, so single drives are estimates; a month averages them out.')) : null);
  });
  body.append(out);
}
