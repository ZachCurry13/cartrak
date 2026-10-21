import { h } from '../ui.js';
import { api } from '../api.js';
import { badgeRow } from '../funcharts.js';

// Your trophy collection: every trophy this car can earn. The ones not yet earned are grayed out with how far along they are, so
// you can see what is coming. Trophies that cannot apply to this car (charging ones for a gas car, the tank ones for an electric
// car) are not listed at all.

const GROUPS = [
  ['Distance and totals', ['km160', 'km1600', 'km16000', 'world', 'trips100', 'trips1000', 'month1000', 'road', 'longday', 'marathon', 'odo50', 'odo100', 'odo150']],
  ['Efficiency', ['hyper', 'gentle', 'winter', 'lowslow', 'cold', 'hot']],
  ['Charging', ['kw150', 'quick', 'mwh1', 'mwh10', 'night50', 'plugstreak', 'skipper', 'detective']],
  ['The tank', ['muncher', 'thrifty']],
  ['Places and habits', ['towns25', 'regular', 'triple', 'round', 'streak30', 'weekend', 'sunrise', 'early', 'nightrider', 'holiday']],
];

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
      const items = keys.map((k) => list.find((b) => b.key === k)).filter(Boolean).sort((a, b) => (b.earned > 0) - (a.earned > 0) || b.earned - a.earned || b.progress - a.progress);
      if (!items.length) return null;
      return h('section', { class: 'card' }, h('h2', {}, `${title} · ${items.filter((b) => b.earned).length} of ${items.length}`), ...items.map((b) => badgeRow(b, list.indexOf(b), '$')));
    }),
    h('p', { class: 'muted' }, 'Each trophy says what it asks for and where you stand. Efficiency trophies use the odometer’s own distances, so a trip whose distance was corrected never counts toward them.'));
}
