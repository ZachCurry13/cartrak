import { h, dist, temp, num, isMetric } from './ui.js';
import { api } from './api.js';
import { lineChart } from './chart.js';
import { stackBars, heatGrid, calendarHeat, scatter } from './charts2.js';
import { efficiency } from './energy.js';

// The detailed Insights charts for charging and for driving (#/analytics/charging, #/analytics/driving).

export const WEEK = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
export const HOURS = Array.from({ length: 24 }, (_, i) => (i % 6 === 0 ? `${(i % 12) || 12}${i < 12 ? 'a' : 'p'}` : ''));
const card = (title, ...kids) => h('section', { class: 'card' }, h('h2', {}, title), ...kids);
const monthShort = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short' }); };
const eff = (k) => efficiency(k * 100, 100) || '—';
const hourText = (c) => new Date(2000, 0, 1, c).toLocaleTimeString([], { hour: 'numeric' });

// The window the page looks over: 90 days, a year or everything. Changing it draws the page again.
export function windowPicker(body, draw, value = '365') {
  const seg = h('div', { class: 'seg', style: 'margin-bottom:10px' });
  const set = (v) => { seg.replaceChildren(...[['90', '90 days'], ['365', '12 months'], ['0', 'All']].map(([k, t]) => h('button', { type: 'button', class: k === v ? 'on' : '', 'aria-pressed': String(k === v), onclick: () => set(k) }, t))); draw(v); };
  body.append(seg);
  set(value);
}

export function charging(body) {
  const out = h('div');
  windowPicker(body, async (days) => {
    const d = await api(`/api/analytics/charging?days=${days}`);
    const cur = d.currency || '$';
    if (!d.sessions) { out.replaceChildren(h('p', { class: 'muted' }, 'No charging sessions in this stretch.')); return; }
    const money = (v) => `${cur}${v.toFixed(2)}`;
    const avg = d.months.reduce((a, m) => a + m.homeCost + m.publicCost, 0) / Math.max(1, d.kwh);
    const plugTop = WEEK.map((w, r) => d.plugIn[r].map((v, c) => [v, r, c])).flat().sort((a, b) => b[0] - a[0])[0];
    out.replaceChildren(
      card('In this stretch', h('div', { class: 'bigfig' }, h('div', {}, h('b', {}, num(d.kwh, 0, ' kWh')), h('span', {}, `${d.sessions} sessions`)), h('div', {}, h('b', {}, money(d.cost)), h('span', {}, 'charging cost')),
        h('div', {}, h('b', {}, `${Math.round(d.homeShare * 100)}%`), h('span', {}, 'of the energy at home')), h('div', {}, h('b', {}, money(avg)), h('span', {}, 'per kWh in the battery')))),
      d.months.length ? card('Energy by month', stackBars(d.months.map((m) => ({ label: monthShort(m.month), parts: [m.homeKwh, m.publicKwh],
        text: `${num(m.homeKwh, 0, ' kWh')} at home, ${num(m.publicKwh, 0, ' kWh')} public · ${money(m.homeCost + m.publicCost)}` })), { classes: ['home', 'public'], legend: ['Home', 'Public'], summary: 'Tap a month' })) : null,
      d.months.filter((m) => m.pricePerKwh > 0).length >= 2 ? card('What a kWh cost', lineChart({ points: d.months.filter((m) => m.pricePerKwh > 0).map((m, i) => ({ x: i, y: m.pricePerKwh, m: m.month })),
        xFmt: (x) => monthShort(d.months.filter((m) => m.pricePerKwh > 0)[Math.round(x)].month), yFmt: (y) => `${cur}${y.toFixed(2)}`, summary: 'Average price per kWh from the charger, by month', yTitle: `${cur}/kWh` })) : null,
      card('Where the energy came from', ...d.places.map((p) => h('div', { class: 'row' }, h('span', { class: 'grow', style: 'min-width:0' }, p.name, h('div', { class: 'bar', style: 'margin-top:4px' }, h('i', { style: `width:${(p.kwh / d.places[0].kwh) * 100}%;background:var(--${p.where === 'home' ? 'ok' : 'personal'})` }))),
        h('span', { style: 'text-align:right' }, h('b', {}, num(p.kwh, 0, ' kWh')), h('div', { class: 'muted' }, p.cost > 0 ? money(p.cost) : `${p.sessions} sessions`))))),
      card('When you plug in', heatGrid(WEEK, HOURS, d.plugIn, { text: (r, c, v) => `${WEEK[r]} ${hourText(c)}: ${v} session${v === 1 ? '' : 's'} started`, summary: plugTop && plugTop[0] ? `Most often ${WEEK[plugTop[1]]}s around ${hourText(plugTop[2])}` : '' })),
      d.fastest.length ? card('Your fastest charges', h('p', { class: 'muted' }, 'Tap one to see its charging curve.'), ...d.fastest.map((f) => h('a', { class: 'row', href: '#/charging' },
        h('span', { class: 'grow' }, h('b', {}, num(f.peakKw, 0, ' kW')), h('div', { class: 'muted' }, `${new Date(f.start * 1000).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' })} · ${f.place}`)), h('span', {}, num(f.kwh, 1, ' kWh'))))) : null);
  });
  body.append(out);
}

