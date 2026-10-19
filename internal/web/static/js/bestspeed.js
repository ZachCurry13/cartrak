import { planTrip } from './tripplan.js';

// The ideal cruising speed for a trip: not the most efficient speed, and not the fastest, but the one that gets there
// soonest once charging time is counted. Going faster saves driving time and costs energy, and energy costs charging time.

const mid = (b) => (b.hiKph >= 300 ? b.loKph + 8 : (b.loKph + b.hiKph) / 2);

// Energy per kilometre against speed from your own drives: c(v) = a + b*v*v, fitted to the speed ranges you have driven
// (weighted by distance). It is smooth, so it can say something about speeds close to those you have driven.
export function fitSpeedModel(bands) {
  const use = bands.filter((b) => b.km >= 4 && b.kwh > 0);
  if (use.length < 3) return null;
  let W = 0, Sx = 0, Sy = 0, Sxy = 0, Sxx = 0;
  for (const b of use) {
    const v = mid(b), x = v * v, y = b.kwh / b.km, w = b.km;
    W += w; Sx += w * x; Sy += w * y; Sxy += w * x * y; Sxx += w * x * x;
  }
  let k = (W * Sxy - Sx * Sy) / (W * Sxx - Sx * Sx || 1);
  if (!(k > 1e-9)) k = 1e-9; // energy never falls as speed rises in this model
  const a = (Sy - k * Sx) / W;
  const vTypical = use.reduce((s, b) => s + mid(b) * b.km, 0) / use.reduce((s, b) => s + b.km, 0);
  const lo = Math.min(...use.map(mid)), hi = Math.max(...use.map(mid));
  return { c: (v) => Math.max(0.05, a + k * v * v), vTypical, lo, hi };
}

// Total time for a trip at each cruising speed: driving, charging and a little time lost at each stop.
//   cChosen is the energy per km the planner is using now (so the weather choice carries over).
export function speedTable({ model, distanceKm, startPct, reservePct, targetPct, chargeKw, capacityKwh, cChosen, stopMinutes = 5, speeds }) {
  if (!model || !(distanceKm > 0) || !(capacityKwh > 0) || !(cChosen > 0)) return null;
  const scale = cChosen / model.c(model.vTypical);
  const rows = [];
  for (const kph of speeds) {
    const c = model.c(kph) * scale; // kWh per km at this speed, for these conditions
    const kmPerPct = capacityKwh / 100 / c;
    const plan = planTrip({ distanceKm, startPct, reservePct, targetPct, kmPerPct, capacityKwh, chargeKw });
    if (!plan.ok) continue;
    const drive = (distanceKm / kph) * 60;
    const charge = plan.extraMinutes;
    const stops = plan.stops.length;
    rows.push({ kph, drive, charge, stops, stopTime: stops * stopMinutes, total: drive + charge + stops * stopMinutes, kwh: distanceKm * c,
      estimated: kph < model.lo - 8 || kph > model.hi + 8 });
  }
  if (!rows.length) return null;
  const best = rows.reduce((a, r) => (r.total < a.total ? r : a));
  return { rows, best };
}
