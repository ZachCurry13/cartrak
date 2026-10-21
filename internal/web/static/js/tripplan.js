// A road-trip plan from three numbers: how far the battery goes per percent (from your own driving),
// how big the battery is, and how fast you charge. It makes no map route: it says how many charging
// stops a distance needs and about how far along each one is.

// All distances are kilometers. Percentages are of the dash battery. Charging power is kW.
export function planTrip({ distanceKm, startPct, reservePct, targetPct, kmPerPct, capacityKwh, chargeKw }) {
  const out = { stops: [], arrivePct: null, extraMinutes: 0, ok: true, reason: '' };
  if (!(kmPerPct > 0)) return { ...out, ok: false, reason: 'CarTrak does not know yet how far your battery goes.' };
  if (!(distanceKm > 0)) return { ...out, ok: false, reason: 'Enter a trip distance to see the plan.' };
  if (!(targetPct > reservePct)) return { ...out, ok: false, reason: 'The charge target must be higher than the reserve.' };
  const firstLeg = Math.max(0, startPct - reservePct) * kmPerPct;
  const kwhPerPct = capacityKwh > 0 ? capacityKwh / 100 : 0;
  const minutesFor = (pct) => (kwhPerPct > 0 && chargeKw > 0 ? Math.round((pct * kwhPerPct / chargeKw) * 60) : null);

  if (distanceKm <= firstLeg) {
    out.arrivePct = startPct - distanceKm / kmPerPct;
    return out;
  }
  let covered = firstLeg;
  let at = firstLeg;
  while (covered < distanceKm && out.stops.length < 40) {
    const remaining = distanceKm - covered;
    // Charge only as much as the rest of the trip needs, up to the target.
    const needPct = Math.min(targetPct, reservePct + remaining / kmPerPct);
    const chargePct = needPct - reservePct;
    const minutes = minutesFor(chargePct);
    out.stops.push({ atKm: at, arrivePct: reservePct, chargeToPct: needPct, minutes });
    if (minutes) out.extraMinutes += minutes;
    covered += chargePct * kmPerPct;
    at = covered;
  }
  out.arrivePct = reservePct + Math.max(0, (covered - distanceKm)) / kmPerPct;
  if (out.stops.length >= 40) return { ...out, ok: false, reason: 'That trip needs more stops than make sense. Check the numbers.' };
  return out;
}
