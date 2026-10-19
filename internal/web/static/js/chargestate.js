// Is the car taking charge? Ford says IN_PROGRESS while charging (a fast charger reports no power at all, so the
// word is the only sign there), older feeds say CHARGING; NOT_READY, PAUSED, FAULT and COMPLETE are not charging.
export function isCharging(x) {
  if ((x.chargeKw || 0) > 0.2) return true;
  const s = String(x.chargeState || '').toUpperCase();
  if (/NOT|PAUSE|COMPLETE|FAULT|STOP/.test(s)) return false;
  return /IN_PROGRESS|CHARGING/.test(s);
}
