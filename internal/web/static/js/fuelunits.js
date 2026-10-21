import { isMetric } from './ui.js';

// Fuel in the units this device uses: gallons and miles per gallon, or liters and liters per 100 km. CarTrak keeps gallons and the
// price per gallon; the screens convert both ways, so a person who thinks in liters types liters and price per liter.

const L = 3.785411784; // liters in a US gallon
const MI = 0.621371;

export const volUnit = () => (isMetric() ? 'L' : 'gal');
export const toDisplayVol = (gal) => (isMetric() ? gal * L : gal);
export const fromDisplayVol = (v) => (isMetric() ? v / L : v);
export const toDisplayPrice = (perGal) => (isMetric() ? perGal / L : perGal);
export const fromDisplayPrice = (v) => (isMetric() ? v * L : v);

// one decimal, none once it is a hundred or more
export const volText = (gal, digits) => (gal == null ? '—' : `${toDisplayVol(gal).toFixed(digits ?? (toDisplayVol(gal) >= 100 ? 0 : 1))} ${volUnit()}`);
export const priceText = (cur, perGal) => (perGal == null ? '—' : `${cur}${toDisplayPrice(perGal).toFixed(2)}/${volUnit()}`);

// "38.2 mpg" or "6.2 L/100 km"; null when there is nothing to divide.
export function economy(km, gal) {
  if (!(km > 0) || !(gal > 0)) return null;
  return isMetric() ? `${((gal * L) / km * 100).toFixed(1)} L/100 km` : `${((km * MI) / gal).toFixed(1)} mpg`;
}

// the same from a figure already in km per gallon (a record, a month)
export const economyFromKmPerGal = (kmPerGal) => (kmPerGal > 0 ? economy(kmPerGal, 1) : null);
