// Where a trip began and ended, in words. A place you named wins, otherwise the city. A drive that stayed inside one city is "Around"
// that city, unless both ends are places you named (then "Home → Gym" says more). Returns the words for the start and the end, and
// whether the whole drive is "around" one place.
export function tripPlaces(t) {
  const startNamed = t.startPlace, endNamed = t.endPlace;
  const from = startNamed || t.startCity || '', end = endNamed || t.endCity || (t.open ? 'on the way' : '');
  if (!t.open && t.startCity && t.startCity === t.endCity && !(startNamed && endNamed && startNamed !== endNamed)) return { from: '', to: t.startCity, around: true };
  if (from && from === end) return { from: '', to: from, around: true };
  return { from, to: end, around: false };
}
