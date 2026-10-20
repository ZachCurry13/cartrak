import { h, stat, dist, num, when } from '../ui.js';
import { api } from '../api.js';
import { catLabel } from '../categories.js';
import { effSpan } from '../energy.js';
import { tapBars } from '../chart.js';

const monthName = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short' }); };

// Bars for kilometres per month, scaled to the busiest one.
function bars(months) {
  return tapBars(months.map((m) => ({ label: monthName(m.month), value: m.km, text: `${dist(m.km, 0)} in ${m.trips} trip${m.trips === 1 ? '' : 's'}` })), { summary: 'Distance driven each month' });
}

// A plain-text version to paste into a message or a note.
function summaryText(d) {
  const v = d.driving, c = d.charging, cur = d.currency;
  return [`CarTrak ${d.year}${d.partial ? ' so far' : ''}`,
    `Driven: ${dist(d.odometerKm || v.km, 0)} in ${v.trips} trips (work ${dist(v.workKm, 0)}, personal ${dist(v.personalKm, 0)}${Object.entries(v.otherKm || {}).map(([k, km]) => `, ${catLabel(k).toLowerCase()} ${dist(km, 0)}`).join('')})`,
    v.energyTrips ? `Energy used: ${num(v.energyKwh, 0, ' kWh')}` : null,
    c.sessions ? `Charging: ${c.sessions} sessions, ${num(c.addedKwh, 0, ' kWh')} added${c.priced ? `, ${cur}${c.cost.toFixed(2)}` : ''}` : null].filter(Boolean).join('\n');
}

export default async function year(root, [y]) {
  const d = await api(`/api/report/year${y ? `?year=${encodeURIComponent(y)}` : ''}`);
  const v = d.driving, c = d.charging, cur = d.currency;
  const pick = h('select', { 'aria-label': 'Year', onchange: (e) => { location.hash = `#/year/${e.target.value}`; } },
    ...d.years.slice().reverse().map((x) => h('option', { value: x, selected: x === d.year }, String(x))));
  const msg = h('span', { class: 'muted', role: 'status' });
  const copy = async () => { try { await navigator.clipboard.writeText(summaryText(d)); msg.textContent = 'Copied'; } catch (e) { msg.textContent = 'Could not copy'; } };
  root.append(...[h('h2', {}, `${d.year} in review`), h('p', {}, pick),
    d.partial ? h('p', { class: 'muted' }, 'This year is not over yet, so these are the totals so far.') : null,
    !v.trips && !c.sessions ? h('p', { class: 'muted' }, 'Nothing was recorded this year.')
      : h('div', {},
        h('section', { class: 'card' }, h('h2', {}, 'Driving'),
          h('div', { class: 'grid' }, stat('Distance', dist(d.odometerKm || v.km, 0)), stat('Trips', String(v.trips)), stat('Work', dist(v.workKm, 0)), stat('Personal', dist(v.personalKm, 0)), ...Object.entries(v.otherKm || {}).map(([k, km]) => stat(catLabel(k), dist(km, 0))), d.mileageRate && v.workKm > 0 ? stat('Work deduction (est.)', `${cur}${(v.workKm * 0.621371 * d.mileageRate).toFixed(0)}`) : null,
            v.energyTrips ? stat('Energy used', num(v.energyKwh, 0, ' kWh')) : null, v.energyTrips ? stat(v.estimated ? 'Efficiency (estimate)' : 'Efficiency', effSpan(v.energyKm, v.energyKwh, v.estimated ? '≈ ' : '')) : null),
          v.months.length ? bars(v.months) : null,
          v.busiest ? h('p', { class: 'muted' }, `Busiest month: ${new Date(+v.busiest.slice(0, 4), +v.busiest.slice(5) - 1, 1).toLocaleDateString([], { month: 'long' })}.`) : null,
          v.longest ? h('p', {}, 'Longest trip: ', h('a', { href: `#/trip/${v.longest.id}` }, `${dist(v.longest.km, 0)}, ${when(v.longest.ts)}`)) : null),
        c.sessions ? h('section', { class: 'card' }, h('h2', {}, 'Charging'),
          h('div', { class: 'grid' }, stat('Sessions', String(c.sessions)), stat('Energy added', num(c.addedKwh, 0, ' kWh')),
            c.priced ? stat('Cost', `${cur}${c.cost.toFixed(2)}`) : null,
            c.homeKwh + c.publicKwh > 0 ? stat('At home', `${Math.round((c.homeKwh / (c.homeKwh + c.publicKwh)) * 100)}%`) : null,
            c.lossPercent != null ? stat('Lost as heat', `${c.lossPercent}%`) : null)) : null,
        h('p', { class: 'seg' }, h('button', { class: 'sec', onclick: copy }, 'Copy summary'), ' ', msg))].filter(Boolean));
}
