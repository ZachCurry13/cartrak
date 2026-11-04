// "Do my winter tires change my range?" Trips on winter tires and on other tires are compared only at the same outside temperature and
// speed, so the cold itself is not blamed on the tires. With too little overlap it says what is missing instead of giving a number.
import { h, dist, temp } from './ui.js';
import { api } from './api.js';
import { efficiency } from './energy.js';

const verdictText = { fair: 'a fair amount of driving', good: 'plenty of driving' };

function headline(r) {
  const more = r.pct >= 0;
  const range = Math.abs((1 / (1 + r.pct / 100) - 1) * 100);
  return [h('p', { style: 'font-size:var(--t-title-m);margin:0 0 4px' }, h('b', {}, `Winter tires use about ${Math.abs(r.pct).toFixed(1)}% ${more ? 'more' : 'less'} energy`), ' at the same temperature.'),
    h('p', {}, `That is about ${range.toFixed(1)}% ${more ? 'less' : 'more'} range on a full charge.`,
      r.matched > 1 ? h('span', { class: 'muted' }, ` Across the temperatures compared it ranges from ${r.lowPct.toFixed(1)}% to ${r.highPct.toFixed(1)}%.`) : null)];
}

function cellRow(c) {
  const side = (s) => (s.km > 0 ? `${efficiency(s.km, s.kwh) || '—'} (${dist(s.km, 0)})` : 'none yet');
  return h('div', { class: 'row', style: c.matched ? '' : 'opacity:.6' },
    h('div', { class: 'grow' }, h('b', {}, `${temp(c.loC)} to ${temp(c.hiC)}`), h('span', { class: 'muted' }, ` · ${c.highway ? 'highway' : 'city'}`),
      h('div', { class: 'muted' }, `Other tires ${side(c.other)} · winter ${side(c.winter)}`)),
    c.matched ? h('b', { class: c.pct > 0 ? 'bad' : '' }, `${c.pct > 0 ? '+' : ''}${c.pct.toFixed(1)}%`) : h('span', { class: 'muted' }, 'one side only'));
}

export async function tireCompareCard() {
  const d = await api('/api/tire-sets/compare').catch(() => null);
  if (!d) return null;
  const r = d.result;
  const body = [];
  if (!d.hasWinter && !d.hasOther) body.push(h('p', { class: 'muted' }, 'Add the tires you have on now (and the date they went on) to start. When the winter tires go on, add them too and CarTrak starts comparing.'));
  else if (!d.hasWinter || !d.hasOther) body.push(h('p', { class: 'muted' }, d.hasWinter ? 'Add the tires you drove on before the winter set, with the date they went on, so there is something to compare with.' : 'Log the day the winter tires go on, and CarTrak starts comparing.'));
  else if (r.verdict === 'none') {
    body.push(h('p', {}, h('b', {}, 'Not enough overlap yet. '), `Winter tires have ${dist(r.winterKm, 0)} of trips and the other tires ${dist(r.otherKm, 0)}, but not enough at the same temperature and speed to compare fairly. `,
      'It needs about 20 miles on each at the same temperature (a band about 9\u00b0F wide), and about 100 miles in all. It will not blame the tires for the cold.'));
  } else body.push(...headline(r), Math.abs(r.pct) > 15 ? h('p', { class: 'banner' }, 'That is far more than tires usually change efficiency. Check the dates you logged, and think about what else differed: a heavy load, a roof box, a different route or very different driving.') : null, h('p', { class: 'muted' }, `Based on ${dist(r.matchedKm, 0)} of matched driving: ${verdictText[r.verdict]}.${r.verdict === 'fair' ? ' It firms up as both sets cover more temperatures.' : ''}`));
  if (r.cells.length && d.hasWinter && d.hasOther) body.push(h('details', { class: 'curvebox', open: r.verdict === 'none' }, h('summary', { class: 'muted' }, `By temperature (${r.matched} compared)`), ...r.cells.map(cellRow)));
  body.push(h('p', { class: 'muted' }, `Trips are compared in groups of the same outside temperature and the same kind of driving (city or highway). Trips on wet or snowy roads are left out${r.excluded ? ` (${r.excluded} so far)` : ''}, because winter tires meet more of them. Tire pressure and wind still add some noise.`));
  return h('section', { class: 'card' }, h('h2', {}, 'Do the winter tires change your range?'), ...body);
}
