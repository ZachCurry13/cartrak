// Estimates for road trips and charging. They are only estimates from the latest reading.
// The charging target is a choice for this device, kept in the browser; the reserve is the car's, from its Charging settings.

const clamp = (n, lo, hi) => Math.min(hi, Math.max(lo, n));
const read = (k, d) => { try { const v = localStorage.getItem(`cartrak.${k}`); return v == null ? d : Number(v) || d; } catch (e) { return d; } };
const write = (k, v) => { try { localStorage.setItem(`cartrak.${k}`, String(v)); } catch (e) { /* storage blocked */ } };

export const PRESETS = [80, 90, 100];
let reserve = 20;
export const setCarReserve = (v) => { if (v > 0 && v < 100) reserve = v; };
export const getReserve = () => reserve;
export const getTarget = () => clamp(read('etaTarget', 80), 20, 100);
export const setTarget = (v) => write('etaTarget', clamp(Math.round(v), 20, 100));

// Distance (km) the car can still go before the battery falls to the reserve percent, from the car's
// own range estimate scaled to the battery above the reserve. 0 when already below it, null if unknown.
export function reserveKm(latest, reserve = getReserve()) {
  const soc = latest.socPct;
  const range = latest.rangeKm;
  if (soc == null || range == null || soc <= 0) return null;
  return soc <= reserve ? 0 : (range * (soc - reserve)) / soc;
}

// Minutes until the battery reaches the target percent while charging.
// Returns {done}, {idle} (not charging), {minutes, source} or null when it cannot be worked out.
// Fast chargers slow down a lot above 80 percent, so that is allowed for; AC charging is steady.
export function chargeEta(latest, target = getTarget(), fordMinutesToFull = null) {
  const soc = latest.socPct;
  if (soc == null) return null;
  if (soc >= target) return { done: true };
  const kw = latest.chargeKw;
  if (!(kw > 0.2)) return { idle: true };
  if (target >= 100 && fordMinutesToFull > 0) return { minutes: Math.round(fordMinutesToFull), source: 'ford' };
  const cap = latest.capacityKwh;
  if (!(cap > 0)) return null;
  const perPct = cap / 100;
  const upTo80 = Math.max(0, Math.min(target, 80) - soc) * perPct;
  const above80 = Math.max(0, target - Math.max(soc, 80)) * perPct;
  const hours = kw >= 25 ? upTo80 / kw + above80 / (kw * 0.35) : (upTo80 + above80) / kw;
  return { minutes: Math.round(hours * 60), source: 'estimate' };
}

export function minutesText(m) {
  if (m < 1) return 'under a minute';
  return m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`;
}

export const clockAt = (minutes) => new Date(Date.now() + minutes * 60000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
