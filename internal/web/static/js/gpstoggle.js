import { gpsLogger } from './gpslog.js';

// One recorder for the whole app, so the button in the top bar and the card on the Live page are the same switch and recording
// carries on as you move between pages (it still stops when the app is closed or the screen locks: a browser cannot record in the
// background).
let state = { on: false, sent: 0, queued: 0, error: '' };
const listeners = new Set();
const log = gpsLogger((s) => { state = s; listeners.forEach((f) => f(s)); });

export const gps = {
  toggle() { return state.on ? log.stop() : log.start(); },
  subscribe(f) { listeners.add(f); f(state); return () => listeners.delete(f); },
};
