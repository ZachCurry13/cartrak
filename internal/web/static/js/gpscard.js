import { h } from './ui.js';
import { gps } from './gpstoggle.js';
import { keepAwake, setKeepAwake } from './gpslog.js';

// The "Record GPS from this phone" card on the Live page. Returns the element and a stop function.
export function gpsCard() {
  const msg = h('p', { class: 'muted', role: 'status' });
  const btn = h('button', { class: 'sec' }, 'Record GPS from this phone');
  const unsubscribe = gps.subscribe((s) => {
    btn.textContent = s.on ? 'Stop recording GPS' : 'Record GPS from this phone';
    msg.textContent = s.error ? s.error : s.on ? `Recording. ${s.sent} points sent${s.queued ? `, ${s.queued} waiting to send` : ''}.` : '';
  });
  btn.onclick = () => gps.toggle();
  const awake = h('input', { type: 'checkbox', class: 'check', id: 'gps-awake', checked: keepAwake() });
  awake.onchange = () => setKeepAwake(awake.checked);
  const awakeRow = h('label', { for: 'gps-awake', style: 'display:flex;gap:10px;align-items:center;min-height:44px' }, awake, 'Keep the screen on while recording (uses more battery, but the phone keeps sending its position)');
  const el = h('section', { class: 'card' }, h('h2', {}, 'GPS from this phone'),
    h('p', { class: 'muted' }, 'While this page stays open with the screen on, CarTrak records the location of this phone and adds it to the trip route. A browser cannot record in the background; for that, set up an automation app under More, Phone tracking.'),
    h('p', {}, btn), awakeRow, msg);
  return { el, stop: unsubscribe }; // recording carries on after leaving the page; the top bar button stops it
}
