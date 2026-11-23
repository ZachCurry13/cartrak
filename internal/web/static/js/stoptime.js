import { h } from './ui.js';
import { icon } from './navlayout.js';

// A stop's set time ("be here at 12:00") and how long is spent there, for the route planner. A stop keeps them as at ("12:00", or
// "" for none) and stay (minutes, or null for the planner's time at each stop).

const KEY = 'cartrak.stayMin';
export function defaultStay() {
  try { const v = Number(localStorage.getItem(KEY)); return localStorage.getItem(KEY) != null && v >= 0 && v <= 600 ? v : 30; } catch { return 30; }
}
export function keepDefaultStay(v) { try { localStorage.setItem(KEY, String(v)); } catch { /* storage blocked */ } }

const pretty = (hhmm) => { const [hr, mi] = hhmm.split(':').map(Number); return new Date(2000, 0, 1, hr, mi).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }); };

// What a stop's card says about its time: "Be here at 12:00 PM · 60 min", or null.
export function stopTimeText(s) {
  const bits = [s.at ? `Be here at ${pretty(s.at)}` : null, s.stay != null ? `${s.stay} min here` : null].filter(Boolean);
  return bits.length ? bits.join(' · ') : null;
}

// The moment a stop's set time falls on, on the day of leaving (unix seconds), or 0. A time before leaving is taken as the next day.
export function atTs(s, leaving) {
  if (!s.at) return 0;
  const [hr, mi] = s.at.split(':').map(Number);
  const d = new Date(leaving.getTime());
  d.setHours(hr, mi, 0, 0);
  if (d < leaving) d.setDate(d.getDate() + 1);
  return Math.floor(d.getTime() / 1000);
}

// The clock button and the small form it opens under a stop. done() redraws the stops.
export function stopTimeEditor(s, done) {
  const at = h('input', { type: 'time', value: s.at || '', 'aria-label': 'Be here at', style: 'min-height:44px' });
  const stay = h('input', { type: 'number', min: 0, max: 600, step: 5, value: s.stay ?? '', placeholder: String(defaultStay()), 'aria-label': 'Minutes here', style: 'width:6em;min-height:44px' });
  const form = h('div', { hidden: true, style: 'margin-top:8px' },
    h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap;align-items:center' },
      h('label', { style: 'display:flex;gap:6px;align-items:center;white-space:nowrap' }, 'Be here at', at),
      h('label', { style: 'display:flex;gap:6px;align-items:center;white-space:nowrap' }, 'Minutes here', stay)),
    h('p', { class: 'muted', style: 'margin:6px 0' }, 'A stop with a set time keeps it when Best order rearranges the others, and the plan waits when you would be early.'),
    h('div', { class: 'seg' },
      h('button', { type: 'button', onclick: () => { s.at = at.value || ''; s.stay = stay.value === '' ? null : Math.max(0, Math.min(600, Number(stay.value))); done(); } }, 'Keep'),
      h('button', { class: 'text sm', type: 'button', onclick: () => { s.at = ''; s.stay = null; done(); } }, 'Clear')));
  const button = h('button', { class: 'sec', type: 'button', 'aria-label': 'Set a time or how long here', title: 'Set a time or how long here', style: 'flex:0 0 44px;min-width:44px;padding:0', onclick: () => { form.hidden = !form.hidden; } }, icon('o12,12,9|M12 7v5l3 3'));
  button.firstChild.setAttribute('style', 'width:22px;height:22px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round');
  return { button, form };
}
