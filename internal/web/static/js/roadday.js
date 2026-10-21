import { h, dist, pct, num } from './ui.js';
import { span } from './eventrow.js';

// What the Day view adds for a day with charging on the road:
//  - a small charge at home just before a drive (the car finishing its charge) folds into that drive as "Left home at 98%";
//  - a charge knows the whole stop, from the drive before to the drive after, when you stayed longer than it charged;
//  - the drives between road charges are summed up as legs (how far, how long, what you arrived with) when a leg has more than one
//    drive; a leg of one drive would only repeat its row.
export function roadDay(events) {
  const hidden = new Set(), extra = new Map(), legBefore = new Map();
  const add = (v, more) => extra.set(v, { ...(extra.get(v) || {}), ...more });
  events.forEach((v, i) => {
    const next = events[i + 1];
    if (v.kind === 'charging' && v.where === 'home' && (v.addedKwh || 0) < 2 && next && next.kind === 'driving' && v.end != null && next.start - v.end < 1800) {
      hidden.add(v);
      add(next, { leftNote: `Left home at ${pct(next.startSoc)}${v.addedKwh > 0 ? ` after a ${num(v.addedKwh, 1, ' kWh')} top-up` : ''}` });
    }
  });
  events.forEach((v, i) => {
    if (v.kind !== 'charging') return;
    let prev = null, next = null;
    for (let j = i - 1; j >= 0 && !prev; j--) if (events[j].kind === 'driving') prev = events[j]; else if (events[j].kind === 'charging') break;
    for (let j = i + 1; j < events.length && !next; j++) if (events[j].kind === 'driving') next = events[j]; else if (events[j].kind === 'charging') break;
    if (prev && next && prev.end != null && prev.end <= v.start + 60 && next.start >= (v.end || v.start)) add(v, { stopSec: next.start - prev.end });
  });
  const road = events.filter((v) => v.kind === 'charging' && v.where !== 'home' && !hidden.has(v));
  let lastLeg = null, longestKm = 0;
  const roadChargeSec = road.reduce((a, v) => a + ((v.end || Date.now() / 1000) - v.start), 0);
  if (road.length) {
    let km = 0, sec = 0, soc = null, n = 0, drives = 0;
    const leg = (title) => h('p', { class: 'muted hleg' }, h('b', {}, title), ` ${dist(km, 0)} in ${span(sec)}${soc != null ? `, arrived with ${pct(soc)}` : ''}`);
    for (const v of events) {
      if (hidden.has(v)) continue;
      if (v.kind === 'driving' && v.end != null) {
        km += v.distanceKm || 0;
        drives++;
        sec += v.end - v.start;
        soc = v.endSoc;
      } else if (road.includes(v) && km > 0) {
        longestKm = Math.max(longestKm, km);
        n++;
        if (drives > 1) legBefore.set(v, leg(`Leg ${n}:`));
        km = sec = drives = 0;
        soc = null;
      }
    }
    longestKm = Math.max(longestKm, km);
    if (km > 0 && n > 0 && drives > 1) lastLeg = leg(`Leg ${n + 1}:`);
  }
  return { hidden, extra, legBefore, lastLeg, roadCharges: road.length, roadChargeSec, longestKm };
}
