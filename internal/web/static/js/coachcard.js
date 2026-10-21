import { h } from './ui.js';
import { api } from './api.js';

// Tips from your charging: a few pieces of advice worked out from the last two months, each shown only when the numbers clearly
// back it up. Nothing shows when there is nothing worth saying.
export async function coachCard() {
  const d = await api('/api/analytics/coach').catch(() => null);
  if (!d || !(d.tips || []).length) return null;
  return h('section', { class: 'card' }, h('h2', {}, 'Tips from your charging'),
    ...d.tips.map((t) => h('div', { class: 'rec' }, h('span', { class: 'ri', 'aria-hidden': 'true' }, t.emoji), h('span', { class: 'grow' }, h('b', {}, t.title), h('span', { class: 'muted' }, t.body)))),
    h('p', { class: 'muted', style: 'margin-top:6px' }, `From your last ${d.days} days of charging (${d.sessions} sessions).`));
}