export function driving(body) {
  const out = h('div');
  windowPicker(body, async (days) => {
    const d = await api(`/api/analytics/driving?days=${days}`);
    if (!d.trips) { out.replaceChildren(h('p', { class: 'muted' }, 'No trips in this stretch.')); return; }
    const f = d.tempFit; // km/kWh = a + b·°C
    const at = (c) => (f ? eff(f[0] + f[1] * c) : '—');
    const cold = isMetric() ? -5 : (20 - 32) * 5 / 9, warm = isMetric() ? 20 : (70 - 32) * 5 / 9;
    const lengthLabels = d.lengths.map((b, i) => (b.toKm ? `<${dist(b.toKm, 0).replace(/ (mi|km)/, '')}` : `${dist(d.lengths[i - 1].toKm, 0).replace(/ (mi|km)/, '')}+`));
    const busiest = d.lengths.reduce((a, b, i) => (b.trips > d.lengths[a].trips ? i : a), 0);
    const bestSpeed = d.speed.reduce((a, s) => (!a || s.kmPerKwh > a.kmPerKwh ? s : a), null);
    out.replaceChildren(
      card('Efficiency and the weather', h('p', { class: 'muted' }, 'Every trip by the outside temperature. The dashed line is your own trend.'),
        scatter(d.temp.map((p) => ({ x: p.tempC, y: p.kmPerKwh, size: p.km, text: `${temp(p.tempC)} · ${eff(p.kmPerKwh)} · ${dist(p.km)}`, href: `#/trip/${p.id}` })),
          { xFmt: (x) => temp(x), yFmt: (y) => (eff(y) || '').split(' ')[0], xTitle: 'Outside temperature', yTitle: (eff(1) || '').split(' ').slice(1).join(' '), line: f, summary: f ? `At ${temp(cold)} about ${at(cold)}; at ${temp(warm)} about ${at(warm)}` : 'Tap a dot to read it' })),
      card('Where the miles went', calendarHeat(d.days.map((x) => ({ date: x.date, value: x.km, text: `${dist(x.km, 0)} in ${x.trips} trip${x.trips === 1 ? '' : 's'}` })),
        { weeks: days === '90' ? 14 : 26, color: 'var(--warn)', summary: `${d.days.length} days with driving · ${dist(d.km, 0)}` })),
      d.speed.length ? card('Efficiency by average speed', ...d.speed.map((s) => h('div', { class: 'row' }, h('span', { style: 'width:6.5em;flex:0 0 auto' }, `${speedRange(s)}`),
        h('div', { class: 'bar grow' }, h('i', { style: `width:${(s.kmPerKwh / bestSpeed.kmPerKwh) * 100}%;background:var(--ok)` })), h('b', { style: 'flex:0 0 auto' }, eff(s.kmPerKwh).split(' ')[0])))) : null,
      card('How long your trips are', stackBars(d.lengths.map((b, i) => ({ label: lengthLabels[i], parts: [b.trips], text: `${b.trips} trips` })), { classes: ['cost'], legend: [`Trips, by ${isMetric() ? 'km' : 'miles'}`], summary: `Most trips are ${lengthLabels[busiest]} ${isMetric() ? 'km' : 'miles'}` })),
      card('When you drive', heatGrid(WEEK, HOURS, d.hourWeek, { color: 'var(--warn)', text: (r, c, v) => `${WEEK[r]} ${hourText(c)}: ${dist(v, 0)} driven`, summary: 'Tap a square' })));
  });
  body.append(out);
}

const speedRange = (s) => {
  const u = (k) => Math.round(isMetric() ? k : k * 0.621371);
  return s.toKph >= 200 ? `${u(s.fromKph)}+ ${isMetric() ? 'km/h' : 'mph'}` : `${u(s.fromKph)}–${u(s.toKph)}`;
};
