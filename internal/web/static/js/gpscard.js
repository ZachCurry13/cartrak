import { h } from './ui.js';
import { gpsLogger } from './gpslog.js';

// The "Record GPS from this phone" card on the Live page. Returns the element and a stop function.
export function gpsCard() {
  const msg = h('p', { class: 'muted', role: 'status' });
  const btn = h('button', { class: 'sec' }, 'Record GPS from this phone');
  const log = gpsLogger((s) => {
    btn.textContent = s.on ? 'Stop recording GPS' : 'Record GPS from this phone';
    msg.textContent = s.error ? s.error : s.on ? `Recording. ${s.sent} points sent${s.queued ? `, ${s.queued} waiting to send` : ''}.` : '';
  });
  let on = false;
  btn.onclick = () => { on = !on; if (on) log.start(); else log.stop(); };
  const el = h('section', { class: 'card' }, h('h2', {}, 'GPS from this phone'),
    h('p', { class: 'muted' }, 'While this page stays open with the screen on, CarTrak records the location of this phone and adds it to the trip route. A browser cannot record in the background; for that, set up an automation app under More, Phone tracking.'),
    h('p', {}, btn), msg);
  return { el, stop: () => { if (on) log.stop(); } };
}
