import { h } from '../ui.js';
import { api } from '../api.js';
import { badgeRow } from '../funcharts.js';
import { compareEnergyTier, compareDistance } from '../funwords.js';

// Your trophy collection: every trophy this car can earn. The ones not yet earned are grayed out with how far along they are, so
// you can see what is coming. Trophies that cannot apply to this car (charging ones for a gas car, the tank ones for an electric
// car) are not listed at all.

const GROUPS = [
  ['Distance and totals', ['km160', 'km1600', 'km16000', 'world', 'trips100', 'trips1000', 'trips5000', 'month1000', 'road', 'longday', 'marathon', 'odo50', 'odo100', 'odo150']],
  ['Efficiency', ['hyper', 'gentle', 'winter', 'lowslow', 'cold', 'hot']],
  ['Charging', ['kw150', 'quick', 'mwh1', 'mwh10', 'mwh100', 'night50', 'plugstreak', 'skipper', 'detective']],
  ['The tank', ['muncher', 'thrifty']],
  ['Places and habits', ['towns25', 'regular', 'regular50', 'regular100', 'triple', 'round', 'streak7', 'streak30', 'streak100', 'weekend', 'sunrise', 'early', 'nightrider', 'holiday']],
];

// Ladders: trophies that are rungs of one climb. Only the rung you are on is shown in full; the ones you have passed are checked off above it.
// [title, rung keys in order, what a rung is compared with]
const LADDERS = {
  'Distance and totals': [['Distance driven', ['km160', 'km1600', 'km16000', 'world'], 'km'], ['Trips', ['trips100', 'trips1000', 'trips5000']], ['Odometer', ['odo50', 'odo100', 'odo150']]],
  Charging: [['Energy charged', ['mwh1', 'mwh10', 'mwh100'], 'kwh']],
  'Places and habits': [['Driving streak', ['streak7', 'streak30', 'streak100']], ['Regular at one place', ['regular', 'regular50', 'regular100']]],
};

function ladder([title, keys, compare], list) {
  const rungs = keys.map((k) => list.find((b) => b.key === k)).filter(Boolean);
  if (!rungs.length) return null;
  const now = rungs.find((b) => !b.earned) || rungs[rungs.length - 1];
  const done = rungs.filter((b) => b.earned);
  const say = compare === 'kwh' ? `Reaching ${now.name} is enough electricity to run ${compareEnergyTier(now.goal)}.` : compare === 'km' ? `That is about ${compareDistance(now.goal)}.` : null;
  return h('div', { class: 'ladder' }, h('h3', {}, title, ' ', h('span', { class: 'muted' }, `${done.length} of ${rungs.length}`)),
    done.length && now !== done[done.length - 1] ? h('p', {}, ...done.map((b) => h('span', { class: 'chip ok', style: 'margin-right:6px' }, `✓ ${b.name}`))) : null,
    badgeRow(now, list.indexOf(now), '$'), say ? h('p', { class: 'muted' }, say) : null);
}

export default async function trophies(root) {
  root.append(h('p', {}, h('a', { href: '#/analytics' }, '← Insights')), h('div', { class: 'page-title' }, h('h2', {}, 'Trophies')));
  const list = await api('/api/analytics/badges').catch(() => []);
  if (!list.length) return root.append(h('p', { class: 'muted' }, 'Trophies appear once CarTrak has some trips to look at.'));
  const earned = list.filter((b) => b.earned).length;
  const known = new Set(GROUPS.flatMap(([, keys]) => keys));
  const groups = [...GROUPS, ['More', list.filter((b) => !known.has(b.key)).map((b) => b.key)]];
  const next = list.filter((b) => !b.earned && b.progress > 0).sort((a, b) => b.progress - a.progress).slice(0, 3);
  root.append(
    h('section', { class: 'card' }, h('div', { class: 'bigfig' }, h('div', {}, h('b', {}, `${earned} of ${list.length}`), h('span', {}, 'earned'))),
      h('div', { class: 'bar', style: 'margin-top:10px' }, h('i', { style: `width:${Math.round((earned / list.length) * 100)}%` }))),
    next.length ? h('section', { class: 'card' }, h('h2', {}, 'Next up'), ...next.map((b) => badgeRow(b, list.indexOf(b), '$'))) : null,
    ...groups.map(([title, keys]) => {
      const ladders = (LADDERS[title] || []).map((l) => ladder(l, list)).filter(Boolean);
      const stacked = new Set((LADDERS[title] || []).flatMap(([, ks]) => ks));
      const items = keys.filter((k) => !stacked.has(k)).map((k) => list.find((b) => b.key === k)).filter(Boolean).sort((a, b) => (b.earned > 0) - (a.earned > 0) || b.earned - a.earned || b.progress - a.progress);
      if (!items.length && !ladders.length) return null;
      const mine = keys.map((k) => list.find((b) => b.key === k)).filter(Boolean);
      return h('section', { class: 'card' }, h('h2', {}, `${title} · ${mine.filter((b) => b.earned).length} of ${mine.length}`), ...ladders, ...items.map((b) => badgeRow(b, list.indexOf(b), '$')));
    }),
    h('p', { class: 'muted' }, 'Each trophy says what it asks for and where you stand. Efficiency trophies use the odometer’s own distances, so a trip whose distance was corrected never counts toward them.'));
}
