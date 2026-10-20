import { h } from './ui.js';
import { api } from './api.js';

// Seconds left on a remote start, counted down from what the car last said and when it said it. 0 when none is running.
export function remoteLeft(x, now = Date.now() / 1000) {
  if (!x || !(x.remoteStartSec > 0) || x.remoteStartTs == null) return 0;
  return Math.max(0, x.remoteStartSec - (now - x.remoteStartTs));
}

const clock = (s) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, '0')}`;

// A chip saying a remote start is running and how long it has left, ticking each second. null when none is running.
// The countdown is the car's own, as of its last report, so it can drift a little if the car has not reported since.
export function remoteChip(x) {
  if (remoteLeft(x) <= 0) return null;
  const el = h('span', { class: 'chip personal', role: 'status', title: 'The car is running from a remote start and will shut off by itself.' });
  let seen = false;
  const paint = () => {
    const left = remoteLeft(x);
    if (el.isConnected) seen = true;
    if (left <= 0 || (seen && !el.isConnected)) { clearInterval(timer); if (left <= 0) el.textContent = 'Remote start ended'; return; }
    el.textContent = `Remote start · ${clock(left)} left`;
  };
  const timer = setInterval(paint, 1000);
  paint();
  api('/api/remote-starts').then((d) => { if (d && d.sinceDrive > 0) el.title += ` ${d.sinceDrive} remote start${d.sinceDrive === 1 ? '' : 's'} since the car was last driven. The car allows only two in a row.`; }).catch(() => {});
  return el;
}
