import { h, ago } from './ui.js';
import { api } from './api.js';
import { copyButton } from './phonehelp.js';

const when = (ts) => new Date(ts * 1000).toLocaleDateString([], { month: 'short', day: 'numeric' });

// Sharing a saved trip: a link anyone can open, with no sign-in, showing the plan as a plain page. It ends on a date and can be stopped.
// By default it leaves out where the trip starts, which is usually home. trip.share is the link already made, if there is one.
export function shareBox(trip, reload) {
  const msg = h('p', { class: 'muted', role: 'status' });
  const days = h('select', { 'aria-label': 'How long the link works' }, ...[[1, '1 day'], [7, '7 days'], [30, '30 days']].map(([v, t]) => h('option', { value: v, selected: v === 7 }, t)));
  const showStart = h('input', { type: 'checkbox', id: `ts-${trip.id}` });
  const make = async () => {
    try { await api(`/api/saved-trips/${trip.id}/share`, { method: 'POST', body: { days: Number(days.value), hideStart: !showStart.checked } }); reload(); } catch (e) { msg.textContent = e.message; }
  };
  const sh = trip.share;
  return h('div', { class: 'card', style: 'margin:6px 0;padding:10px' },
    sh ? h('div', {}, h('p', { class: 'muted' }, `Shared until ${when(sh.expires)}. ${sh.opens ? `Opened ${sh.opens} time${sh.opens === 1 ? '' : 's'}.` : 'Not opened yet.'} ${sh.hideStart ? 'The start is hidden.' : 'The start is shown.'}`),
      h('p', { style: 'overflow-wrap:anywhere' }, sh.url),
      h('p', { class: 'seg' }, copyButton(sh.url, 'Copy the link'), h('button', { class: 'sec', type: 'button', onclick: async () => { await api(`/api/saved-trips/${trip.id}/share`, { method: 'DELETE' }); reload(); } }, 'Stop sharing')))
      : h('p', { class: 'muted' }, 'Anyone with the link can see this plan, with no sign-in, until it ends.'),
    h('label', { for: `ts-${trip.id}`, style: 'display:flex;gap:8px;align-items:center;min-height:44px' }, showStart, 'Show where the trip starts (usually home)'),
    h('label', {}, 'Works for'), days,
    h('p', {}, h('button', { type: 'button', onclick: make }, sh ? 'Make a new link' : 'Make a link')), msg);
}
