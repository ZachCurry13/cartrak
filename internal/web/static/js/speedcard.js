import { h, isMetric } from './ui.js';
import { api } from './api.js';
import { effSpan } from './energy.js';
import { lineChart } from './chart.js';
import { speedCompare } from './speedcompare.js';

const MI = 0.621371;
const disp = (kph) => (isMetric() ? kph : kph * MI);
const unit = () => (isMetric() ? 'km/h' : 'mph');
const label = (b) => {
  const f = (k) => Math.round(disp(k));
  return b.hiKph >= 300 ? `${f(b.loKph)}+ ${unit()}` : `${f(b.loKph)}–${f(b.hiKph)} ${unit()}`;
};
const mid = (b) => (b.hiKph >= 300 ? b.loKph + 8 : (b.loKph + b.hiKph) / 2);
const effNum = (kmPerKwh) => (isMetric() ? kmPerKwh : kmPerKwh * MI);
const effUnit = () => (isMetric() ? 'km/kWh' : 'mi/kWh');

// How much energy the car uses at each speed, from your own drives: a chart you can scrub, your most efficient speed, and
// what cruising faster costs.
export async function speedCard() {
  const d = await api('/api/speed-efficiency').catch(() => null);
  if (!d || !d.bands.length) return null;
  const solid = d.bands.filter((b) => b.km >= 8); // enough driving to trust
  const best = (solid.length ? solid : d.bands).reduce((a, b) => (b.kmPerKwh > a.kmPerKwh ? b : a));
  const fast = [...solid].reverse().find((b) => b.loKph > best.loKph + 20 && b.km >= 15);
  const sweet = h('p', {}, h('b', {}, `Most efficient: ${label(best)}`), `, about ${effNum(best.kmPerKwh).toFixed(2)} ${effUnit()}.`,
    fast ? ` Cruising at ${label(fast)} used ${Math.round((best.kmPerKwh / fast.kmPerKwh - 1) * 100)}% more energy per ${isMetric() ? 'km' : 'mile'}.` : '');
  const chart = d.bands.length >= 3 ? lineChart({
    points: d.bands.map((b) => ({ x: disp(mid(b)), y: effNum(b.kmPerKwh) })),
    xFmt: (x) => `${Math.round(x)} ${unit()}`,
    yFmt: (v, axis) => (axis ? v.toFixed(1) : `${v.toFixed(2)} ${effUnit()}`),
    xTitle: `Speed (${unit()})`,
    yTitle: `Efficiency (${effUnit()})`,
    yMin: Math.min(...d.bands.map((b) => effNum(b.kmPerKwh))) * 0.7,
    summary: `Best near ${Math.round(disp(mid(best)))} ${unit()}`,
  }) : null;
  const rows = d.bands.map((b) => h('div', { class: 'row' }, h('span', { class: 'grow' }, label(b), ' ', h('span', { class: 'muted' }, `${Math.round(isMetric() ? b.km : b.km * MI)} ${isMetric() ? 'km' : 'mi'}`)), h('b', {}, effSpan(b.km, b.kwh))));
  const compare = await speedCompare(d.bands).catch(() => null);
  return h('section', { class: 'card' }, h('h2', {}, 'By speed'),
    h('p', { class: 'muted' }, `From your drives in the last ${d.days} days. Wind, hills and temperature mix in, so treat it as a guide.`),
    sweet, chart, compare, h('details', {}, h('summary', { class: 'muted' }, 'Every speed range'), ...rows));
}
