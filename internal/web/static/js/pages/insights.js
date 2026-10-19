import { h, dist, temp, when, tempBandLabel } from '../ui.js';
import { api } from '../api.js';
import { effSpan } from '../energy.js';
import { speedCard } from '../speedcard.js';
import { reasonsCard } from '../reasonscard.js';
import { fillTags } from '../tripwhy.js';

const monthName = (m) => { const [y, mo] = m.split('-').map(Number); return new Date(y, mo - 1, 1).toLocaleDateString([], { month: 'short', year: 'numeric' }); };

// How efficiently the car has been driven, and what makes the difference.
export default async function insights(root) {
  const d = await api('/api/insights');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Efficiency insights')));
  if (d.trips < 3) {
    return root.append(h('p', { class: 'muted' }, 'CarTrak needs a few trips with an energy reading (at least 3 miles each) before it can find patterns. Come back after some driving.'));
  }
  const best = Math.max(...d.bands.map((b) => b.kmPerKwh), 0.0001);
  const tripRow = (t) => {
    const tags = h('div', { class: 'tags' }); // the reasons arrive a moment later
    fillTags(tags, t.id);
    return h('a', { class: 'row', href: `#/trip/${t.id}`, style: 'color:inherit;text-decoration:none' },
      h('div', { class: 'grow' }, h('b', {}, effSpan(t.km, t.km / t.kmPerKwh)), h('div', { class: 'muted' }, `${dist(t.km, 1)} · ${when(t.startTs)}${t.tempC != null ? ` · ${temp(t.tempC)}` : ''}`), tags));
  };
  root.append(...[
    h('p', { class: 'muted' }, `Based on ${d.trips} trips from the last year. Cold weather, highway speed and heating use more energy; gentle city driving and mild weather use less.`),
    d.bands.length ? h('section', { class: 'card' }, h('h2', {}, 'By temperature'),
      ...d.bands.map((b) => h('div', { style: 'padding:6px 0' },
        h('div', { class: 'row', style: 'border:0;padding:0' }, h('span', { class: 'grow' }, tempBandLabel(b.name), ' ', h('span', { class: 'muted' }, `${b.trips} trips`)), h('b', {}, effSpan(b.km, b.km / b.kmPerKwh))),
        h('div', { class: 'bar', style: 'margin-top:4px' }, h('i', { style: `width:${Math.round((b.kmPerKwh / best) * 100)}%` }))))) : null,
    h('section', { class: 'card' }, h('h2', {}, 'By month'),
      ...d.months.map((m) => h('div', { class: 'row' }, h('span', { class: 'grow' }, monthName(m.month), ' ', h('span', { class: 'muted' }, `${m.trips} trips${m.avgTempC != null ? ` · ${temp(m.avgTempC)}` : ''}`)), h('b', {}, effSpan(m.km, m.kwh))))),
    d.best.length ? h('section', { class: 'card' }, h('h2', {}, 'Most efficient trips'), ...d.best.map(tripRow)) : null,
    d.worst.length ? h('section', { class: 'card' }, h('h2', {}, 'Least efficient trips'), ...d.worst.map(tripRow)) : null].filter(Boolean));
  const sp = await speedCard();
  const rc = await reasonsCard();
  const first = root.querySelector('section.card');
  for (const c of [rc, sp].filter(Boolean)) { if (first) first.after(c); else root.append(c); } // speed first, then what costs you most, then the rest
}
