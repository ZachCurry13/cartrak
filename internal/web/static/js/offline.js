// Remembers the last good answer on this device, so the app can still open and
// show the last readings when there is no signal. Only used for read-only screens.
import { currentCar } from './car.js';

// Remembered answers are kept for each car separately.
const KEY = (k) => `cache:${currentCar()}:${k}`;

export function cacheSet(k, v) {
  try { localStorage.setItem(KEY(k), JSON.stringify({ t: Date.now(), v })); } catch (e) { /* storage full or blocked */ }
}
export function cacheGet(k) {
  try { return JSON.parse(localStorage.getItem(KEY(k))); } catch (e) { return null; }
}

// Runs fn(); on a network failure (not a server error or a sign-out) returns the
// remembered answer instead. staleSince is when that answer was saved, else null.
export async function cachedEx(k, fn) {
  try {
    const value = await fn();
    cacheSet(k, value);
    return { value, staleSince: null };
  } catch (e) {
    if (!(e instanceof TypeError)) throw e; // fetch() rejects with TypeError when offline
    const c = cacheGet(k);
    if (!c) throw e;
    return { value: c.v, staleSince: c.t };
  }
}
export async function cached(k, fn) { return (await cachedEx(k, fn)).value; }
