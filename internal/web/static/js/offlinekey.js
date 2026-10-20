// Screens ask for "the last 30 days" with exact timestamps that differ on every visit, so a saved answer would never be found again
// with no signal. For such a window this gives a second key built from its length in days, which does not change between visits.
export function windowKey(prefix, path, nowMs = Date.now()) {
  let u;
  try { u = new URL(path, 'http://x'); } catch (e) { return null; }
  const from = Number(u.searchParams.get('from')), to = u.searchParams.has('to') ? Number(u.searchParams.get('to')) : nowMs / 1000;
  if (!(from > 1e9) || !(to > 1e9)) return null; // not a time window (a percentage, a plain date)
  u.searchParams.delete('from');
  u.searchParams.delete('to');
  return `${prefix}${u.pathname}${u.search}#${Math.max(0, Math.round((to - from) / 86400))}d`;
}
