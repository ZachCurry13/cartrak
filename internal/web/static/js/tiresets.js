// The tire sets the owner logged (More, Tires), and which one a moment fell in. Kept for a minute so a page that asks several times asks once.
import { api } from './api.js';

let cache = null, at = 0;

export async function tireSets() {
  if (cache && Date.now() - at < 60000) return cache;
  try { cache = (await api('/api/tire-sets')).sets || []; } catch (e) { cache = []; }
  at = Date.now();
  return cache;
}

export const forgetTireSets = () => { cache = null; };

// The set on at a unix time: the latest that went on at or before it, or null.
export function tireAt(sets, ts) {
  let best = null;
  for (const s of sets) if (s.fromTs <= ts && (!best || s.fromTs >= best.fromTs)) best = s;
  return best;
}

export const KIND_LABEL = { winter: 'Winter', summer: 'Summer', allseason: 'All-season', other: 'Other' };
