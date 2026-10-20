import { withCar } from './car.js';
// Data mode: how much this device may use. Stored per device, so a phone on a
// limited plan can use Data saver while the desktop stays on Normal or Live.
//   normal : refresh every 30 s, full detail
//   live   : server pushes each new reading (no empty polling), full detail
//   saver  : essentials only, refresh every 5 min, nothing in the background
const KEY = 'dataMode';
export const MODES = { auto: 'Auto', normal: 'Normal', live: 'Live', saver: 'Data saver' };

export function getPref() {
  try { return localStorage.getItem(KEY) || 'auto'; } catch (e) { return 'auto'; }
}
export function setPref(v) {
  try { localStorage.setItem(KEY, v); } catch (e) { /* ignore */ }
}

// Auto follows the phone's own Data Saver setting when the browser exposes it.
export function effective() {
  const p = getPref();
  if (p !== 'auto') return p;
  return navigator.connection && navigator.connection.saveData ? 'saver' : 'normal';
}
export const isSaver = () => effective() === 'saver';

const label = (t) => { const el = document.getElementById('sync'); if (el && t) el.dataset.live = t; };

// Calls load() on a schedule that fits the mode. Returns a function that stops it.
// Nothing runs while the tab is hidden; the live connection is closed then too.
export function startRefresh(load) {
  const mode = effective();
  let timer = null;
  let es = null;
  let stopped = false;
  const run = () => { if (!stopped) load().catch(() => {}); };
  const open = () => {
    if (mode === 'live') {
      if (es || document.hidden) return;
      es = new EventSource(withCar('/api/live'));
      es.addEventListener('reading', run);
      label('live');
    } else if (!timer) {
      timer = setInterval(() => { if (!document.hidden) run(); }, mode === 'saver' ? 300000 : 30000);
    }
  };
  const close = () => {
    if (es) { es.close(); es = null; }
    if (timer) { clearInterval(timer); timer = null; }
  };
  const onVisible = () => {
    if (document.hidden) { if (mode === 'live') close(); return; }
    run();
    open();
  };
  document.addEventListener('visibilitychange', onVisible);
  open();
  return () => { stopped = true; close(); document.removeEventListener('visibilitychange', onVisible); label(''); };
}
