import { h, isMetric } from './ui.js';
import { api } from './api.js';
import { effSpan } from './energy.js';

const MI = 0.621371;
const label = (b) => {
  const f = (k) => Math.round(isMetric() ? k : k * MI);
  const u = isMetric() ? 'km/h' : 'mph';
  return b.hiKph >= 300 ? `${f(b.loKph)}+ ${u}` : `${f(b.loKph)}–${f(b.hiKph)} ${u}`;
};

// How much energy the car uses at each speed, from your own drives, and what a faster cruise costs.
export async function speedCard() {
  const d = await api('/api/speed-efficiency').catch(() => null);
  if (!d || !d.bands.length) return null;
  const best = Math.max(...d.bands.map((b) => b.kmPerKwh));
  const rows = d.bands.map((b) => h('div', { style: 'padding:6px 0' },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('span', { class: 'grow' }, label(b), ' ', h('span', { class: 'muted' }, `${Math.round(isMetric() ? b.km : b.km * MI)} ${isMetric() ? 'km' : 'mi'} driven`)), h('b', {}, effSpan(b.km, b.kwh))),
    h('div', { class: 'bar' }, h('i', { style: `width:${Math.max(4, Math.round((b.kmPerKwh / best) * 100))}%` }))));
  let line = null;
  const big = d.bands.filter((b) => b.km >= 15);
  if (big.length >= 2) {
    const a = big[big.length - 2], z = big[big.length - 1];
    const more = Math.round((a.kmPerKwh / z.kmPerKwh - 1) * 100);
    if (more >= 3) line = h('p', {}, `Driving ${label(z)} used about ${more}% more energy per ${isMetric() ? 'km' : 'mile'} than ${label(a)}.`);
  }
  return h('section', { class: 'card' }, h('h2', {}, 'By speed'),
    h('p', { class: 'muted' }, `From your drives in the last ${d.days} days. Wind, hills and temperature mix in, but the pattern is clear.`), ...rows, line);
}
