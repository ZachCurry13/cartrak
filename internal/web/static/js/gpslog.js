import { api } from './api.js';

// Records this phone's location while the page is open and sends it to CarTrak in batches.
// Browsers cannot track in the background; for that, use an automation app (More, Phone tracking).
// Fixes wait in this browser's storage only until they are sent, so a dropped signal loses nothing.
const KEY = 'cartrak.gpsq';
const MAX_QUEUE = 3000;
const MIN_GAP_MS = 3000; // at most one fix every 3 seconds
const FLUSH_MS = 10000;

const load = () => { try { return JSON.parse(localStorage.getItem(KEY) || '[]'); } catch (e) { return []; } };
const save = (q) => { try { localStorage.setItem(KEY, JSON.stringify(q.slice(-MAX_QUEUE))); } catch (e) { /* storage blocked or full: carry on from memory */ } };

export function gpsLogger(onStatus) {
  let watch = null;
  let timer = null;
  let last = 0;
  let sent = 0;
  let busy = false;
  let error = '';
  let queue = load();
  const report = () => onStatus({ on: watch !== null, sent, queued: queue.length, error });

  async function flush() {
    if (busy || !queue.length) return;
    busy = true;
    const batch = queue.slice(0, 500);
    try {
      const r = await api('/api/track/points', { method: 'POST', body: { points: batch } });
      queue = queue.slice(batch.length);
      save(queue);
      sent += r.saved;
      error = '';
    } catch (e) {
      error = e.message; // offline or the server is away: keep the queue and try again
    }
    busy = false;
    report();
  }

  function onFix(pos) {
    const now = Date.now();
    if (now - last < MIN_GAP_MS) return;
    last = now;
    const c = pos.coords;
    queue.push({ ts: Math.floor(pos.timestamp / 1000), lat: c.latitude, lon: c.longitude, speed: c.speed == null ? undefined : c.speed, acc: c.accuracy });
    save(queue);
    report();
    if (queue.length >= 20) flush();
  }

  const onOnline = () => flush();

  return {
    start() {
      if (watch !== null) return;
      if (!('geolocation' in navigator)) { error = 'This browser cannot provide GPS.'; report(); return; }
      error = '';
      watch = navigator.geolocation.watchPosition(onFix, (e) => {
        error = e.code === 1 ? 'Location permission was refused. Allow it in the browser settings for this site.' : e.message;
        report();
      }, { enableHighAccuracy: true, maximumAge: 2000, timeout: 20000 });
      timer = setInterval(flush, FLUSH_MS);
      window.addEventListener('online', onOnline);
      report();
      flush();
    },
    stop() {
      if (watch !== null) navigator.geolocation.clearWatch(watch);
      watch = null;
      clearInterval(timer);
      window.removeEventListener('online', onOnline);
      report();
      return flush();
    },
  };
}
